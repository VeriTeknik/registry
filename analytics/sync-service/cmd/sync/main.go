package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/elasticsearch"
	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/mongodb"
	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/sync"
)

func main() {
	log.Println("Starting MongoDB to Elasticsearch sync service...")

	// Configuration from environment
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017/mcp-registry")
	esURL := getEnv("ELASTICSEARCH_URL", "http://localhost:9200")
	syncInterval := getEnv("SYNC_INTERVAL", "60s")

	// Parse sync interval
	interval, err := time.ParseDuration(syncInterval)
	if err != nil {
		log.Fatalf("Invalid sync interval: %v", err)
	}

	// Create MongoDB client
	mongoClient, err := mongodb.NewClient(mongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	// Create Elasticsearch client
	esClient, err := elasticsearch.NewClient(esURL)
	if err != nil {
		log.Fatalf("Failed to connect to Elasticsearch: %v", err)
	}

	// Initialize indices
	if err := esClient.InitializeIndices(); err != nil {
		log.Printf("Warning: Failed to initialize indices: %v", err)
	}

	// Create sync service
	syncService := sync.NewService(mongoClient, esClient)

	// Start initial sync
	log.Println("Starting initial sync...")
	if err := syncService.FullSync(context.Background()); err != nil {
		log.Printf("Initial sync failed: %v", err)
	} else {
		log.Println("Initial sync completed")
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start change stream monitoring
	go func() {
		if err := syncService.WatchChanges(ctx); err != nil {
			log.Printf("Change stream error: %v", err)
		}
	}()

	// Start periodic full sync
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("Running periodic sync...")
				if err := syncService.FullSync(ctx); err != nil {
					log.Printf("Periodic sync failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down sync service...")
	cancel()
	time.Sleep(2 * time.Second) // Give goroutines time to finish
	log.Println("Sync service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}