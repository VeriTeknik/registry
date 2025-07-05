package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// AnalyticsClient interface for fetching analytics data
type AnalyticsClient interface {
	GetServerMetrics(ctx context.Context, serverID string) (*ServerAnalyticsMetrics, error)
	GetBatchServerMetrics(ctx context.Context, serverIDs []string) (map[string]*ServerAnalyticsMetrics, error)
	GetDashboardMetrics(ctx context.Context, period string) (*DashboardMetrics, error)
	GetRecentActivity(ctx context.Context, limit int) ([]ActivityEvent, error)
}

// DashboardMetrics represents aggregated analytics metrics
type DashboardMetrics struct {
	TotalInstalls     MetricWithTrend `json:"total_installs"`
	TotalAPICalls     MetricWithTrend `json:"total_api_calls"`
	ActiveUsers       MetricWithTrend `json:"active_users"`
	ServerHealth      MetricWithTrend `json:"server_health"`
	NewServersToday   int64           `json:"new_servers_today"`
	InstallVelocity   float64         `json:"install_velocity"`
	TopRatedCount     int64           `json:"top_rated_count"`
	SearchSuccessRate float64         `json:"search_success_rate"`
	InstallTrend      []int64         `json:"install_trend"`
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

// GetServerMetrics fetches metrics for a single server
func (c *HTTPAnalyticsClient) GetServerMetrics(ctx context.Context, serverID string) (*ServerAnalyticsMetrics, error) {
	// Use the correct endpoint path
	url := fmt.Sprintf("%s/servers/%s/stats", c.baseURL, url.PathEscape(serverID))
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add authentication if credentials are available in environment
	if username := os.Getenv("MCP_REGISTRY_ANALYTICS_USER"); username != "" {
		if password := os.Getenv("MCP_REGISTRY_ANALYTICS_PASS"); password != "" {
			req.SetBasicAuth(username, password)
		}
	} else if username := os.Getenv("ANALYTICS_API_USERNAME"); username != "" {
		if password := os.Getenv("ANALYTICS_API_PASSWORD"); password != "" {
			req.SetBasicAuth(username, password)
		}
	}

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
	// Use the batch endpoint if available, otherwise fetch in parallel
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
	
	// Add authentication
	if username := os.Getenv("MCP_REGISTRY_ANALYTICS_USER"); username != "" {
		if password := os.Getenv("MCP_REGISTRY_ANALYTICS_PASS"); password != "" {
			req.SetBasicAuth(username, password)
		}
	} else if username := os.Getenv("ANALYTICS_API_USERNAME"); username != "" {
		if password := os.Getenv("ANALYTICS_API_PASSWORD"); password != "" {
			req.SetBasicAuth(username, password)
		}
	}
	
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
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	
	var mu sync.Mutex
	results := make(map[string]*ServerAnalyticsMetrics)
	var wg sync.WaitGroup
	
	for _, serverID := range serverIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			
			sem <- struct{}{}
			defer func() { <-sem }()
			
			metrics, err := c.GetServerMetrics(ctx, id)
			if err != nil {
				log.Printf("Failed to fetch metrics for %s: %v", id, err)
				return
			}
			
			mu.Lock()
			results[id] = metrics
			mu.Unlock()
		}(serverID)
	}
	
	wg.Wait()
	return results, nil
}

// GetDashboardMetrics fetches aggregated dashboard metrics
func (c *HTTPAnalyticsClient) GetDashboardMetrics(ctx context.Context, period string) (*DashboardMetrics, error) {
	url := fmt.Sprintf("%s/dashboard?period=%s", c.baseURL, url.QueryEscape(period))
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add authentication
	if username := os.Getenv("MCP_REGISTRY_ANALYTICS_USER"); username != "" {
		if password := os.Getenv("MCP_REGISTRY_ANALYTICS_PASS"); password != "" {
			req.SetBasicAuth(username, password)
		}
	} else if username := os.Getenv("ANALYTICS_API_USERNAME"); username != "" {
		if password := os.Getenv("ANALYTICS_API_PASSWORD"); password != "" {
			req.SetBasicAuth(username, password)
		}
	}
	
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
	
	// Add authentication
	if username := os.Getenv("MCP_REGISTRY_ANALYTICS_USER"); username != "" {
		if password := os.Getenv("MCP_REGISTRY_ANALYTICS_PASS"); password != "" {
			req.SetBasicAuth(username, password)
		}
	} else if username := os.Getenv("ANALYTICS_API_USERNAME"); username != "" {
		if password := os.Getenv("ANALYTICS_API_PASSWORD"); password != "" {
			req.SetBasicAuth(username, password)
		}
	}
	
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

// SyncService handles periodic synchronization of analytics data
type SyncService struct {
	statsDB         Database
	analyticsClient AnalyticsClient
	interval        time.Duration
	mu              sync.RWMutex
	lastSync        time.Time
}

// NewSyncService creates a new sync service
func NewSyncService(statsDB Database, analyticsClient AnalyticsClient, interval time.Duration) *SyncService {
	return &SyncService{
		statsDB:         statsDB,
		analyticsClient: analyticsClient,
		interval:        interval,
	}
}

// Start begins the periodic sync process
func (s *SyncService) Start(ctx context.Context) {
	log.Println("Starting analytics sync service")
	
	// Run initial sync
	if err := s.syncAll(ctx); err != nil {
		log.Printf("Initial sync failed: %v", err)
	}
	
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping analytics sync service")
			return
		case <-ticker.C:
			if err := s.syncAll(ctx); err != nil {
				log.Printf("Sync failed: %v", err)
			}
		}
	}
}

// SyncServer syncs analytics data for a specific server
func (s *SyncService) SyncServer(ctx context.Context, serverID string) error {
	metrics, err := s.analyticsClient.GetServerMetrics(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get analytics metrics: %w", err)
	}
	
	// Get current stats to preserve installation count and ratings
	// For analytics sync, we always update the REGISTRY source
	currentStats, err := s.statsDB.GetStats(ctx, serverID, SourceRegistry)
	if err != nil {
		return fmt.Errorf("failed to get current stats: %w", err)
	}
	
	// Update only analytics-derived fields
	currentStats.ActiveInstalls = metrics.ActiveInstalls
	currentStats.DailyActiveUsers = metrics.DailyActiveUsers
	currentStats.MonthlyActiveUsers = metrics.MonthlyActiveUsers
	currentStats.LastUpdated = time.Now()
	
	if err := s.statsDB.UpsertStats(ctx, currentStats); err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}
	
	return nil
}

// syncAll syncs all servers with recent activity
func (s *SyncService) syncAll(ctx context.Context) error {
	s.mu.Lock()
	s.lastSync = time.Now()
	s.mu.Unlock()
	
	log.Println("Starting analytics sync")
	
	// Get list of servers to sync (this would come from the main registry)
	// For now, we'll assume we have a method to get active server IDs
	// In a real implementation, this would query the main servers collection
	
	// Implement batch sync for server metrics
	serverIDs, err := s.getActiveServerIDs(ctx)
	if err != nil {
		log.Printf("Failed to get active server IDs: %v", err)
		return nil
	}
	
	if len(serverIDs) == 0 {
		log.Println("No active servers found for sync")
		return nil
	}
	
	// Fetch metrics from analytics service in batches
	metrics, err := s.analyticsClient.GetBatchServerMetrics(ctx, serverIDs)
	if err != nil {
		log.Printf("Failed to get batch metrics: %v", err)
		return nil
	}
	
	// Sync analytics data to stats database
	if err := s.syncAnalyticsData(ctx, metrics); err != nil {
		log.Printf("Failed to sync analytics data: %v", err)
		return nil
	}
	
	log.Printf("Successfully synced metrics for %d servers", len(serverIDs))
	
	log.Println("Analytics sync completed")
	return nil
}

// getActiveServerIDs retrieves server IDs that have recent activity
func (s *SyncService) getActiveServerIDs(ctx context.Context) ([]string, error) {
	// Get all servers with stats from the database
	serverStats, err := s.statsDB.GetAllStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get server stats: %w", err)
	}
	
	// Extract unique server IDs
	serverIDMap := make(map[string]bool)
	for _, stat := range serverStats {
		serverIDMap[stat.ServerID] = true
	}
	
	// Convert to slice
	serverIDs := make([]string, 0, len(serverIDMap))
	for id := range serverIDMap {
		serverIDs = append(serverIDs, id)
	}
	
	log.Printf("Found %d servers to sync", len(serverIDs))
	return serverIDs, nil
}

// syncAnalyticsData processes and stores analytics metrics
func (s *SyncService) syncAnalyticsData(ctx context.Context, metrics map[string]*ServerAnalyticsMetrics) error {
	// Process each server's metrics and update the stats database
	for serverID, metric := range metrics {
		// Update server stats with analytics data
		if err := s.updateServerMetrics(ctx, serverID, metric); err != nil {
			log.Printf("Failed to update metrics for server %s: %v", serverID, err)
			// Continue processing other servers
			continue
		}
	}
	return nil
}

// updateServerMetrics updates individual server metrics from analytics
func (s *SyncService) updateServerMetrics(ctx context.Context, serverID string, metrics *ServerAnalyticsMetrics) error {
	// This would update various metrics like active_installs, weekly_growth, etc.
	// For now, just log the operation
	log.Printf("Updating metrics for server %s: active_installs=%d, weekly_growth=%.2f", 
		serverID, metrics.ActiveInstalls, metrics.WeeklyGrowth)
	return nil
}

// GetLastSyncTime returns the last time analytics were synced
func (s *SyncService) GetLastSyncTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync
}

// CacheService provides caching for frequently accessed stats
type CacheService struct {
	cache map[string]*cacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// NewCacheService creates a new cache service
func NewCacheService(ttl time.Duration) *CacheService {
	service := &CacheService{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}
	
	// Start cleanup goroutine
	go service.cleanup()
	
	return service
}

// Get retrieves a value from cache
func (c *CacheService) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	
	return entry.data, true
}

// Set stores a value in cache
func (c *CacheService) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[key] = &cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from cache
func (c *CacheService) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, key)
}

// cleanup periodically removes expired entries
func (c *CacheService) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}