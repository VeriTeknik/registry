package stats

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/registry/internal/validation"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	statsCollectionName = "server_stats"
	defaultTimeout      = 10 * time.Second
)

// Database interface defines stats database operations
type Database interface {
	// Stats operations with source support
	GetStats(ctx context.Context, serverID string, source string) (*ServerStats, error)
	GetStatsByServerID(ctx context.Context, serverID string) ([]*ServerStats, error)
	GetBatchStats(ctx context.Context, serverIDs []string, source string) (map[string]*ServerStats, error)
	GetAggregatedStats(ctx context.Context, serverID string) (*AggregatedStats, error)
	UpsertStats(ctx context.Context, stats *ServerStats) error
	IncrementInstallCount(ctx context.Context, serverID string, source string) error
	UpdateRating(ctx context.Context, serverID string, source string, rating float64) error
	
	// Leaderboard operations with source support
	GetTopByInstalls(ctx context.Context, limit int, source string) ([]*ServerStats, error)
	GetTopByRating(ctx context.Context, limit int, source string) ([]*ServerStats, error)
	GetTrending(ctx context.Context, limit int, source string) ([]*ServerStats, error)
	GetRecentServers(ctx context.Context, limit int, source string) ([]*ServerStats, error)
	
	// Global stats with source support
	GetGlobalStats(ctx context.Context, source string) (*GlobalStats, error)
	
	// Bulk operations
	SyncAnalyticsData(ctx context.Context, updates []StatsUpdateRequest) error
	TransferStats(ctx context.Context, fromServerID, toServerID, fromSource, toSource string) error
	
	// Migration
	MigrateExistingStats(ctx context.Context) error
}

// MongoDatabase implements Database interface using MongoDB
type MongoDatabase struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	mu         sync.RWMutex
}

// NewMongoDatabase creates a new MongoDB stats database
func NewMongoDatabase(client *mongo.Client, databaseName string) (*MongoDatabase, error) {
	if client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	database := client.Database(databaseName)
	collection := database.Collection(statsCollectionName)

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	indexes := []mongo.IndexModel{
		// Compound unique index on server_id + source
		{
			Keys:    bson.D{{Key: "server_id", Value: 1}, {Key: "source", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Keep simple server_id index for backward compatibility
		{
			Keys: bson.D{{Key: "server_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "source", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "source", Value: 1}, {Key: "installation_count", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "source", Value: 1}, {Key: "rating", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "installation_count", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "rating", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "active_installs", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "last_updated", Value: -1}},
		},
	}

	// Create indexes one by one to handle conflicts gracefully
	for _, index := range indexes {
		_, err := collection.Indexes().CreateOne(ctx, index)
		if err != nil {
			// Log index creation errors but don't fail if index already exists
			if !strings.Contains(err.Error(), "IndexKeySpecsConflict") && !strings.Contains(err.Error(), "already exists") {
				return nil, fmt.Errorf("failed to create index: %w", err)
			}
			// Index already exists, which is fine
		}
	}

	return &MongoDatabase{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// GetStats retrieves stats for a single server with specific source
func (db *MongoDatabase) GetStats(ctx context.Context, serverID string, source string) (*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Validate server ID
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %w", err)
	}

	// Default to REGISTRY if source not specified
	if source == "" {
		source = SourceRegistry
	}

	// Validate source
	validatedSource, err := validation.ValidateSource(source)
	if err != nil {
		return nil, fmt.Errorf("invalid source parameter: %w", err)
	}

	var stats ServerStats
	filter := bson.M{
		"server_id": sanitizedServerID,
		"source":    validatedSource,
	}
	
	err = db.collection.FindOne(ctx, filter).Decode(&stats)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return empty stats if none exist
			now := time.Now()
			return &ServerStats{
				ServerID:          sanitizedServerID,
				Source:            validatedSource,
				InstallationCount: 0,
				Rating:            0,
				RatingCount:       0,
				FirstSeen:         now,
				LastUpdated:       now,
			}, nil
		}
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// GetStatsByServerID retrieves all stats entries for a server (all sources)
func (db *MongoDatabase) GetStatsByServerID(ctx context.Context, serverID string) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Validate server ID
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %w", err)
	}

	filter := bson.M{"server_id": sanitizedServerID}
	cursor, err := db.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats by server ID: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*ServerStats
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return results, nil
}

// GetBatchStats retrieves stats for multiple servers with specific source
func (db *MongoDatabase) GetBatchStats(ctx context.Context, serverIDs []string, source string) (map[string]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Default to REGISTRY if source not specified
	if source == "" {
		source = SourceRegistry
	}

	filter := bson.M{
		"server_id": bson.M{"$in": serverIDs},
		"source":    source,
	}
	
	cursor, err := db.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch stats: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[string]*ServerStats)
	for cursor.Next(ctx) {
		var stats ServerStats
		if err := cursor.Decode(&stats); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
		result[stats.ServerID] = &stats
	}

	// Fill in missing stats with empty values
	for _, serverID := range serverIDs {
		if _, exists := result[serverID]; !exists {
			now := time.Now()
			result[serverID] = &ServerStats{
				ServerID:          serverID,
				Source:            source,
				InstallationCount: 0,
				Rating:            0,
				RatingCount:       0,
				FirstSeen:         now,
				LastUpdated:       now,
			}
		}
	}

	return result, nil
}

// GetAggregatedStats retrieves combined stats from all sources for a server
func (db *MongoDatabase) GetAggregatedStats(ctx context.Context, serverID string) (*AggregatedStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Validate server ID first
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %w", err)
	}

	// Get all stats for this server
	allStats, err := db.GetStatsByServerID(ctx, sanitizedServerID)
	if err != nil {
		return nil, err
	}

	if len(allStats) == 0 {
		return &AggregatedStats{
			ServerID:         sanitizedServerID,
			TotalInstalls:    0,
			AverageRating:    0,
			TotalRatingCount: 0,
			SourceBreakdown:  make(map[string]*ServerStats),
			LastUpdated:      time.Now(),
		}, nil
	}

	// Aggregate the stats
	aggregated := &AggregatedStats{
		ServerID:        sanitizedServerID,
		SourceBreakdown: make(map[string]*ServerStats),
		LastUpdated:     time.Now(),
	}

	totalRating := float64(0)
	for _, stats := range allStats {
		aggregated.TotalInstalls += stats.InstallationCount
		aggregated.TotalRatingCount += stats.RatingCount
		totalRating += stats.Rating * float64(stats.RatingCount)
		aggregated.SourceBreakdown[stats.Source] = stats
	}

	if aggregated.TotalRatingCount > 0 {
		aggregated.AverageRating = totalRating / float64(aggregated.TotalRatingCount)
	}

	return aggregated, nil
}

// UpsertStats creates or updates server stats
func (db *MongoDatabase) UpsertStats(ctx context.Context, stats *ServerStats) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Default to REGISTRY if source not specified
	if stats.Source == "" {
		stats.Source = SourceRegistry
	}

	stats.LastUpdated = time.Now()
	
	filter := bson.M{
		"server_id": stats.ServerID,
		"source":    stats.Source,
	}
	update := bson.M{"$set": stats}
	opts := options.Update().SetUpsert(true)

	_, err := db.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert stats: %w", err)
	}

	return nil
}

// IncrementInstallCount atomically increments the installation count
func (db *MongoDatabase) IncrementInstallCount(ctx context.Context, serverID string, source string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Validate server ID
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return fmt.Errorf("invalid server ID: %w", err)
	}

	// Default to REGISTRY if source not specified
	if source == "" {
		source = SourceRegistry
	}

	// Validate source
	validatedSource, err := validation.ValidateSource(source)
	if err != nil {
		return fmt.Errorf("invalid source parameter: %w", err)
	}

	filter := bson.M{
		"server_id": sanitizedServerID,
		"source":    validatedSource,
	}
	now := time.Now()
	update := bson.M{
		"$inc": bson.M{"installation_count": 1},
		"$set": bson.M{
			"last_updated": now,
			"source":       validatedSource,
		},
		"$setOnInsert": bson.M{
			"first_seen": now,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err = db.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to increment install count: %w", err)
	}

	return nil
}

// UpdateRating updates the rating for a server
func (db *MongoDatabase) UpdateRating(ctx context.Context, serverID string, source string, newRating float64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Validate server ID
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return fmt.Errorf("invalid server ID: %w", err)
	}

	// Default to REGISTRY if source not specified
	if source == "" {
		source = SourceRegistry
	}

	// Validate source
	validatedSource, err := validation.ValidateSource(source)
	if err != nil {
		return fmt.Errorf("invalid source parameter: %w", err)
	}

	// First get current stats to calculate new average
	var current ServerStats
	filter := bson.M{
		"server_id": sanitizedServerID,
		"source":    validatedSource,
	}
	
	err = db.collection.FindOne(ctx, filter).Decode(&current)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to get current stats: %w", err)
	}

	// Calculate new average rating
	totalRating := current.Rating * float64(current.RatingCount)
	newRatingCount := current.RatingCount + 1
	newAvgRating := (totalRating + newRating) / float64(newRatingCount)

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"rating":       newAvgRating,
			"rating_count": newRatingCount,
			"last_updated": now,
			"source":       source,
		},
		"$setOnInsert": bson.M{
			"first_seen": now,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err = db.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update rating: %w", err)
	}

	return nil
}

// GetTopByInstalls returns servers with highest installation counts
func (db *MongoDatabase) GetTopByInstalls(ctx context.Context, limit int, source string) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Validate limit
	validatedLimit, err := validation.ValidateLimit(limit)
	if err != nil {
		return nil, fmt.Errorf("invalid limit: %w", err)
	}

	// Create safe filter using validation
	filter, err := validation.CreateSafeFilter(source)
	if err != nil {
		return nil, fmt.Errorf("invalid source parameter: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "installation_count", Value: -1}}).
		SetLimit(int64(validatedLimit))

	cursor, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top by installs: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*ServerStats
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	return results, nil
}

// GetTopByRating returns servers with highest ratings
func (db *MongoDatabase) GetTopByRating(ctx context.Context, limit int, source string) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Validate limit
	validatedLimit, err := validation.ValidateLimit(limit)
	if err != nil {
		return nil, fmt.Errorf("invalid limit: %w", err)
	}

	// Create base filter for rating count
	filter := bson.M{"rating_count": bson.M{"$gte": 5}}
	
	// Validate and add source filter if provided
	if source != "" && source != "ALL" {
		validatedSource, err := validation.ValidateSource(source)
		if err != nil {
			return nil, fmt.Errorf("invalid source parameter: %w", err)
		}
		filter["source"] = validatedSource
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "rating", Value: -1}}).
		SetLimit(int64(validatedLimit))

	cursor, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top by rating: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*ServerStats
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	return results, nil
}

// GetTrending returns trending servers based on recent growth
func (db *MongoDatabase) GetTrending(ctx context.Context, limit int, source string) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Validate limit
	validatedLimit, err := validation.ValidateLimit(limit)
	if err != nil {
		return nil, fmt.Errorf("invalid limit: %w", err)
	}

	// Create safe filter using validation
	filter, err := validation.CreateSafeFilter(source)
	if err != nil {
		return nil, fmt.Errorf("invalid source parameter: %w", err)
	}

	// Implement trending algorithm based on install growth rate and recent activity
	// Formula: trend_score = (recent_installs * 2) + (total_installs * 0.1) + (rating * rating_count * 0.05)
	// This favors servers with recent growth while still considering established servers
	pipeline := []bson.M{
		{"$match": filter},
		{"$addFields": bson.M{
			"trend_score": bson.M{
				"$add": []interface{}{
					// Recent installs weighted heavily (assuming last 7 days activity)
					bson.M{"$multiply": []interface{}{"$weekly_growth", 10}},
					// Total installs weighted lightly for established servers
					bson.M{"$multiply": []interface{}{"$installation_count", 0.1}},
					// Rating quality factor
					bson.M{"$multiply": []interface{}{
						"$rating",
						bson.M{"$multiply": []interface{}{"$rating_count", 0.05}},
					}},
					// Active installs factor
					bson.M{"$multiply": []interface{}{"$active_installs", 0.3}},
				},
			},
		}},
		{"$sort": bson.M{"trend_score": -1}},
		{"$limit": int64(validatedLimit)},
	}

	cursor, err := db.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*ServerStats
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	return results, nil
}

// GetRecentServers returns servers ordered by first_seen date (most recent first)
func (db *MongoDatabase) GetRecentServers(ctx context.Context, limit int, source string) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Validate limit
	validatedLimit, err := validation.ValidateLimit(limit)
	if err != nil {
		return nil, fmt.Errorf("invalid limit: %w", err)
	}

	// Create base filter
	filter := bson.M{}
	
	// Validate and add source filter if provided
	if source != "" && source != "ALL" {
		validatedSource, err := validation.ValidateSource(source)
		if err != nil {
			return nil, fmt.Errorf("invalid source parameter: %w", err)
		}
		filter["source"] = validatedSource
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "first_seen", Value: -1}}).
		SetLimit(int64(validatedLimit))

	cursor, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent servers: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*ServerStats
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	return results, nil
}

// GetGlobalStats returns aggregate statistics for all servers
func (db *MongoDatabase) GetGlobalStats(ctx context.Context, source string) (*GlobalStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Build match stage for source filtering
	pipeline := []bson.M{}
	if source != "" && source != "ALL" {
		validatedSource, err := validation.ValidateSource(source)
		if err != nil {
			return nil, fmt.Errorf("invalid source parameter: %w", err)
		}
		pipeline = append(pipeline, bson.M{
			"$match": bson.M{"source": validatedSource},
		})
	}

	// Add aggregation stage
	pipeline = append(pipeline, bson.M{
		"$group": bson.M{
			"_id":              nil,
			"total_servers":    bson.M{"$sum": 1},
			"total_installs":   bson.M{"$sum": "$installation_count"},
			"active_servers":   bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$gt": bson.A{"$active_installs", 0}}, 1, 0}}},
			"total_rating":     bson.M{"$sum": bson.M{"$multiply": bson.A{"$rating", "$rating_count"}}},
			"total_ratings":    bson.M{"$sum": "$rating_count"},
		},
	})

	cursor, err := db.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate global stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalServers  int     `bson:"total_servers"`
		TotalInstalls int     `bson:"total_installs"`
		ActiveServers int     `bson:"active_servers"`
		TotalRating   float64 `bson:"total_rating"`
		TotalRatings  int     `bson:"total_ratings"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode global stats: %w", err)
		}
	}

	avgRating := float64(0)
	if result.TotalRatings > 0 {
		avgRating = result.TotalRating / float64(result.TotalRatings)
	}

	return &GlobalStats{
		TotalServers:  result.TotalServers,
		TotalInstalls: result.TotalInstalls,
		ActiveServers: result.ActiveServers,
		AverageRating: avgRating,
		LastUpdated:   time.Now(),
	}, nil
}

// SyncAnalyticsData updates stats with data from analytics service
func (db *MongoDatabase) SyncAnalyticsData(ctx context.Context, updates []StatsUpdateRequest) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Use bulk write for efficiency
	var models []mongo.WriteModel
	for _, update := range updates {
		filter := bson.M{"server_id": update.InstallationDelta}
		updateDoc := bson.M{
			"$set": bson.M{
				"active_installs":      update.ActiveInstalls,
				"daily_active_users":   update.DailyActiveUsers,
				"monthly_active_users": update.MonthlyActiveUsers,
				"last_updated":         time.Now(),
			},
		}
		
		model := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(updateDoc).
			SetUpsert(true)
		
		models = append(models, model)
	}

	if len(models) > 0 {
		_, err := db.collection.BulkWrite(ctx, models)
		if err != nil {
			return fmt.Errorf("failed to sync analytics data: %w", err)
		}
	}

	return nil
}

// TransferStats transfers stats from one server to another (for claiming)
func (db *MongoDatabase) TransferStats(ctx context.Context, fromServerID, toServerID, fromSource, toSource string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Default sources
	if fromSource == "" {
		fromSource = SourceCommunity
	}
	if toSource == "" {
		toSource = SourceRegistry
	}

	// Get source stats
	var sourceStats ServerStats
	sourceFilter := bson.M{
		"server_id": fromServerID,
		"source":    fromSource,
	}
	err := db.collection.FindOne(ctx, sourceFilter).Decode(&sourceStats)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No stats to transfer
			return nil
		}
		return fmt.Errorf("failed to get source stats: %w", err)
	}

	// Get target stats if they exist
	targetFilter := bson.M{
		"server_id": toServerID,
		"source":    toSource,
	}
	var targetStats ServerStats
	targetExists := true
	err = db.collection.FindOne(ctx, targetFilter).Decode(&targetStats)
	if err == mongo.ErrNoDocuments {
		targetExists = false
	} else if err != nil {
		return fmt.Errorf("failed to get target stats: %w", err)
	}

	// Calculate merged stats
	var newStats ServerStats
	if targetExists {
		// Merge stats
		newStats = ServerStats{
			ServerID:          toServerID,
			Source:            toSource,
			InstallationCount: targetStats.InstallationCount + sourceStats.InstallationCount,
			RatingCount:       targetStats.RatingCount + sourceStats.RatingCount,
			LastUpdated:       time.Now(),
		}
		
		// Calculate weighted average rating
		if newStats.RatingCount > 0 {
			totalRating := (targetStats.Rating * float64(targetStats.RatingCount)) +
				(sourceStats.Rating * float64(sourceStats.RatingCount))
			newStats.Rating = totalRating / float64(newStats.RatingCount)
		}
		
		// Preserve analytics metrics from target
		newStats.ActiveInstalls = targetStats.ActiveInstalls
		newStats.DailyActiveUsers = targetStats.DailyActiveUsers
		newStats.MonthlyActiveUsers = targetStats.MonthlyActiveUsers
	} else {
		// Copy source stats to target with new source
		newStats = sourceStats
		newStats.ServerID = toServerID
		newStats.Source = toSource
		newStats.LastUpdated = time.Now()
	}

	// Add claim tracking
	newStats.ClaimedFrom = fromSource
	newStats.ClaimedAt = time.Now()

	// Upsert the new stats
	if err := db.UpsertStats(ctx, &newStats); err != nil {
		return fmt.Errorf("failed to upsert transferred stats: %w", err)
	}

	// Mark source stats as claimed (don't delete for audit trail)
	claimUpdate := bson.M{
		"$set": bson.M{
			"claimed_at": time.Now(),
			"claimed_to": toServerID,
		},
	}
	_, err = db.collection.UpdateOne(ctx, sourceFilter, claimUpdate)
	if err != nil {
		// Log but don't fail the transfer
		fmt.Printf("Failed to mark source stats as claimed: %v\n", err)
	}

	return nil
}

// MigrateExistingStats adds source field to existing stats entries
func (db *MongoDatabase) MigrateExistingStats(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Update all documents without a source field to have source=REGISTRY
	filter := bson.M{"source": bson.M{"$exists": false}}
	update := bson.M{
		"$set": bson.M{
			"source": SourceRegistry,
			"last_updated": time.Now(),
		},
	}

	_, err := db.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to migrate existing stats: %w", err)
	}

	return nil
}