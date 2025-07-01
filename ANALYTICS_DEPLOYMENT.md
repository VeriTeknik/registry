# MCP Registry Analytics - Deployment Guide

## Current Status

The analytics infrastructure is now deployed with the following services:

### ✅ Running Services

1. **Elasticsearch** (http://localhost:9200)
   - Status: Healthy ✓
   - Purpose: Analytics data storage and search
   - Container: `analytics-elasticsearch`

2. **Kibana** (http://localhost:5601)
   - Status: Running ✓
   - Purpose: Data visualization and dashboards
   - Container: `analytics-kibana`
   - Access via Traefik: https://kibana.plugged.in

3. **Redis** (localhost:6379)
   - Status: Healthy ✓
   - Purpose: Caching and real-time counters
   - Container: `analytics-redis`

### 🚧 Services Ready to Deploy

4. **MongoDB Sync Service**
   - Status: Built, not running
   - Purpose: Syncs server data from MongoDB to Elasticsearch
   - Start command: `docker compose -f analytics/docker-compose.yml up -d sync-service`

5. **Analytics API**
   - Status: Built, not running
   - Purpose: REST API for event tracking and analytics queries
   - Start command: `docker compose -f analytics/docker-compose.yml up -d analytics-api`
   - Will be available at: https://analytics.plugged.in

## Quick Start Commands

### View Service Status
```bash
# Check all analytics services
docker ps | grep analytics

# View logs
docker logs analytics-elasticsearch
docker logs analytics-kibana
docker logs analytics-redis
```

### Access Services

1. **Kibana Dashboard**
   - Local: http://localhost:5601
   - Production: https://kibana.plugged.in
   - Username/Password: Not required (security disabled for development)

2. **Elasticsearch**
   - Health check: `curl http://localhost:9200/_cluster/health?pretty`
   - Indices: `curl http://localhost:9200/_cat/indices?v`

3. **Redis**
   - Test connection: `redis-cli ping`

## Next Steps

### 1. Initialize Elasticsearch Indices
```bash
cd analytics/elasticsearch
./init-indices.sh
```

### 2. Start Sync Service
```bash
docker compose -f analytics/docker-compose.yml up -d sync-service
```
This will:
- Connect to MongoDB
- Perform initial sync of all servers
- Start watching for changes

### 3. Start Analytics API
```bash
docker compose -f analytics/docker-compose.yml up -d analytics-api
```
This will:
- Start REST API on port 8081
- Connect to Elasticsearch and Redis
- Enable event tracking endpoints

### 4. Configure Kibana Dashboards
1. Open Kibana: http://localhost:5601
2. Go to Stack Management → Index Patterns
3. Create patterns for:
   - `servers*`
   - `events*`
   - `metrics*`
   - `feedback*`

## API Integration

Once the Analytics API is running, you can start tracking events:

```javascript
// Track an install
fetch('https://analytics.plugged.in/api/v1/track', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    event_type: 'install',
    server_id: 'server-uuid',
    client_id: 'client-uuid',
    metadata: {
      version: '1.0.0',
      platform: 'macos'
    }
  })
});

// Get server stats
fetch('https://analytics.plugged.in/api/v1/servers/server-uuid/stats')
  .then(res => res.json())
  .then(stats => console.log(stats));
```

## Monitoring

### Health Checks
- Elasticsearch: http://localhost:9200/_cluster/health
- Kibana: http://localhost:5601/api/status
- Analytics API: http://localhost:8081/api/v1/health

### Resource Usage
```bash
docker stats analytics-elasticsearch analytics-kibana analytics-redis
```

### Logs
```bash
# Follow all analytics logs
docker compose -f analytics/docker-compose.yml logs -f
```

## Troubleshooting

### Elasticsearch Issues
```bash
# Check cluster health
curl http://localhost:9200/_cluster/health?pretty

# Check node info
curl http://localhost:9200/_nodes?pretty

# Increase heap size if needed
# Edit docker-compose.yml: ES_JAVA_OPTS=-Xms2g -Xmx2g
```

### Kibana Connection Issues
```bash
# Check Kibana logs
docker logs analytics-kibana

# Verify Elasticsearch is reachable
docker exec analytics-kibana curl http://elasticsearch:9200
```

### Network Issues
```bash
# List networks
docker network ls

# Inspect analytics network
docker network inspect analytics_analytics
```

## Production Considerations

1. **Security**
   - Enable Elasticsearch security features
   - Configure SSL/TLS for all services
   - Set up authentication for Kibana
   - Restrict Redis access

2. **Performance**
   - Increase Elasticsearch heap size
   - Configure index lifecycle policies
   - Set up Redis persistence
   - Use dedicated nodes for production

3. **Backup**
   - Schedule Elasticsearch snapshots
   - Export Kibana dashboards
   - Backup Redis data

4. **Monitoring**
   - Set up Elastic APM
   - Configure alerts
   - Monitor disk usage
   - Track query performance