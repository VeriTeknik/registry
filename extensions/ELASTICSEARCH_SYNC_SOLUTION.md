# Elasticsearch to MongoDB Sync Solution

## Date: 2025-07-05

## Problem Statement

The VP Analytics API reads from MongoDB, but events are being tracked to Elasticsearch. The existing sync service (`/analytics/sync-service/`) syncs FROM MongoDB TO Elasticsearch, but we need the opposite direction.

## Current State

1. **Events are tracked to Elasticsearch** ✅
   - pluggedin-app sends events to analytics.plugged.in
   - Events include both UUID and registry ID servers

2. **VP API reads from MongoDB** ✅
   - Analytics endpoints query MongoDB collections
   - Code already accepts all server ID formats

3. **Missing: Elasticsearch → MongoDB sync** ❌
   - No sync service exists for this direction
   - Events in Elasticsearch are not visible in VP dashboard

## Solution Options

### Option 1: Create Elasticsearch → MongoDB Sync Service (Recommended)

Create a new sync service that:
1. Reads events from Elasticsearch
2. Transforms them to MongoDB format
3. Stores in MongoDB collections that VP API uses

**Implementation Path:**
```go
// New sync service structure
type ElasticsearchToMongoSync struct {
    esClient    *elasticsearch.Client
    mongoDB     *mongo.Database
    interval    time.Duration
}

// Sync flow
func (s *ElasticsearchToMongoSync) SyncEvents(ctx context.Context) error {
    // 1. Query Elasticsearch for recent events
    events, err := s.esClient.GetRecentEvents(lastSyncTime)
    
    // 2. Transform events to MongoDB format
    for _, event := range events {
        // Transform based on event type
        switch event.Type {
        case "installation":
            s.recordInstallation(event)
        case "usage":
            s.recordUsage(event)
        }
    }
    
    // 3. Update MongoDB collections
    // - activity_events
    // - analytics_metrics
    // - time_series_data
}
```

### Option 2: Modify VP API to Read from Elasticsearch

Change the VP analytics handlers to query Elasticsearch directly instead of MongoDB.

**Pros:**
- Real-time data
- No sync delay
- Single source of truth

**Cons:**
- Requires significant code changes
- Different query syntax
- May impact performance

### Option 3: Configure External Analytics URL

The existing code has provision for external analytics:

```go
// In router.go
if config.AnalyticsBaseURL != "" {
    analyticsClient := stats.NewHTTPAnalyticsClient(config.AnalyticsBaseURL)
    syncService := stats.NewSyncService(statsDB, analyticsClient, 15*time.Minute)
    go syncService.Start(context.Background())
}
```

Set `MCP_REGISTRY_ANALYTICS_URL` to point to an analytics API that reads from Elasticsearch.

## Recommended Implementation

### 1. Quick Fix: Enable Analytics Sync

```bash
# Set environment variable
export MCP_REGISTRY_ANALYTICS_URL=https://analytics.plugged.in

# Restart registry service
docker-compose restart registry
```

### 2. Long-term: Elasticsearch Sync Service

Create `/extensions/elasticsearch-sync/`:

```go
package main

import (
    "context"
    "time"
    "go.mongodb.org/mongo-driver/mongo"
    elastic "github.com/elastic/go-elasticsearch/v8"
)

func main() {
    // Connect to Elasticsearch
    es, _ := elastic.NewClient(elastic.Config{
        Addresses: []string{"https://analytics.plugged.in"},
    })
    
    // Connect to MongoDB
    mongo, _ := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    
    // Sync loop
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        syncElasticsearchToMongo(es, mongo)
    }
}

func syncElasticsearchToMongo(es *elastic.Client, mongo *mongo.Client) {
    // 1. Query Elasticsearch for events since last sync
    // 2. Group by event type
    // 3. Update MongoDB collections:
    //    - activity_events: Raw events
    //    - analytics_metrics: Aggregated counts
    //    - server_stats: Per-server statistics
}
```

## Verification Steps

1. **Check current MongoDB state**:
```bash
# Connect to MongoDB
docker exec -it mongodb mongosh

# Check if events exist
use mcp-registry
db.activity_events.countDocuments()
db.analytics_metrics.findOne()
```

2. **Test with analytics URL**:
```bash
# Set analytics URL and restart
export MCP_REGISTRY_ANALYTICS_URL=https://analytics.plugged.in
docker-compose up -d registry

# Check logs
docker logs registry-extended | grep -i sync
```

3. **Verify dashboard shows data**:
```bash
curl https://registry.plugged.in/vp/analytics/dashboard
```

## Summary

The VP Analytics code is ready and accepts all server ID formats. The missing piece is syncing events from Elasticsearch to MongoDB. The quickest solution is to configure the external analytics URL if an appropriate endpoint exists. Otherwise, implement an Elasticsearch → MongoDB sync service.