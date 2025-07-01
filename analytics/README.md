# MCP Registry Analytics System

A comprehensive analytics platform for tracking MCP server usage, performance, and community engagement.

## Architecture Overview

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   MCP Registry  │     │  Analytics API   │     │    Frontend     │
│   (MongoDB)     │     │  (Go/Gin)        │     │  (plugged.in)   │
└────────┬────────┘     └────────┬─────────┘     └────────┬────────┘
         │                       │                          │
         ▼                       ▼                          │
┌─────────────────┐     ┌─────────────────┐               │
│   Sync Service  │────►│  Elasticsearch  │◄──────────────┘
│   (Go)          │     │                 │
└─────────────────┘     └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │     Kibana      │
                        │  (Dashboards)   │
                        └─────────────────┘
```

## Components

### 1. MongoDB Sync Service
- Monitors MongoDB change streams for real-time updates
- Performs periodic full synchronization
- Transforms and indexes server data to Elasticsearch

### 2. Analytics API
- RESTful API for event tracking and analytics queries
- Redis caching for frequently accessed data
- GitHub authentication for user-specific features
- CORS-enabled for frontend integration

### 3. Elasticsearch
- Stores server metadata, events, metrics, and feedback
- Provides powerful search and aggregation capabilities
- Time-series data with automatic retention policies

### 4. Kibana
- Pre-built dashboards for server analytics
- Real-time monitoring of trends and usage
- Custom visualizations for business insights

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Access to MongoDB instance (from main registry)

### Starting the Analytics Stack

```bash
cd analytics
docker compose up -d
```

This will start:
- Elasticsearch on http://localhost:9200
- Kibana on http://localhost:5601
- Redis on localhost:6379
- Sync Service (automatic)
- Analytics API on http://localhost:8081

### Accessing Services
- **Kibana Dashboard**: https://kibana.plugged.in
- **Analytics API**: https://analytics.plugged.in
- **API Documentation**: https://analytics.plugged.in/docs

## API Endpoints

### Event Tracking
```bash
POST /api/v1/track
{
  "event_type": "install",
  "server_id": "uuid",
  "client_id": "anonymous-uuid",
  "metadata": {
    "version": "1.0.0",
    "platform": "macos"
  }
}
```

### Server Statistics
```bash
GET /api/v1/servers/{id}/stats
GET /api/v1/servers/{id}/timeline?period=30d
```

### Trending & Discovery
```bash
GET /api/v1/trending?period=24h
GET /api/v1/popular?category=database
GET /api/v1/search?q=sqlite&package_types=npm,docker
```

### User Feedback
```bash
POST /api/v1/servers/{id}/rate
{
  "rating": 5,
  "comment": "Excellent server!"
}

GET /api/v1/servers/{id}/ratings
GET /api/v1/servers/{id}/comments
```

## Data Retention

- **Raw Events**: 90 days (configurable)
- **Aggregated Metrics**: Indefinite
- **User Feedback**: Indefinite
- **Server Metadata**: Synced with MongoDB

## Privacy & Security

### Anonymous Tracking
- Client IDs are randomly generated UUIDs
- No personally identifiable information collected
- IP addresses are hashed before storage
- Country/region derived from IP then discarded

### Authentication
- GitHub OAuth for ratings and comments
- API key authentication for write operations
- CORS configured for trusted origins only

## Monitoring & Maintenance

### Health Checks
```bash
curl https://analytics.plugged.in/api/v1/health
```

### Elasticsearch Indices
```bash
# Check index health
curl localhost:9200/_cat/indices?v

# Check cluster health
curl localhost:9200/_cluster/health?pretty
```

### Sync Service Logs
```bash
docker logs analytics-sync -f
```

## Development

### Running Locally
```bash
# Start dependencies
docker compose up elasticsearch redis -d

# Run sync service
cd sync-service
go run cmd/sync/main.go

# Run API
cd ../analytics-api
go run cmd/api/main.go
```

### Adding New Metrics
1. Update event schema in `elasticsearch/mappings/events.json`
2. Add aggregation logic in `analytics-api/internal/service`
3. Create visualization in Kibana
4. Update API documentation

## Troubleshooting

### Sync Service Issues
- Check MongoDB connection: `docker logs analytics-sync`
- Verify change stream permissions in MongoDB
- Ensure Elasticsearch is healthy

### Performance Issues
- Check Elasticsearch heap size
- Review Redis memory usage
- Enable query profiling in Elasticsearch

### Data Inconsistencies
- Run manual full sync: restart sync service
- Check for mapping conflicts in Elasticsearch
- Verify MongoDB replica set configuration

## Future Enhancements

1. **Machine Learning**
   - Anomaly detection for usage patterns
   - Server recommendation engine
   - Predictive trending analysis

2. **Advanced Analytics**
   - Cohort analysis for retention
   - A/B testing framework
   - Custom event schemas

3. **Integration**
   - Webhook notifications
   - Slack/Discord alerts
   - Export to data warehouses