package stats

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	// Stats operations
	GetStats(ctx context.Context, serverID string) (*ServerStats, error)
	GetBatchStats(ctx context.Context, serverIDs []string) (map[string]*ServerStats, error)
	UpsertStats(ctx context.Context, stats *ServerStats) error
	IncrementInstallCount(ctx context.Context, serverID string) error
	UpdateRating(ctx context.Context, serverID string, rating float64) error
	
	// Leaderboard operations
	GetTopByInstalls(ctx context.Context, limit int) ([]*ServerStats, error)
	GetTopByRating(ctx context.Context, limit int) ([]*ServerStats, error)
	GetTrending(ctx context.Context, limit int) ([]*ServerStats, error)
	
	// Global stats
	GetGlobalStats(ctx context.Context) (*GlobalStats, error)
	
	// Bulk operations
	SyncAnalyticsData(ctx context.Context, updates []StatsUpdateRequest) error
	TransferStats(ctx context.Context, fromServerID, toServerID string) error
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
		{
			Keys:    bson.D{{Key: "server_id", Value: 1}},
			Options: options.Index().SetUnique(true),
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

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &MongoDatabase{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// GetStats retrieves stats for a single server
func (db *MongoDatabase) GetStats(ctx context.Context, serverID string) (*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var stats ServerStats
	err := db.collection.FindOne(ctx, bson.M{"server_id": serverID}).Decode(&stats)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return empty stats if none exist
			return &ServerStats{
				ServerID:          serverID,
				InstallationCount: 0,
				Rating:            0,
				RatingCount:       0,
				LastUpdated:       time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// GetBatchStats retrieves stats for multiple servers
func (db *MongoDatabase) GetBatchStats(ctx context.Context, serverIDs []string) (map[string]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	filter := bson.M{"server_id": bson.M{"$in": serverIDs}}
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
			result[serverID] = &ServerStats{
				ServerID:          serverID,
				InstallationCount: 0,
				Rating:            0,
				RatingCount:       0,
				LastUpdated:       time.Now(),
			}
		}
	}

	return result, nil
}

// UpsertStats creates or updates server stats
func (db *MongoDatabase) UpsertStats(ctx context.Context, stats *ServerStats) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	stats.LastUpdated = time.Now()
	
	filter := bson.M{"server_id": stats.ServerID}
	update := bson.M{"$set": stats}
	opts := options.Update().SetUpsert(true)

	_, err := db.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert stats: %w", err)
	}

	return nil
}

// IncrementInstallCount atomically increments the installation count
func (db *MongoDatabase) IncrementInstallCount(ctx context.Context, serverID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	filter := bson.M{"server_id": serverID}
	update := bson.M{
		"$inc": bson.M{"installation_count": 1},
		"$set": bson.M{"last_updated": time.Now()},
	}
	opts := options.Update().SetUpsert(true)

	_, err := db.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to increment install count: %w", err)
	}

	return nil
}

// UpdateRating updates the rating for a server
func (db *MongoDatabase) UpdateRating(ctx context.Context, serverID string, newRating float64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// First get current stats to calculate new average
	var current ServerStats
	err := db.collection.FindOne(ctx, bson.M{"server_id": serverID}).Decode(&current)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to get current stats: %w", err)
	}

	// Calculate new average rating
	totalRating := current.Rating * float64(current.RatingCount)
	newRatingCount := current.RatingCount + 1
	newAvgRating := (totalRating + newRating) / float64(newRatingCount)

	filter := bson.M{"server_id": serverID}
	update := bson.M{
		"$set": bson.M{
			"rating":       newAvgRating,
			"rating_count": newRatingCount,
			"last_updated": time.Now(),
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
func (db *MongoDatabase) GetTopByInstalls(ctx context.Context, limit int) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	opts := options.Find().
		SetSort(bson.D{{Key: "installation_count", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := db.collection.Find(ctx, bson.M{}, opts)
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
func (db *MongoDatabase) GetTopByRating(ctx context.Context, limit int) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Only include servers with at least 5 ratings
	filter := bson.M{"rating_count": bson.M{"$gte": 5}}
	opts := options.Find().
		SetSort(bson.D{{Key: "rating", Value: -1}}).
		SetLimit(int64(limit))

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
func (db *MongoDatabase) GetTrending(ctx context.Context, limit int) ([]*ServerStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// For now, return servers with most active installs
	// TODO: Implement proper trending algorithm based on growth rate
	opts := options.Find().
		SetSort(bson.D{{Key: "active_installs", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := db.collection.Find(ctx, bson.M{}, opts)
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

// GetGlobalStats returns aggregate statistics for all servers
func (db *MongoDatabase) GetGlobalStats(ctx context.Context) (*GlobalStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":              nil,
				"total_servers":    bson.M{"$sum": 1},
				"total_installs":   bson.M{"$sum": "$installation_count"},
				"active_servers":   bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$gt": bson.A{"$active_installs", 0}}, 1, 0}}},
				"total_rating":     bson.M{"$sum": bson.M{"$multiply": bson.A{"$rating", "$rating_count"}}},
				"total_ratings":    bson.M{"$sum": "$rating_count"},
			},
		},
	}

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
func (db *MongoDatabase) TransferStats(ctx context.Context, fromServerID, toServerID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Get source stats
	var sourceStats ServerStats
	err := db.collection.FindOne(ctx, bson.M{"server_id": fromServerID}).Decode(&sourceStats)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No stats to transfer
			return nil
		}
		return fmt.Errorf("failed to get source stats: %w", err)
	}

	// Update target stats
	filter := bson.M{"server_id": toServerID}
	update := bson.M{
		"$inc": bson.M{
			"installation_count": sourceStats.InstallationCount,
			"rating_count":       sourceStats.RatingCount,
		},
		"$set": bson.M{
			"last_updated": time.Now(),
		},
	}

	// Calculate new average rating if target has existing ratings
	var targetStats ServerStats
	err = db.collection.FindOne(ctx, filter).Decode(&targetStats)
	if err == nil && targetStats.RatingCount > 0 {
		// Weighted average of ratings
		totalRating := (targetStats.Rating * float64(targetStats.RatingCount)) + 
			(sourceStats.Rating * float64(sourceStats.RatingCount))
		totalCount := targetStats.RatingCount + sourceStats.RatingCount
		newRating := totalRating / float64(totalCount)
		update["$set"].(bson.M)["rating"] = newRating
	} else {
		// Just use source rating if no existing rating
		update["$set"].(bson.M)["rating"] = sourceStats.Rating
	}

	opts := options.Update().SetUpsert(true)
	_, err = db.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to transfer stats: %w", err)
	}

	// Optionally delete source stats
	// _, err = db.collection.DeleteOne(ctx, bson.M{"server_id": fromServerID})

	return nil
}