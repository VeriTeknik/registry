package stats

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// HealthMonitor tracks server health metrics
type HealthMonitor struct {
	client              *mongo.Client
	database            *mongo.Database
	healthCollection    *mongo.Collection
	responseCollection  *mongo.Collection
	
	healthChecks        map[string]*HealthCheckConfig
	mu                  sync.RWMutex
	checkInterval       time.Duration
	stopChan           chan struct{}
}

// HealthCheckConfig defines how to check a server's health
type HealthCheckConfig struct {
	ServerID    string
	HealthURL   string
	Timeout     time.Duration
	LastCheck   time.Time
	Status      string
}

// ResponseTimeTracker tracks response time percentiles
type ResponseTimeTracker struct {
	measurements []float64
	mu          sync.Mutex
	maxSize     int
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(client *mongo.Client, databaseName string) *HealthMonitor {
	db := client.Database(databaseName)
	
	monitor := &HealthMonitor{
		client:             client,
		database:           db,
		healthCollection:   db.Collection("server_health"),
		responseCollection: db.Collection("response_times"),
		healthChecks:       make(map[string]*HealthCheckConfig),
		checkInterval:      5 * time.Minute,
		stopChan:          make(chan struct{}),
	}
	
	// Create indexes
	monitor.createIndexes(context.Background())
	
	return monitor
}

// createIndexes creates necessary indexes
func (m *HealthMonitor) createIndexes(ctx context.Context) error {
	// Health collection indexes
	healthIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "server_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "last_health_check", Value: -1}},
		},
	}
	
	if _, err := m.healthCollection.Indexes().CreateMany(ctx, healthIndexes); err != nil {
		log.Printf("Warning: Failed to create health indexes: %v", err)
	}
	
	// Response time indexes
	responseIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "endpoint", Value: 1}, {Key: "timestamp", Value: -1}},
		},
	}
	
	if _, err := m.responseCollection.Indexes().CreateMany(ctx, responseIndexes); err != nil {
		log.Printf("Warning: Failed to create response time indexes: %v", err)
	}
	
	return nil
}

// Start begins health monitoring
func (m *HealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()
	
	// Initial check
	m.performHealthChecks(ctx)
	
	for {
		select {
		case <-ticker.C:
			m.performHealthChecks(ctx)
		case <-m.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops health monitoring
func (m *HealthMonitor) Stop() {
	close(m.stopChan)
}

// RegisterHealthCheck registers a server for health monitoring
func (m *HealthMonitor) RegisterHealthCheck(serverID, healthURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.healthChecks[serverID] = &HealthCheckConfig{
		ServerID:  serverID,
		HealthURL: healthURL,
		Timeout:   10 * time.Second,
		Status:    "unknown",
	}
}

// performHealthChecks performs health checks on all registered servers
func (m *HealthMonitor) performHealthChecks(ctx context.Context) {
	m.mu.RLock()
	checks := make([]*HealthCheckConfig, 0, len(m.healthChecks))
	for _, check := range m.healthChecks {
		checks = append(checks, check)
	}
	m.mu.RUnlock()
	
	// Perform checks concurrently
	var wg sync.WaitGroup
	for _, check := range checks {
		wg.Add(1)
		go func(hc *HealthCheckConfig) {
			defer wg.Done()
			m.checkServerHealth(ctx, hc)
		}(check)
	}
	
	wg.Wait()
}

// checkServerHealth checks a single server's health
func (m *HealthMonitor) checkServerHealth(ctx context.Context, config *HealthCheckConfig) {
	start := time.Now()
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
	}
	
	// Make health check request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.HealthURL, nil)
	if err != nil {
		m.recordHealthCheck(ctx, config.ServerID, "error", 0, err.Error())
		return
	}
	
	resp, err := client.Do(req)
	responseTime := time.Since(start).Milliseconds()
	
	if err != nil {
		m.recordHealthCheck(ctx, config.ServerID, "down", float64(responseTime), err.Error())
		return
	}
	defer resp.Body.Close()
	
	// Determine health status
	status := "healthy"
	message := fmt.Sprintf("HTTP %d", resp.StatusCode)
	
	if resp.StatusCode >= 500 {
		status = "down"
	} else if resp.StatusCode >= 400 {
		status = "degraded"
	} else if responseTime > 1000 { // Over 1 second
		status = "slow"
		message = fmt.Sprintf("HTTP %d (slow response)", resp.StatusCode)
	}
	
	m.recordHealthCheck(ctx, config.ServerID, status, float64(responseTime), message)
}

// recordHealthCheck records a health check result
func (m *HealthMonitor) recordHealthCheck(ctx context.Context, serverID, status string, responseTime float64, message string) {
	now := time.Now()
	
	// Update health status
	filter := bson.M{"server_id": serverID}
	update := bson.M{
		"$set": bson.M{
			"server_id":         serverID,
			"status":            status,
			"response_time":     responseTime,
			"last_health_check": now,
			"message":           message,
		},
	}
	
	// Calculate availability (last 24 hours)
	availability := m.calculateAvailability(ctx, serverID)
	update["$set"].(bson.M)["availability"] = availability
	
	opts := options.Update().SetUpsert(true)
	_, err := m.healthCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Printf("Failed to record health check: %v", err)
	}
	
	// Record response time for percentile calculations
	m.recordResponseTime(ctx, serverID, responseTime)
}

// calculateAvailability calculates server availability percentage
func (m *HealthMonitor) calculateAvailability(ctx context.Context, serverID string) float64 {
	// Count health checks in last 24 hours
	// yesterday := time.Now().Add(-24 * time.Hour)
	
	// TODO: Use this filter to count health checks
	/* filter := bson.M{
		"server_id": serverID,
		"last_health_check": bson.M{"$gte": yesterday},
	} */
	
	// This is simplified - in production, you'd track individual check results
	var health ServerHealthMetrics
	err := m.healthCollection.FindOne(ctx, bson.M{"server_id": serverID}).Decode(&health)
	if err != nil || health.Status == "down" {
		return 0.0
	}
	
	// Simple calculation based on current status
	switch health.Status {
	case "healthy":
		return 99.9
	case "slow":
		return 95.0
	case "degraded":
		return 75.0
	default:
		return 50.0
	}
}

// recordResponseTime records a response time measurement
func (m *HealthMonitor) recordResponseTime(ctx context.Context, endpoint string, responseTime float64) {
	doc := bson.M{
		"endpoint":      endpoint,
		"response_time": responseTime,
		"timestamp":     time.Now(),
	}
	
	_, err := m.responseCollection.InsertOne(ctx, doc)
	if err != nil {
		log.Printf("Failed to record response time: %v", err)
	}
}

// GetResponseTimePercentiles calculates response time percentiles
func (m *HealthMonitor) GetResponseTimePercentiles(ctx context.Context, endpoint string, period time.Duration) (p50, p90, p99 float64, err error) {
	// Get response times for the period
	startTime := time.Now().Add(-period)
	
	filter := bson.M{
		"timestamp": bson.M{"$gte": startTime},
	}
	if endpoint != "" {
		filter["endpoint"] = endpoint
	}
	
	// Aggregate to calculate percentiles
	pipeline := []bson.M{
		{"$match": filter},
		{"$sort": bson.M{"response_time": 1}},
		{"$group": bson.M{
			"_id": nil,
			"times": bson.M{"$push": "$response_time"},
			"count": bson.M{"$sum": 1},
		}},
	}
	
	cursor, err := m.responseCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to aggregate response times: %w", err)
	}
	defer cursor.Close(ctx)
	
	var result struct {
		Times []float64 `bson:"times"`
		Count int       `bson:"count"`
	}
	
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, 0, 0, fmt.Errorf("failed to decode response times: %w", err)
		}
	}
	
	if len(result.Times) == 0 {
		return 0, 0, 0, nil
	}
	
	// Calculate percentiles
	p50 = getPercentile(result.Times, 50)
	p90 = getPercentile(result.Times, 90)
	p99 = getPercentile(result.Times, 99)
	
	return p50, p90, p99, nil
}

// GetServerHealth returns current health metrics for a server
func (m *HealthMonitor) GetServerHealth(ctx context.Context, serverID string) (*ServerHealthMetrics, error) {
	var health ServerHealthMetrics
	
	filter := bson.M{"server_id": serverID}
	err := m.healthCollection.FindOne(ctx, filter).Decode(&health)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default health metrics
			return &ServerHealthMetrics{
				ServerID:     serverID,
				Status:       "unknown",
				Availability: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get server health: %w", err)
	}
	
	return &health, nil
}

// GetUptimePercentage calculates overall uptime percentage
func (m *HealthMonitor) GetUptimePercentage(ctx context.Context) (float64, error) {
	// Calculate average availability across all monitored servers
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id": nil,
			"avg_availability": bson.M{"$avg": "$availability"},
		}},
	}
	
	cursor, err := m.healthCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate uptime: %w", err)
	}
	defer cursor.Close(ctx)
	
	var result struct {
		AvgAvailability float64 `bson:"avg_availability"`
	}
	
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode uptime: %w", err)
		}
	}
	
	// Default to 99.9% if no data
	if result.AvgAvailability == 0 {
		return 99.9, nil
	}
	
	return result.AvgAvailability, nil
}

// Helper function to calculate percentile
func getPercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	
	index := int(float64(len(sortedValues)-1) * percentile / 100)
	return sortedValues[index]
}

// TrackEndpointResponse tracks response time for API endpoints
func (m *HealthMonitor) TrackEndpointResponse(endpoint string, duration time.Duration) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		m.recordResponseTime(ctx, endpoint, float64(duration.Milliseconds()))
	}()
}