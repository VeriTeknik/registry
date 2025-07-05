package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// AnalyticsClient interface for fetching analytics data
type AnalyticsClient interface {
	GetServerMetrics(ctx context.Context, serverID string) (*ServerAnalyticsMetrics, error)
	GetBatchServerMetrics(ctx context.Context, serverIDs []string) (map[string]*ServerAnalyticsMetrics, error)
	GetDashboardMetrics(ctx context.Context, period string) (*DashboardMetrics, error)
	GetRecentActivity(ctx context.Context, limit int) ([]ActivityEvent, error)
}

// ServerAnalyticsMetrics represents metrics from the analytics service for a specific server
type ServerAnalyticsMetrics struct {
	ServerID           string    `json:"server_id"`
	ActiveInstalls     int       `json:"active_installs"`
	DailyActiveUsers   int       `json:"daily_active_users"`
	MonthlyActiveUsers int       `json:"monthly_active_users"`
	WeeklyGrowth       float64   `json:"weekly_growth"`
	LastUpdated        time.Time `json:"last_updated"`
}

// HTTPAnalyticsClient implements AnalyticsClient using HTTP
type HTTPAnalyticsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPAnalyticsClient creates a new HTTP analytics client
func NewHTTPAnalyticsClient(baseURL string) *HTTPAnalyticsClient {
	return &HTTPAnalyticsClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// addAuth adds authentication to the request
func (c *HTTPAnalyticsClient) addAuth(req *http.Request) {
	// Try MCP_REGISTRY_ANALYTICS_USER/PASS first
	if username := os.Getenv("MCP_REGISTRY_ANALYTICS_USER"); username != "" {
		if password := os.Getenv("MCP_REGISTRY_ANALYTICS_PASS"); password != "" {
			req.SetBasicAuth(username, password)
			return
		}
	}
	
	// Fall back to ANALYTICS_API_USERNAME/PASSWORD
	if username := os.Getenv("ANALYTICS_API_USERNAME"); username != "" {
		if password := os.Getenv("ANALYTICS_API_PASSWORD"); password != "" {
			req.SetBasicAuth(username, password)
		}
	}
}

// GetServerMetrics fetches metrics for a single server
func (c *HTTPAnalyticsClient) GetServerMetrics(ctx context.Context, serverID string) (*ServerAnalyticsMetrics, error) {
	url := fmt.Sprintf("%s/servers/%s/stats", c.baseURL, url.PathEscape(serverID))
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		ServerID          string    `json:"server_id"`
		InstallationCount int       `json:"installation_count"`
		Rating            float64   `json:"rating"`
		RatingCount       int       `json:"rating_count"`
		ViewCount         int       `json:"view_count"`
		DailyActiveUsers  int       `json:"daily_active_users"`
		WeeklyGrowthRate  float64   `json:"weekly_growth_rate"`
		LastUpdated       string    `json:"last_updated"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Transform to our internal metrics format
	metrics := &ServerAnalyticsMetrics{
		ServerID:           response.ServerID,
		ActiveInstalls:     response.InstallationCount,
		DailyActiveUsers:   response.DailyActiveUsers,
		MonthlyActiveUsers: response.DailyActiveUsers * 30, // Rough estimate
		WeeklyGrowth:       response.WeeklyGrowthRate,
		LastUpdated:        time.Now(),
	}

	return metrics, nil
}

// GetBatchServerMetrics fetches metrics for multiple servers
func (c *HTTPAnalyticsClient) GetBatchServerMetrics(ctx context.Context, serverIDs []string) (map[string]*ServerAnalyticsMetrics, error) {
	if len(serverIDs) == 0 {
		return make(map[string]*ServerAnalyticsMetrics), nil
	}
	
	// Try batch endpoint first
	url := fmt.Sprintf("%s/servers/stats/batch", c.baseURL)
	
	body, err := json.Marshal(map[string][]string{
		"server_ids": serverIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Fall back to individual requests
		return c.fetchIndividualMetrics(ctx, serverIDs)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		// Fall back to individual requests
		return c.fetchIndividualMetrics(ctx, serverIDs)
	}
	
	var response struct {
		Stats map[string]struct {
			ServerID          string  `json:"server_id"`
			InstallationCount int     `json:"installation_count"`
			Rating            float64 `json:"rating"`
			RatingCount       int     `json:"rating_count"`
			ViewCount         int     `json:"view_count"`
			DailyActiveUsers  int     `json:"daily_active_users"`
			WeeklyGrowthRate  float64 `json:"weekly_growth_rate"`
		} `json:"stats"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Transform to our format
	results := make(map[string]*ServerAnalyticsMetrics)
	for id, stat := range response.Stats {
		results[id] = &ServerAnalyticsMetrics{
			ServerID:           stat.ServerID,
			ActiveInstalls:     stat.InstallationCount,
			DailyActiveUsers:   stat.DailyActiveUsers,
			MonthlyActiveUsers: stat.DailyActiveUsers * 30,
			WeeklyGrowth:       stat.WeeklyGrowthRate,
			LastUpdated:        time.Now(),
		}
	}
	
	return results, nil
}

// fetchIndividualMetrics fetches metrics individually when batch fails
func (c *HTTPAnalyticsClient) fetchIndividualMetrics(ctx context.Context, serverIDs []string) (map[string]*ServerAnalyticsMetrics, error) {
	results := make(map[string]*ServerAnalyticsMetrics)
	
	// Simple sequential fetch - could be made concurrent if needed
	for _, serverID := range serverIDs {
		metrics, err := c.GetServerMetrics(ctx, serverID)
		if err == nil {
			results[serverID] = metrics
		}
	}
	
	return results, nil
}

// GetDashboardMetrics fetches aggregated dashboard metrics
func (c *HTTPAnalyticsClient) GetDashboardMetrics(ctx context.Context, period string) (*DashboardMetrics, error) {
	url := fmt.Sprintf("%s/dashboard?period=%s", c.baseURL, url.QueryEscape(period))
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	c.addAuth(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dashboard metrics: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var metrics DashboardMetrics
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &metrics, nil
}

// GetRecentActivity fetches recent activity events
func (c *HTTPAnalyticsClient) GetRecentActivity(ctx context.Context, limit int) ([]ActivityEvent, error) {
	url := fmt.Sprintf("%s/events/recent?limit=%d", c.baseURL, limit)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	c.addAuth(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch activity: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var response struct {
		Activity []ActivityEvent `json:"activity"`
		Count    int             `json:"count"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response.Activity, nil
}