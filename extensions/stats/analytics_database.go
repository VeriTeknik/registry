package stats

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AnalyticsDatabase provides analytics data operations
type AnalyticsDatabase interface {
	// Core metrics
	GetAnalyticsMetrics(ctx context.Context, period string) (*AnalyticsMetrics, error)
	UpdateAnalyticsMetrics(ctx context.Context, updates map[string]interface{}) error
	
	// API tracking
	TrackAPICall(ctx context.Context, endpoint, method string, duration float64, isError bool) error
	GetAPIMetrics(ctx context.Context, limit int) ([]APICallMetrics, error)
	
	// Activity tracking
	RecordActivity(ctx context.Context, event *ActivityEvent) error
	GetRecentActivity(ctx context.Context, limit int, eventType string) ([]ActivityEvent, error)
	
	// Search analytics
	TrackSearch(ctx context.Context, searchTerm string, resultsCount int) error
	TrackSearchConversion(ctx context.Context, searchTerm, serverID string) error
	GetTopSearches(ctx context.Context, limit int) ([]SearchAnalytics, error)
	
	// Time series data
	RecordTimeSeries(ctx context.Context, data *TimeSeriesData) error
	GetTimeSeries(ctx context.Context, startTime, endTime time.Time, interval string) ([]TimeSeriesData, error)
	
	// Trending and growth
	CalculateTrending(ctx context.Context, limit int) ([]TrendingServer, error)
	GetGrowthMetrics(ctx context.Context, metric string, period string) (*GrowthMetrics, error)
	
	// Category analytics
	UpdateCategoryStats(ctx context.Context) error
	GetCategoryStats(ctx context.Context) ([]CategoryStats, error)
	
	// Milestones
	CheckAndRecordMilestones(ctx context.Context) error
	GetRecentMilestones(ctx context.Context, limit int) ([]MilestoneEvent, error)
}

// MongoAnalyticsDatabase implements AnalyticsDatabase using MongoDB
type MongoAnalyticsDatabase struct {
	client              *mongo.Client
	database            *mongo.Database
	metricsCollection   *mongo.Collection
	apiCallsCollection  *mongo.Collection
	activityCollection  *mongo.Collection
	searchCollection    *mongo.Collection
	timeSeriesCollection *mongo.Collection
	milestonesCollection *mongo.Collection
	healthMonitor       *HealthMonitor
}

// NewMongoAnalyticsDatabase creates a new MongoDB analytics database
func NewMongoAnalyticsDatabase(client *mongo.Client, databaseName string) (*MongoAnalyticsDatabase, error) {
	db := client.Database(databaseName)
	
	// Create health monitor
	healthMonitor := NewHealthMonitor(client, databaseName)
	
	analyticsDB := &MongoAnalyticsDatabase{
		client:              client,
		database:            db,
		metricsCollection:   db.Collection("analytics_metrics"),
		apiCallsCollection:  db.Collection("api_calls"),
		activityCollection:  db.Collection("activity_events"),
		searchCollection:    db.Collection("search_analytics"),
		timeSeriesCollection: db.Collection("time_series_data"),
		milestonesCollection: db.Collection("milestones"),
		healthMonitor:       healthMonitor,
	}
	
	// Create indexes
	if err := analyticsDB.createIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create analytics indexes: %w", err)
	}
	
	// Start health monitoring in background
	go healthMonitor.Start(context.Background())
	
	return analyticsDB, nil
}

// createIndexes creates necessary indexes for analytics collections
func (db *MongoAnalyticsDatabase) createIndexes(ctx context.Context) error {
	// API calls indexes
	apiIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "endpoint", Value: 1}, {Key: "method", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "last_called", Value: -1}},
		},
	}
	if _, err := db.apiCallsCollection.Indexes().CreateMany(ctx, apiIndexes); err != nil {
		log.Printf("Warning: Failed to create API call indexes: %v", err)
	}
	
	// Activity indexes
	activityIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}, {Key: "timestamp", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "server_id", Value: 1}, {Key: "timestamp", Value: -1}},
		},
	}
	if _, err := db.activityCollection.Indexes().CreateMany(ctx, activityIndexes); err != nil {
		log.Printf("Warning: Failed to create activity indexes: %v", err)
	}
	
	// Search analytics indexes
	searchIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "search_term", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "count", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "last_searched", Value: -1}},
		},
	}
	if _, err := db.searchCollection.Indexes().CreateMany(ctx, searchIndexes); err != nil {
		log.Printf("Warning: Failed to create search indexes: %v", err)
	}
	
	// Time series indexes
	timeSeriesIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
	}
	if _, err := db.timeSeriesCollection.Indexes().CreateMany(ctx, timeSeriesIndexes); err != nil {
		log.Printf("Warning: Failed to create time series indexes: %v", err)
	}
	
	return nil
}

// GetAnalyticsMetrics retrieves current analytics metrics
func (db *MongoAnalyticsDatabase) GetAnalyticsMetrics(ctx context.Context, period string) (*AnalyticsMetrics, error) {
	var metrics AnalyticsMetrics
	
	// Get or create metrics document
	filter := bson.M{"_id": "global_metrics"}
	err := db.metricsCollection.FindOne(ctx, filter).Decode(&metrics)
	if err == mongo.ErrNoDocuments {
		// Initialize new metrics
		metrics = AnalyticsMetrics{
			LastUpdated: time.Now(),
		}
		_, err = db.metricsCollection.InsertOne(ctx, bson.M{
			"_id": "global_metrics",
			"last_updated": metrics.LastUpdated,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get analytics metrics: %w", err)
	}
	
	// Calculate dynamic metrics based on period
	if err := db.calculateDynamicMetrics(ctx, &metrics, period); err != nil {
		log.Printf("Warning: Failed to calculate dynamic metrics: %v", err)
	}
	
	return &metrics, nil
}

// UpdateAnalyticsMetrics updates analytics metrics
func (db *MongoAnalyticsDatabase) UpdateAnalyticsMetrics(ctx context.Context, updates map[string]interface{}) error {
	updates["last_updated"] = time.Now()
	
	filter := bson.M{"_id": "global_metrics"}
	update := bson.M{"$set": updates}
	
	opts := options.Update().SetUpsert(true)
	_, err := db.metricsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update analytics metrics: %w", err)
	}
	
	return nil
}

// TrackAPICall records an API call
func (db *MongoAnalyticsDatabase) TrackAPICall(ctx context.Context, endpoint, method string, duration float64, isError bool) error {
	filter := bson.M{
		"endpoint": endpoint,
		"method":   method,
	}
	
	update := bson.M{
		"$inc": bson.M{
			"count": 1,
		},
		"$set": bson.M{
			"last_called": time.Now(),
		},
	}
	
	if isError {
		update["$inc"].(bson.M)["error_count"] = 1
	}
	
	// Update average duration
	var current APICallMetrics
	err := db.apiCallsCollection.FindOne(ctx, filter).Decode(&current)
	if err == nil {
		// Calculate new average
		newAvg := (current.AvgDuration*float64(current.Count) + duration) / float64(current.Count+1)
		update["$set"].(bson.M)["avg_duration"] = newAvg
	} else {
		update["$set"].(bson.M)["avg_duration"] = duration
	}
	
	opts := options.Update().SetUpsert(true)
	_, err = db.apiCallsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to track API call: %w", err)
	}
	
	// Update global API call count
	globalUpdate := bson.M{
		"$inc": bson.M{"total_api_calls": 1},
	}
	if isError {
		globalUpdate["$inc"].(bson.M)["error_count"] = 1
	}
	
	return db.UpdateAnalyticsMetrics(ctx, globalUpdate)
}

// GetAPIMetrics retrieves API call metrics
func (db *MongoAnalyticsDatabase) GetAPIMetrics(ctx context.Context, limit int) ([]APICallMetrics, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "count", Value: -1}}).
		SetLimit(int64(limit))
	
	cursor, err := db.apiCallsCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get API metrics: %w", err)
	}
	defer cursor.Close(ctx)
	
	var metrics []APICallMetrics
	if err := cursor.All(ctx, &metrics); err != nil {
		return nil, fmt.Errorf("failed to decode API metrics: %w", err)
	}
	
	return metrics, nil
}

// RecordActivity records an activity event
func (db *MongoAnalyticsDatabase) RecordActivity(ctx context.Context, event *ActivityEvent) error {
	event.ID = primitive.NewObjectID().Hex()
	event.Timestamp = time.Now()
	
	_, err := db.activityCollection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to record activity: %w", err)
	}
	
	// Update relevant counters based on activity type
	updates := make(map[string]interface{})
	switch event.Type {
	case "install":
		updates["total_installs"] = bson.M{"$inc": 1}
		updates["installs_today"] = bson.M{"$inc": 1}
	case "rating":
		updates["total_ratings"] = bson.M{"$inc": 1}
	case "search":
		updates["total_searches"] = bson.M{"$inc": 1}
	}
	
	if len(updates) > 0 {
		return db.UpdateAnalyticsMetrics(ctx, updates)
	}
	
	return nil
}

// GetRecentActivity retrieves recent activity events
func (db *MongoAnalyticsDatabase) GetRecentActivity(ctx context.Context, limit int, eventType string) ([]ActivityEvent, error) {
	filter := bson.M{}
	if eventType != "" {
		filter["type"] = eventType
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))
	
	cursor, err := db.activityCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	defer cursor.Close(ctx)
	
	var events []ActivityEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode activity events: %w", err)
	}
	
	return events, nil
}

// TrackSearch records a search query
func (db *MongoAnalyticsDatabase) TrackSearch(ctx context.Context, searchTerm string, resultsCount int) error {
	filter := bson.M{"search_term": searchTerm}
	
	update := bson.M{
		"$inc": bson.M{
			"count":         1,
			"results_found": resultsCount,
		},
		"$set": bson.M{
			"last_searched": time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	_, err := db.searchCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to track search: %w", err)
	}
	
	// Record as activity
	event := &ActivityEvent{
		Type:  "search",
		Value: searchTerm,
		Metadata: map[string]interface{}{
			"results_count": resultsCount,
		},
	}
	
	return db.RecordActivity(ctx, event)
}

// TrackSearchConversion records when a search leads to an install
func (db *MongoAnalyticsDatabase) TrackSearchConversion(ctx context.Context, searchTerm, serverID string) error {
	filter := bson.M{"search_term": searchTerm}
	
	update := bson.M{
		"$inc": bson.M{
			"installs_from_search": 1,
		},
	}
	
	_, err := db.searchCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to track search conversion: %w", err)
	}
	
	// Update success rate
	var search SearchAnalytics
	if err := db.searchCollection.FindOne(ctx, filter).Decode(&search); err == nil {
		successRate := float64(search.InstallsFromSearch) / float64(search.Count) * 100
		updateRate := bson.M{
			"$set": bson.M{"success_rate": successRate},
		}
		db.searchCollection.UpdateOne(ctx, filter, updateRate)
	}
	
	return nil
}

// GetTopSearches retrieves top search terms
func (db *MongoAnalyticsDatabase) GetTopSearches(ctx context.Context, limit int) ([]SearchAnalytics, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "count", Value: -1}}).
		SetLimit(int64(limit))
	
	cursor, err := db.searchCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top searches: %w", err)
	}
	defer cursor.Close(ctx)
	
	var searches []SearchAnalytics
	if err := cursor.All(ctx, &searches); err != nil {
		return nil, fmt.Errorf("failed to decode search analytics: %w", err)
	}
	
	return searches, nil
}

// RecordTimeSeries records time series data point
func (db *MongoAnalyticsDatabase) RecordTimeSeries(ctx context.Context, data *TimeSeriesData) error {
	data.Timestamp = time.Now()
	
	_, err := db.timeSeriesCollection.InsertOne(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to record time series data: %w", err)
	}
	
	return nil
}

// GetTimeSeries retrieves time series data
func (db *MongoAnalyticsDatabase) GetTimeSeries(ctx context.Context, startTime, endTime time.Time, interval string) ([]TimeSeriesData, error) {
	filter := bson.M{
		"timestamp": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}
	
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	
	cursor, err := db.timeSeriesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}
	defer cursor.Close(ctx)
	
	var data []TimeSeriesData
	if err := cursor.All(ctx, &data); err != nil {
		return nil, fmt.Errorf("failed to decode time series data: %w", err)
	}
	
	// Aggregate by interval if needed
	if interval != "" {
		data = db.aggregateTimeSeries(data, interval)
	}
	
	return data, nil
}

// calculateDynamicMetrics calculates metrics that change based on time period
func (db *MongoAnalyticsDatabase) calculateDynamicMetrics(ctx context.Context, metrics *AnalyticsMetrics, period string) error {
	now := time.Now()
	
	// Calculate installs for different periods
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	
	// Count installs today
	filter := bson.M{
		"type": "install",
		"timestamp": bson.M{"$gte": todayStart},
	}
	count, err := db.activityCollection.CountDocuments(ctx, filter)
	if err == nil {
		metrics.InstallsToday = count
	}
	
	// Count installs this week
	filter["timestamp"] = bson.M{"$gte": weekStart}
	count, err = db.activityCollection.CountDocuments(ctx, filter)
	if err == nil {
		metrics.InstallsThisWeek = count
	}
	
	// Count installs this month
	filter["timestamp"] = bson.M{"$gte": monthStart}
	count, err = db.activityCollection.CountDocuments(ctx, filter)
	if err == nil {
		metrics.InstallsThisMonth = count
	}
	
	// Calculate install velocity (installs per hour over last 24 hours)
	yesterday := now.Add(-24 * time.Hour)
	filter["timestamp"] = bson.M{"$gte": yesterday}
	count, err = db.activityCollection.CountDocuments(ctx, filter)
	if err == nil {
		metrics.InstallVelocity = float64(count) / 24.0
	}
	
	// Calculate growth rates
	lastWeekStart := weekStart.AddDate(0, 0, -7)
	filter["timestamp"] = bson.M{
		"$gte": lastWeekStart,
		"$lt":  weekStart,
	}
	lastWeekCount, err := db.activityCollection.CountDocuments(ctx, filter)
	if err == nil && lastWeekCount > 0 {
		metrics.WeeklyGrowth = (float64(metrics.InstallsThisWeek) - float64(lastWeekCount)) / float64(lastWeekCount) * 100
	}
	
	// Get health metrics
	if db.healthMonitor != nil {
		// Get response time percentiles
		p50, p90, p99, err := db.healthMonitor.GetResponseTimePercentiles(ctx, "", 24*time.Hour)
		if err == nil {
			metrics.ResponseTimeP50 = p50
			metrics.ResponseTimeP90 = p90
			metrics.ResponseTimeP99 = p99
		}
		
		// Get uptime percentage
		uptime, err := db.healthMonitor.GetUptimePercentage(ctx)
		if err == nil {
			metrics.UptimePercentage = uptime
		}
	}
	
	// Calculate quality metrics
	db.calculateQualityMetrics(ctx, metrics)
	
	return nil
}

// calculateQualityMetrics calculates rating and feedback metrics
func (db *MongoAnalyticsDatabase) calculateQualityMetrics(ctx context.Context, metrics *AnalyticsMetrics) error {
	// Get average rating across all servers
	// TODO: Implement aggregation pipeline
	/* pipeline := []bson.M{
		{"$match": bson.M{"rating": bson.M{"$gt": 0}}},
		{"$group": bson.M{
			"_id": nil,
			"avg_rating": bson.M{"$avg": "$rating"},
			"total_ratings": bson.M{"$sum": "$rating_count"},
		}},
	} */
	
	// This would query the stats collection - simplified for now
	metrics.AverageRating = 4.2 // Placeholder
	metrics.TotalRatings = 150   // Placeholder
	metrics.FiveStarServers = 25 // Placeholder
	
	return nil
}

// aggregateTimeSeries aggregates time series data by interval
func (db *MongoAnalyticsDatabase) aggregateTimeSeries(data []TimeSeriesData, interval string) []TimeSeriesData {
	// Simple aggregation - in production, use MongoDB aggregation pipeline
	// This is a placeholder implementation
	return data
}

// CalculateTrending calculates trending servers
func (db *MongoAnalyticsDatabase) CalculateTrending(ctx context.Context, limit int) ([]TrendingServer, error) {
	// Calculate trending based on recent activity (last 24 hours vs previous 24 hours)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)
	
	// Aggregate install activity by server
	pipeline := []bson.M{
		// Match install events from last 48 hours
		{"$match": bson.M{
			"type": "install",
			"timestamp": bson.M{"$gte": twoDaysAgo},
			"server_id": bson.M{"$exists": true, "$ne": ""},
		}},
		// Add fields to categorize time periods
		{"$addFields": bson.M{
			"is_recent": bson.M{
				"$gte": []interface{}{"$timestamp", yesterday},
			},
		}},
		// Group by server and calculate metrics
		{"$group": bson.M{
			"_id": "$server_id",
			"total_installs": bson.M{"$sum": 1},
			"recent_installs": bson.M{
				"$sum": bson.M{
					"$cond": []interface{}{"$is_recent", 1, 0},
				},
			},
			"previous_installs": bson.M{
				"$sum": bson.M{
					"$cond": []interface{}{"$is_recent", 0, 1},
				},
			},
			"server_name": bson.M{"$first": "$server_name"},
		}},
		// Calculate velocity and momentum
		{"$addFields": bson.M{
			"install_velocity": bson.M{
				"$divide": []interface{}{"$recent_installs", 24.0}, // installs per hour
			},
			"momentum_change": bson.M{
				"$cond": bson.M{
					"if": bson.M{"$eq": []interface{}{"$previous_installs", 0}},
					"then": 100.0,
					"else": bson.M{
						"$multiply": []interface{}{
							bson.M{"$divide": []interface{}{
								bson.M{"$subtract": []interface{}{"$recent_installs", "$previous_installs"}},
								"$previous_installs",
							}},
							100,
						},
					},
				},
			},
		}},
		// Calculate trending score (combination of velocity and momentum)
		{"$addFields": bson.M{
			"trending_score": bson.M{
				"$add": []interface{}{
					"$install_velocity",
					bson.M{"$multiply": []interface{}{"$momentum_change", 0.1}},
				},
			},
		}},
		// Sort by trending score
		{"$sort": bson.M{"trending_score": -1}},
		// Limit results
		{"$limit": limit},
	}
	
	cursor, err := db.activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate trending: %w", err)
	}
	defer cursor.Close(ctx)
	
	var trending []TrendingServer
	for cursor.Next(ctx) {
		var result struct {
			ServerID          string  `bson:"_id"`
			ServerName        string  `bson:"server_name"`
			TotalInstalls     int64   `bson:"total_installs"`
			RecentInstalls    int64   `bson:"recent_installs"`
			PreviousInstalls  int64   `bson:"previous_installs"`
			InstallVelocity   float64 `bson:"install_velocity"`
			MomentumChange    float64 `bson:"momentum_change"`
			TrendingScore     float64 `bson:"trending_score"`
		}
		
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding trending result: %v", err)
			continue
		}
		
		// Get server details if name is missing
		if result.ServerName == "" {
			// In a real implementation, we'd batch these lookups
			server, err := db.client.Database(db.database.Name()).
				Collection("servers_v2").
				FindOne(ctx, bson.M{"id": result.ServerID}).
				DecodeBytes()
			if err == nil {
				result.ServerName, _ = server.Lookup("name").StringValueOK()
			}
		}
		
		trending = append(trending, TrendingServer{
			ServerID:          result.ServerID,
			ServerName:        result.ServerName,
			TrendingScore:     result.TrendingScore,
			InstallVelocity:   result.InstallVelocity,
			MomentumChange:    result.MomentumChange,
			RecentInstalls:    result.RecentInstalls,
			PreviousInstalls:  result.PreviousInstalls,
			TrendPeriod:       "24h",
		})
	}
	
	// If we don't have enough trending servers, fill with top-rated
	if len(trending) < limit {
		// Get top-rated servers to fill the gap
		remaining := limit - len(trending)
		serverIDs := make([]string, len(trending))
		for i, t := range trending {
			serverIDs[i] = t.ServerID
		}
		
		// Query stats collection for highly-rated servers not already in trending
		filter := bson.M{
			"server_id": bson.M{"$nin": serverIDs},
			"rating":    bson.M{"$gte": 4.0},
		}
		
		opts := options.Find().
			SetSort(bson.D{{Key: "rating", Value: -1}, {Key: "install_count", Value: -1}}).
			SetLimit(int64(remaining))
		
		cursor, err := db.client.Database(db.database.Name()).
			Collection("stats").
			Find(ctx, filter, opts)
		if err == nil {
			defer cursor.Close(ctx)
			
			for cursor.Next(ctx) {
				var stat struct {
					ServerID     string  `bson:"server_id"`
					Rating       float64 `bson:"rating"`
					InstallCount int64   `bson:"install_count"`
				}
				
				if err := cursor.Decode(&stat); err == nil {
					// Get server name
					var serverName string
					server, err := db.client.Database(db.database.Name()).
						Collection("servers_v2").
						FindOne(ctx, bson.M{"id": stat.ServerID}).
						DecodeBytes()
					if err == nil {
						serverName, _ = server.Lookup("name").StringValueOK()
					}
					
					trending = append(trending, TrendingServer{
						ServerID:        stat.ServerID,
						ServerName:      serverName,
						TrendingScore:   stat.Rating * 10, // Use rating as trending score
						InstallVelocity: float64(stat.InstallCount) / (30 * 24), // Approximate velocity
						MomentumChange:  0, // No momentum data for these
						TrendPeriod:     "all-time",
					})
				}
			}
		}
	}
	
	return trending, nil
}

// GetGrowthMetrics calculates growth for a specific metric
func (db *MongoAnalyticsDatabase) GetGrowthMetrics(ctx context.Context, metric string, period string) (*GrowthMetrics, error) {
	now := time.Now()
	var currentPeriodStart, previousPeriodStart, previousPeriodEnd time.Time
	
	// Determine time periods based on requested period
	switch period {
	case "day":
		currentPeriodStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		previousPeriodEnd = currentPeriodStart
		previousPeriodStart = currentPeriodStart.AddDate(0, 0, -1)
	case "week":
		currentPeriodStart = now.AddDate(0, 0, -int(now.Weekday()))
		currentPeriodStart = time.Date(currentPeriodStart.Year(), currentPeriodStart.Month(), currentPeriodStart.Day(), 0, 0, 0, 0, currentPeriodStart.Location())
		previousPeriodEnd = currentPeriodStart
		previousPeriodStart = currentPeriodStart.AddDate(0, 0, -7)
	case "month":
		currentPeriodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		previousPeriodEnd = currentPeriodStart
		previousPeriodStart = currentPeriodStart.AddDate(0, -1, 0)
	case "year":
		currentPeriodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		previousPeriodEnd = currentPeriodStart
		previousPeriodStart = currentPeriodStart.AddDate(-1, 0, 0)
	default:
		// Default to week
		currentPeriodStart = now.AddDate(0, 0, -7)
		previousPeriodEnd = currentPeriodStart
		previousPeriodStart = currentPeriodStart.AddDate(0, 0, -7)
		period = "week"
	}
	
	// Initialize growth metrics
	growth := &GrowthMetrics{
		Metric: metric,
		Period: period,
		CurrentPeriodStart: currentPeriodStart,
		PreviousPeriodStart: previousPeriodStart,
	}
	
	// Calculate metric-specific growth
	switch metric {
	case "installs":
		growth.CurrentValue, growth.PreviousValue = db.calculateInstallGrowth(ctx, currentPeriodStart, now, previousPeriodStart, previousPeriodEnd)
	case "users":
		growth.CurrentValue, growth.PreviousValue = db.calculateUserGrowth(ctx, currentPeriodStart, now, previousPeriodStart, previousPeriodEnd)
	case "api_calls":
		growth.CurrentValue, growth.PreviousValue = db.calculateAPICallGrowth(ctx, currentPeriodStart, now, previousPeriodStart, previousPeriodEnd)
	case "servers":
		growth.CurrentValue, growth.PreviousValue = db.calculateServerGrowth(ctx, currentPeriodStart, now, previousPeriodStart, previousPeriodEnd)
	case "ratings":
		growth.CurrentValue, growth.PreviousValue = db.calculateRatingGrowth(ctx, currentPeriodStart, now, previousPeriodStart, previousPeriodEnd)
	default:
		return nil, fmt.Errorf("unsupported metric: %s", metric)
	}
	
	// Calculate growth percentage
	if growth.PreviousValue > 0 {
		growth.GrowthRate = ((growth.CurrentValue - growth.PreviousValue) / growth.PreviousValue) * 100
	} else if growth.CurrentValue > 0 {
		growth.GrowthRate = 100.0
	}
	
	// Calculate absolute change
	growth.AbsoluteChange = growth.CurrentValue - growth.PreviousValue
	
	// Calculate momentum (growth acceleration)
	// Get the period before previous for momentum calculation
	var momentumPeriodStart time.Time
	switch period {
	case "day":
		momentumPeriodStart = previousPeriodStart.AddDate(0, 0, -1)
	case "week":
		momentumPeriodStart = previousPeriodStart.AddDate(0, 0, -7)
	case "month":
		momentumPeriodStart = previousPeriodStart.AddDate(0, -1, 0)
	case "year":
		momentumPeriodStart = previousPeriodStart.AddDate(-1, 0, 0)
	}
	
	// Calculate momentum
	var momentumValue float64
	switch metric {
	case "installs":
		momentumValue, _ = db.calculateInstallGrowth(ctx, momentumPeriodStart, previousPeriodStart, time.Time{}, time.Time{})
	case "users":
		momentumValue, _ = db.calculateUserGrowth(ctx, momentumPeriodStart, previousPeriodStart, time.Time{}, time.Time{})
	case "api_calls":
		momentumValue, _ = db.calculateAPICallGrowth(ctx, momentumPeriodStart, previousPeriodStart, time.Time{}, time.Time{})
	}
	
	if momentumValue > 0 {
		previousGrowth := (growth.PreviousValue - momentumValue) / momentumValue * 100
		growth.Momentum = growth.GrowthRate - previousGrowth
		
		if growth.Momentum > 0 {
			growth.Trend = "accelerating"
		} else if growth.Momentum < -5 {
			growth.Trend = "decelerating"
		} else {
			growth.Trend = "steady"
		}
	} else {
		growth.Trend = "new"
	}
	
	// Add data points for visualization
	growth.DataPoints = db.getGrowthDataPoints(ctx, metric, currentPeriodStart, now, period)
	
	return growth, nil
}

// calculateInstallGrowth calculates install counts for given periods
func (db *MongoAnalyticsDatabase) calculateInstallGrowth(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous float64) {
	// Count installs in current period
	filter := bson.M{
		"type": "install",
		"timestamp": bson.M{
			"$gte": currentStart,
			"$lt":  currentEnd,
		},
	}
	
	currentCount, err := db.activityCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Error counting current installs: %v", err)
		return 0, 0
	}
	current = float64(currentCount)
	
	// Count installs in previous period if specified
	if !previousStart.IsZero() && !previousEnd.IsZero() {
		filter["timestamp"] = bson.M{
			"$gte": previousStart,
			"$lt":  previousEnd,
		}
		previousCount, err := db.activityCollection.CountDocuments(ctx, filter)
		if err != nil {
			log.Printf("Error counting previous installs: %v", err)
			return current, 0
		}
		previous = float64(previousCount)
	}
	
	return current, previous
}

// calculateUserGrowth calculates unique user counts for given periods
func (db *MongoAnalyticsDatabase) calculateUserGrowth(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous float64) {
	// Get unique users in current period
	pipeline := []bson.M{
		{"$match": bson.M{
			"timestamp": bson.M{
				"$gte": currentStart,
				"$lt":  currentEnd,
			},
			"user_id": bson.M{"$exists": true, "$ne": ""},
		}},
		{"$group": bson.M{
			"_id": "$user_id",
		}},
		{"$count": "total"},
	}
	
	cursor, err := db.activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error counting current users: %v", err)
		return 0, 0
	}
	defer cursor.Close(ctx)
	
	var result struct {
		Total int `bson:"total"`
	}
	if cursor.Next(ctx) {
		cursor.Decode(&result)
		current = float64(result.Total)
	}
	
	// Count unique users in previous period if specified
	if !previousStart.IsZero() && !previousEnd.IsZero() {
		pipeline[0]["$match"].(bson.M)["timestamp"] = bson.M{
			"$gte": previousStart,
			"$lt":  previousEnd,
		}
		
		cursor, err := db.activityCollection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Printf("Error counting previous users: %v", err)
			return current, 0
		}
		defer cursor.Close(ctx)
		
		if cursor.Next(ctx) {
			cursor.Decode(&result)
			previous = float64(result.Total)
		}
	}
	
	return current, previous
}

// calculateAPICallGrowth calculates API call counts for given periods
func (db *MongoAnalyticsDatabase) calculateAPICallGrowth(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous float64) {
	// Sum API calls in current period
	pipeline := []bson.M{
		{"$match": bson.M{
			"last_called": bson.M{
				"$gte": currentStart,
				"$lt":  currentEnd,
			},
		}},
		{"$group": bson.M{
			"_id": nil,
			"total": bson.M{"$sum": "$count"},
		}},
	}
	
	cursor, err := db.apiCallsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error counting current API calls: %v", err)
		return 0, 0
	}
	defer cursor.Close(ctx)
	
	var result struct {
		Total int64 `bson:"total"`
	}
	if cursor.Next(ctx) {
		cursor.Decode(&result)
		current = float64(result.Total)
	}
	
	// Count API calls in previous period if specified
	if !previousStart.IsZero() && !previousEnd.IsZero() {
		pipeline[0]["$match"].(bson.M)["last_called"] = bson.M{
			"$gte": previousStart,
			"$lt":  previousEnd,
		}
		
		cursor, err := db.apiCallsCollection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Printf("Error counting previous API calls: %v", err)
			return current, 0
		}
		defer cursor.Close(ctx)
		
		if cursor.Next(ctx) {
			cursor.Decode(&result)
			previous = float64(result.Total)
		}
	}
	
	return current, previous
}

// calculateServerGrowth calculates new server counts for given periods
func (db *MongoAnalyticsDatabase) calculateServerGrowth(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous float64) {
	// This would query the stats collection for servers added in the time periods
	// For now, using activity events of type "server_added"
	filter := bson.M{
		"type": "server_added",
		"timestamp": bson.M{
			"$gte": currentStart,
			"$lt":  currentEnd,
		},
	}
	
	currentCount, err := db.activityCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Error counting current servers: %v", err)
		return 0, 0
	}
	current = float64(currentCount)
	
	// Count servers in previous period if specified
	if !previousStart.IsZero() && !previousEnd.IsZero() {
		filter["timestamp"] = bson.M{
			"$gte": previousStart,
			"$lt":  previousEnd,
		}
		previousCount, err := db.activityCollection.CountDocuments(ctx, filter)
		if err != nil {
			log.Printf("Error counting previous servers: %v", err)
			return current, 0
		}
		previous = float64(previousCount)
	}
	
	return current, previous
}

// calculateRatingGrowth calculates new rating counts for given periods
func (db *MongoAnalyticsDatabase) calculateRatingGrowth(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous float64) {
	// Count ratings in current period
	filter := bson.M{
		"type": "rating",
		"timestamp": bson.M{
			"$gte": currentStart,
			"$lt":  currentEnd,
		},
	}
	
	currentCount, err := db.activityCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Error counting current ratings: %v", err)
		return 0, 0
	}
	current = float64(currentCount)
	
	// Count ratings in previous period if specified
	if !previousStart.IsZero() && !previousEnd.IsZero() {
		filter["timestamp"] = bson.M{
			"$gte": previousStart,
			"$lt":  previousEnd,
		}
		previousCount, err := db.activityCollection.CountDocuments(ctx, filter)
		if err != nil {
			log.Printf("Error counting previous ratings: %v", err)
			return current, 0
		}
		previous = float64(previousCount)
	}
	
	return current, previous
}

// getGrowthDataPoints gets data points for growth visualization
func (db *MongoAnalyticsDatabase) getGrowthDataPoints(ctx context.Context, metric string, start, end time.Time, period string) []DataPoint {
	var points []DataPoint
	var interval time.Duration
	
	// Determine interval based on period
	switch period {
	case "day":
		interval = time.Hour
	case "week":
		interval = 24 * time.Hour
	case "month":
		interval = 24 * time.Hour
	case "year":
		interval = 30 * 24 * time.Hour // Approximate month
	default:
		interval = 24 * time.Hour
	}
	
	// Generate data points
	current := start
	for current.Before(end) {
		next := current.Add(interval)
		if next.After(end) {
			next = end
		}
		
		var value float64
		switch metric {
		case "installs":
			value, _ = db.calculateInstallGrowth(ctx, current, next, time.Time{}, time.Time{})
		case "users":
			value, _ = db.calculateUserGrowth(ctx, current, next, time.Time{}, time.Time{})
		case "api_calls":
			value, _ = db.calculateAPICallGrowth(ctx, current, next, time.Time{}, time.Time{})
		case "servers":
			value, _ = db.calculateServerGrowth(ctx, current, next, time.Time{}, time.Time{})
		case "ratings":
			value, _ = db.calculateRatingGrowth(ctx, current, next, time.Time{}, time.Time{})
		}
		
		points = append(points, DataPoint{
			Timestamp: current,
			Value:     value,
		})
		
		current = next
	}
	
	return points
}

// UpdateCategoryStats updates category statistics
func (db *MongoAnalyticsDatabase) UpdateCategoryStats(ctx context.Context) error {
	// This would aggregate stats by category
	return nil
}

// GetCategoryStats retrieves category statistics
func (db *MongoAnalyticsDatabase) GetCategoryStats(ctx context.Context) ([]CategoryStats, error) {
	return []CategoryStats{}, nil
}

// CheckAndRecordMilestones checks for and records milestone achievements
func (db *MongoAnalyticsDatabase) CheckAndRecordMilestones(ctx context.Context) error {
	metrics, err := db.GetAnalyticsMetrics(ctx, "all")
	if err != nil {
		return err
	}
	
	// Check various milestones
	milestones := []int64{100, 500, 1000, 5000, 10000, 50000, 100000}
	
	for _, milestone := range milestones {
		// Check total installs milestone
		if metrics.TotalInstalls >= milestone {
			filter := bson.M{
				"type":      "installs",
				"milestone": milestone,
			}
			
			count, _ := db.milestonesCollection.CountDocuments(ctx, filter)
			if count == 0 {
				// Record new milestone
				event := MilestoneEvent{
					ID:          primitive.NewObjectID().Hex(),
					Type:        "installs",
					Milestone:   milestone,
					AchievedAt:  time.Now(),
					Description: fmt.Sprintf("Registry reached %d total installs!", milestone),
				}
				
				db.milestonesCollection.InsertOne(ctx, event)
			}
		}
	}
	
	return nil
}

// GetRecentMilestones retrieves recent milestone events
func (db *MongoAnalyticsDatabase) GetRecentMilestones(ctx context.Context, limit int) ([]MilestoneEvent, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "achieved_at", Value: -1}}).
		SetLimit(int64(limit))
	
	cursor, err := db.milestonesCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get milestones: %w", err)
	}
	defer cursor.Close(ctx)
	
	var milestones []MilestoneEvent
	if err := cursor.All(ctx, &milestones); err != nil {
		return nil, fmt.Errorf("failed to decode milestones: %w", err)
	}
	
	return milestones, nil
}