package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/models"
)

// Client wraps MongoDB operations
type Client struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewClient creates a new MongoDB client
func NewClient(uri string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	database := client.Database("mcp-registry")
	collection := database.Collection("servers_v2")

	return &Client{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// Disconnect closes the MongoDB connection
func (c *Client) Disconnect(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// GetAllServers retrieves all servers from MongoDB
func (c *Client) GetAllServers(ctx context.Context) ([]models.ServerDetail, error) {
	filter := bson.M{}
	cursor, err := c.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var servers []models.ServerDetail
	if err := cursor.All(ctx, &servers); err != nil {
		return nil, err
	}

	return servers, nil
}

// WatchChanges creates a change stream for real-time updates
func (c *Client) WatchChanges(ctx context.Context, handler func(changeType string, server *models.ServerDetail) error) error {
	pipeline := mongo.Pipeline{}
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	changeStream, err := c.collection.Watch(ctx, pipeline, opts)
	if err != nil {
		return err
	}
	defer changeStream.Close(ctx)

	for changeStream.Next(ctx) {
		var changeEvent struct {
			OperationType string               `bson:"operationType"`
			FullDocument  *models.ServerDetail `bson:"fullDocument"`
		}

		if err := changeStream.Decode(&changeEvent); err != nil {
			return err
		}

		if changeEvent.FullDocument != nil {
			if err := handler(changeEvent.OperationType, changeEvent.FullDocument); err != nil {
				return err
			}
		}
	}

	return changeStream.Err()
}