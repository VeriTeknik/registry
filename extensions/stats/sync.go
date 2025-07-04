package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// AnalyticsClient interface for fetching analytics data
type AnalyticsClient interface {
	GetServerMetrics(ctx context.Context, serverID string) (*AnalyticsMetrics, error)
	GetBatchServerMetrics(ctx context.Context, serverIDs []string) (map[string]*AnalyticsMetrics, error)
}

// AnalyticsMetrics represents metrics from the analytics service
type AnalyticsMetrics struct {
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
func (c *HTTPAnalyticsClient) GetServerMetrics(ctx context.Context, serverID string) (*AnalyticsMetrics, error) {
	url := fmt.Sprintf("%s/api/v1/servers/%s/metrics", c.baseURL, url.PathEscape(serverID))
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var metrics AnalyticsMetrics
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &metrics, nil
}

// GetBatchServerMetrics fetches metrics for multiple servers
func (c *HTTPAnalyticsClient) GetBatchServerMetrics(ctx context.Context, serverIDs []string) (map[string]*AnalyticsMetrics, error) {
	// For simplicity, fetch in parallel with limited concurrency
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	
	var mu sync.Mutex
	results := make(map[string]*AnalyticsMetrics)
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
	
	// TODO: Implement batch sync
	// serverIDs := s.getActiveServerIDs(ctx)
	// metrics := s.analyticsClient.GetBatchServerMetrics(ctx, serverIDs)
	// s.statsDB.SyncAnalyticsData(ctx, metrics)
	
	log.Println("Analytics sync completed")
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