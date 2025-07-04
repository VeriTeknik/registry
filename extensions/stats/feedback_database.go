package stats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/registry/internal/validation"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	feedbackCollectionName = "server_feedback"
)

var (
	// ErrDuplicateFeedback is returned when user tries to submit duplicate feedback
	ErrDuplicateFeedback = errors.New("user has already submitted feedback for this server")
	// ErrFeedbackNotFound is returned when feedback is not found
	ErrFeedbackNotFound = errors.New("feedback not found")
	// ErrUnauthorized is returned when user is not authorized to perform an action
	ErrUnauthorized = errors.New("unauthorized to perform this action")
)

// FeedbackDatabase interface defines feedback operations
type FeedbackDatabase interface {
	// Create operations
	CreateFeedback(ctx context.Context, feedback *ServerFeedback) error
	
	// Read operations
	GetFeedback(ctx context.Context, feedbackID string) (*ServerFeedback, error)
	GetServerFeedback(ctx context.Context, serverID string, source string, limit int, offset int, sort FeedbackSortOrder) (*FeedbackResponse, error)
	GetUserFeedback(ctx context.Context, serverID string, userID string, source string) (*ServerFeedback, error)
	GetUserFeedbackHistory(ctx context.Context, userID string, limit int, offset int) ([]*ServerFeedback, error)
	
	// Update operations
	UpdateFeedback(ctx context.Context, feedback *ServerFeedback) error
	
	// Delete operations
	DeleteFeedback(ctx context.Context, feedbackID string, userID string) error
	
	// Stats operations
	CountServerFeedback(ctx context.Context, serverID string, source string) (int, error)
}

// MongoFeedbackDatabase implements FeedbackDatabase using MongoDB
type MongoFeedbackDatabase struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoFeedbackDatabase creates a new MongoDB feedback database
func NewMongoFeedbackDatabase(client *mongo.Client, databaseName string) (*MongoFeedbackDatabase, error) {
	database := client.Database(databaseName)
	collection := database.Collection(feedbackCollectionName)

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	indexes := []mongo.IndexModel{
		// Unique compound index to prevent duplicate feedback
		{
			Keys:    bson.D{{Key: "server_id", Value: 1}, {Key: "user_id", Value: 1}, {Key: "source", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Index for querying feedback by server
		{
			Keys: bson.D{{Key: "server_id", Value: 1}, {Key: "source", Value: 1}},
		},
		// Index for user's feedback history
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
		// Index for sorting by creation date
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		// Index for sorting by rating
		{
			Keys: bson.D{{Key: "rating", Value: -1}},
		},
	}

	// Create indexes one by one to handle conflicts gracefully
	for _, index := range indexes {
		_, err := collection.Indexes().CreateOne(ctx, index)
		if err != nil {
			// Log index creation errors but don't fail if index already exists
			if !strings.Contains(err.Error(), "IndexKeySpecsConflict") && !strings.Contains(err.Error(), "already exists") {
				return nil, fmt.Errorf("failed to create feedback index: %w", err)
			}
			// Index already exists, which is fine
		}
	}

	return &MongoFeedbackDatabase{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// CreateFeedback creates new feedback
func (db *MongoFeedbackDatabase) CreateFeedback(ctx context.Context, feedback *ServerFeedback) error {
	// Validate input
	if feedback.ServerID == "" || feedback.UserID == "" {
		return errors.New("server_id and user_id are required")
	}

	// Sanitize server ID
	sanitizedServerID, err := validation.SanitizeServerID(feedback.ServerID)
	if err != nil {
		return fmt.Errorf("invalid server ID: %w", err)
	}
	feedback.ServerID = sanitizedServerID

	// Validate source
	if feedback.Source == "" {
		feedback.Source = SourceRegistry
	}
	validatedSource, err := validation.ValidateSource(feedback.Source)
	if err != nil {
		return fmt.Errorf("invalid source: %w", err)
	}
	feedback.Source = validatedSource

	// Generate ID if not provided
	if feedback.ID == "" {
		feedback.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	feedback.CreatedAt = now
	feedback.UpdatedAt = now

	// Default to public
	if !feedback.IsPublic {
		feedback.IsPublic = true
	}

	// Insert the document
	_, err = db.collection.InsertOne(ctx, feedback)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateFeedback
		}
		return fmt.Errorf("failed to create feedback: %w", err)
	}

	return nil
}

// GetFeedback retrieves a single feedback by ID
func (db *MongoFeedbackDatabase) GetFeedback(ctx context.Context, feedbackID string) (*ServerFeedback, error) {
	var feedback ServerFeedback
	
	filter := bson.M{"_id": feedbackID}
	err := db.collection.FindOne(ctx, filter).Decode(&feedback)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrFeedbackNotFound
		}
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}

	return &feedback, nil
}

// GetServerFeedback retrieves all feedback for a server with pagination
func (db *MongoFeedbackDatabase) GetServerFeedback(ctx context.Context, serverID string, source string, limit int, offset int, sort FeedbackSortOrder) (*FeedbackResponse, error) {
	// Validate and sanitize server ID
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %w", err)
	}

	// Validate source
	if source == "" {
		source = SourceRegistry
	}
	validatedSource, err := validation.ValidateSource(source)
	if err != nil {
		return nil, fmt.Errorf("invalid source: %w", err)
	}

	// Build filter
	filter := bson.M{
		"server_id": sanitizedServerID,
		"source":    validatedSource,
		"is_public": true,
	}

	// Count total feedback
	totalCount, err := db.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count feedback: %w", err)
	}

	// Validate limit and offset
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Build sort options
	var sortOpts bson.D
	switch sort {
	case FeedbackSortNewest:
		sortOpts = bson.D{{Key: "created_at", Value: -1}}
	case FeedbackSortOldest:
		sortOpts = bson.D{{Key: "created_at", Value: 1}}
	case FeedbackSortRatingHigh:
		sortOpts = bson.D{{Key: "rating", Value: -1}, {Key: "created_at", Value: -1}}
	case FeedbackSortRatingLow:
		sortOpts = bson.D{{Key: "rating", Value: 1}, {Key: "created_at", Value: -1}}
	default:
		sortOpts = bson.D{{Key: "created_at", Value: -1}}
	}

	// Query with pagination
	opts := options.Find().
		SetSort(sortOpts).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query feedback: %w", err)
	}
	defer cursor.Close(ctx)

	var feedbackList []*ServerFeedback
	if err := cursor.All(ctx, &feedbackList); err != nil {
		return nil, fmt.Errorf("failed to decode feedback: %w", err)
	}
	
	// Ensure feedback list is not nil
	if feedbackList == nil {
		feedbackList = []*ServerFeedback{}
	}

	// Build response
	response := &FeedbackResponse{
		Feedback:   feedbackList,
		TotalCount: int(totalCount),
		HasMore:    int64(offset+limit) < totalCount,
	}

	return response, nil
}

// GetUserFeedback retrieves a user's feedback for a specific server
func (db *MongoFeedbackDatabase) GetUserFeedback(ctx context.Context, serverID string, userID string, source string) (*ServerFeedback, error) {
	// Validate inputs
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %w", err)
	}

	if source == "" {
		source = SourceRegistry
	}
	validatedSource, err := validation.ValidateSource(source)
	if err != nil {
		return nil, fmt.Errorf("invalid source: %w", err)
	}

	var feedback ServerFeedback
	filter := bson.M{
		"server_id": sanitizedServerID,
		"user_id":   userID,
		"source":    validatedSource,
	}

	err = db.collection.FindOne(ctx, filter).Decode(&feedback)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrFeedbackNotFound
		}
		return nil, fmt.Errorf("failed to get user feedback: %w", err)
	}

	return &feedback, nil
}

// GetUserFeedbackHistory retrieves all feedback submitted by a user
func (db *MongoFeedbackDatabase) GetUserFeedbackHistory(ctx context.Context, userID string, limit int, offset int) ([]*ServerFeedback, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	// Validate limit and offset
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query user feedback: %w", err)
	}
	defer cursor.Close(ctx)

	var feedbackList []*ServerFeedback
	if err := cursor.All(ctx, &feedbackList); err != nil {
		return nil, fmt.Errorf("failed to decode feedback: %w", err)
	}

	return feedbackList, nil
}

// UpdateFeedback updates existing feedback
func (db *MongoFeedbackDatabase) UpdateFeedback(ctx context.Context, feedback *ServerFeedback) error {
	if feedback.ID == "" {
		return errors.New("feedback ID is required")
	}

	// Update timestamp
	feedback.UpdatedAt = time.Now()

	// Build update document
	update := bson.M{
		"$set": bson.M{
			"rating":     feedback.Rating,
			"comment":    feedback.Comment,
			"updated_at": feedback.UpdatedAt,
		},
	}

	// Update with user check
	filter := bson.M{
		"_id":     feedback.ID,
		"user_id": feedback.UserID,
	}

	result, err := db.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update feedback: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrFeedbackNotFound
	}

	return nil
}

// DeleteFeedback deletes feedback (only by the user who created it)
func (db *MongoFeedbackDatabase) DeleteFeedback(ctx context.Context, feedbackID string, userID string) error {
	if feedbackID == "" || userID == "" {
		return errors.New("feedback_id and user_id are required")
	}

	// Delete with user check
	filter := bson.M{
		"_id":     feedbackID,
		"user_id": userID,
	}

	result, err := db.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete feedback: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrFeedbackNotFound
	}

	return nil
}

// CountServerFeedback counts the total feedback for a server
func (db *MongoFeedbackDatabase) CountServerFeedback(ctx context.Context, serverID string, source string) (int, error) {
	// Validate inputs
	sanitizedServerID, err := validation.SanitizeServerID(serverID)
	if err != nil {
		return 0, fmt.Errorf("invalid server ID: %w", err)
	}

	if source == "" {
		source = SourceRegistry
	}
	validatedSource, err := validation.ValidateSource(source)
	if err != nil {
		return 0, fmt.Errorf("invalid source: %w", err)
	}

	filter := bson.M{
		"server_id": sanitizedServerID,
		"source":    validatedSource,
		"is_public": true,
	}

	count, err := db.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count feedback: %w", err)
	}

	return int(count), nil
}