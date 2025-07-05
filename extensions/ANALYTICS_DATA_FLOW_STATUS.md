# Analytics Data Flow Status

## Date: 2025-07-05

## Current Status

### Working ✅
1. **VP Analytics API Endpoints**
   - `/vp/analytics/dashboard` - Returns dashboard metrics
   - `/vp/analytics/activity` - Returns activity events  
   - `/vp/analytics/growth` - Returns growth metrics
   - All endpoints properly accept both UUID and registry ID formats

2. **MongoDB Collections**
   - `server_stats` - Server statistics (20 documents)
   - `analytics_metrics` - Analytics metrics collection
   - `activity_events` - Activity events collection
   - `feedback` - User feedback and ratings

3. **Stats Database Interface**
   - `GetAllStats()` method implemented successfully
   - Returns all server stats for sync purposes

### Not Working ❌
1. **Analytics Data Sync**
   - Events are tracked to Elasticsearch at analytics.plugged.in
   - VP API reads from MongoDB (no events present)
   - No sync service exists to move data from Elasticsearch to MongoDB

2. **Analytics Sync Service**
   - Currently tries to sync FROM MongoDB TO external analytics
   - Should sync FROM Elasticsearch TO MongoDB
   - Returns "No active servers found for sync" (fixed, but wrong direction)

## Root Cause

The analytics sync service (`stats.SyncService`) is designed to:
1. Read server IDs from MongoDB
2. Fetch analytics data from an external analytics API
3. Update MongoDB with the fetched data

However:
- The external analytics URL (`MCP_REGISTRY_ANALYTICS_URL`) was set to a non-existent service
- Even if it existed, we need the opposite flow: Elasticsearch → MongoDB

## Solutions

### Option 1: Quick Fix (If Analytics API Exists)
If analytics.plugged.in exposes an API that returns server metrics:

```bash
# Set the analytics URL
export MCP_REGISTRY_ANALYTICS_URL=https://analytics.plugged.in

# Restart the service
docker compose -f docker-compose.yml -f docker-compose.extended-override.yml restart registry
```

### Option 2: Create Elasticsearch Sync Service
Create a new service that:
1. Connects to Elasticsearch at analytics.plugged.in
2. Queries for events and metrics
3. Transforms and stores data in MongoDB collections

### Option 3: Direct MongoDB Population
For testing, manually populate MongoDB with sample data:

```javascript
// Connect to MongoDB
docker exec -it mongodb mongosh mcp-registry

// Insert sample activity events
db.activity_events.insertMany([
  {
    server_id: "postgres-tools",
    event_type: "installation",
    timestamp: new Date(),
    user_id: "user123",
    metadata: { version: "1.0.0" }
  },
  {
    server_id: "bd554881-d64c-45be-a05e-49f7b802d4d8",
    event_type: "usage",
    timestamp: new Date(),
    duration_ms: 5000
  }
])

// Insert sample analytics metrics
db.analytics_metrics.insertOne({
  server_id: "postgres-tools",
  period: "2025-07",
  total_installs: 150,
  api_calls: 3500,
  unique_users: 75,
  avg_response_time_ms: 120
})
```

## Verification

1. **Check sync service logs**:
```bash
docker logs registry-extended | grep -i sync
```

2. **Check MongoDB collections**:
```bash
docker exec -it mongodb mongosh mcp-registry --eval "
  print('Activity Events:', db.activity_events.countDocuments());
  print('Analytics Metrics:', db.analytics_metrics.countDocuments());
  print('Server Stats:', db.server_stats.countDocuments());
"
```

3. **Test analytics dashboard**:
```bash
curl https://registry.plugged.in/vp/analytics/dashboard
```

## Next Steps

1. **Immediate**: Determine if analytics.plugged.in has an API endpoint
2. **Short-term**: Create Elasticsearch → MongoDB sync service if needed
3. **Long-term**: Consider unified data pipeline architecture

## Notes

- The VP Analytics code is fully functional and ready
- All server ID formats (UUID and registry ID) are properly supported
- The only missing piece is populating MongoDB with event data from Elasticsearch