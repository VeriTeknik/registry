# Deploying Registry with Stats Extension

## Quick Start

To deploy the registry with stats extension:

```bash
# Build and start the extended registry
./scripts/start-extended.sh
```

## Manual Deployment

### 1. Build the Extended Image

```bash
docker build -t registry-extended:latest -f Dockerfile .
```

### 2. Start Services

```bash
docker compose -f docker-compose.yml -f docker-compose.extended-override.yml up -d
```

### 3. Verify Deployment

```bash
# Check service status
docker compose -f docker-compose.yml -f docker-compose.extended-override.yml ps

# View logs
docker compose -f docker-compose.yml -f docker-compose.extended-override.yml logs -f registry
```

## Environment Variables

The stats extension supports these additional environment variables:

- `MCP_REGISTRY_ANALYTICS_URL`: URL to analytics API for syncing metrics (default: http://analytics-api:8081)

## What Gets Deployed

1. **Extended Registry Service**
   - Includes all base registry functionality
   - Adds `/vp/*` endpoints for stats operations
   - Runs stats migration on startup
   - Syncs with analytics service if configured

2. **MongoDB**
   - Stores registry data in `servers_v2` collection
   - Stores stats data in `server_stats` collection
   - Proper indexes for performance

3. **Traefik Integration**
   - Routes `registry.plugged.in` to the extended service
   - Handles SSL/TLS termination
   - No port exposure needed

## Verifying Stats Extension

After deployment, test the stats endpoints:

```bash
# Check if stats endpoints are available
curl https://registry.plugged.in/vp/stats/global

# Track an installation
curl -X POST https://registry.plugged.in/vp/servers/{server-id}/install \
  -H "Content-Type: application/json" \
  -d '{"source": "REGISTRY"}'

# Submit a rating
curl -X POST https://registry.plugged.in/vp/servers/{server-id}/rate \
  -H "Content-Type: application/json" \
  -d '{"rating": 5, "source": "REGISTRY"}'

# Get sorted servers
curl https://registry.plugged.in/vp/servers?sort=installs&limit=10
```

## Monitoring

Check MongoDB for stats data:

```bash
# Connect to MongoDB
docker exec -it registry_mongodb_1 mongosh

# Check stats collection
use mcp_registry
db.server_stats.find().limit(5)
db.server_stats.getIndexes()
```

## Troubleshooting

1. **Stats endpoints return 404**
   - Ensure you're using the extended image
   - Check that the container is running the extended binary
   - Verify logs for startup errors

2. **Stats not being tracked**
   - Check MongoDB connection in logs
   - Verify stats collection exists
   - Check for migration errors on startup

3. **Analytics sync not working**
   - Verify analytics service is running
   - Check `MCP_REGISTRY_ANALYTICS_URL` is set correctly
   - Look for sync errors in logs

## Rolling Back

To rollback to the standard registry without stats:

```bash
# Stop services
docker compose -f docker-compose.yml -f docker-compose.extended-override.yml down

# Start standard registry
docker compose up -d
```