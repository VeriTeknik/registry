package sync

import (
	"context"
	"log"
	"time"

	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/elasticsearch"
	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/models"
	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/mongodb"
)

// Service handles synchronization between MongoDB and Elasticsearch
type Service struct {
	mongo *mongodb.Client
	es    *elasticsearch.Client
}

// NewService creates a new sync service
func NewService(mongo *mongodb.Client, es *elasticsearch.Client) *Service {
	return &Service{
		mongo: mongo,
		es:    es,
	}
}

// FullSync performs a full synchronization of all servers
func (s *Service) FullSync(ctx context.Context) error {
	log.Println("Starting full sync...")
	start := time.Now()

	// Get all servers from MongoDB
	servers, err := s.mongo.GetAllServers(ctx)
	if err != nil {
		return err
	}

	log.Printf("Found %d servers to sync", len(servers))

	// Bulk index to Elasticsearch
	if err := s.es.BulkIndex(ctx, servers); err != nil {
		return err
	}

	log.Printf("Full sync completed in %s", time.Since(start))
	return nil
}

// WatchChanges monitors MongoDB for real-time changes
func (s *Service) WatchChanges(ctx context.Context) error {
	log.Println("Starting change stream monitoring...")

	return s.mongo.WatchChanges(ctx, func(changeType string, server *models.ServerDetail) error {
		log.Printf("Change detected: %s for server %s", changeType, server.ID)

		switch changeType {
		case "insert", "update", "replace":
			return s.es.IndexServer(ctx, server)
		case "delete":
			return s.es.DeleteServer(ctx, server.ID)
		default:
			log.Printf("Unknown change type: %s", changeType)
		}

		return nil
	})
}