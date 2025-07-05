# Analytics API Implementation Guide for pluggedin-app

## Date: 2025-07-05

This guide outlines the API endpoints that need to be implemented in pluggedin-app to enable analytics data sync with the MCP Registry.

## Required Endpoints

### 1. Get Server Metrics
**Endpoint:** `GET /api/v1/servers/{id}/stats`

**Purpose:** Fetch analytics metrics for a single server

**Expected Response:**
```json
{
  "server_id": "postgres-tools",
  "active_installs": 150,
  "daily_active_users": 45,
  "monthly_active_users": 120,
  "weekly_growth": 0.15,
  "last_updated": "2025-07-05T12:00:00Z"
}
```

**Implementation Notes:**
- Query Elasticsearch for events with matching `server_id`
- Accept both UUID format (e.g., `bd554881-d64c-45be-a05e-49f7b802d4d8`) and registry ID format (e.g., `postgres-tools`)
- Calculate metrics from event data:
  - `active_installs`: Count of unique active installations
  - `daily_active_users`: Unique users in last 24 hours
  - `monthly_active_users`: Unique users in last 30 days
  - `weekly_growth`: Percentage growth compared to previous week
- Return 200 OK with JSON response
- Return 404 if server not found

### 2. Batch Server Metrics (Optional but Recommended)
**Endpoint:** `GET /api/v1/servers/stats/batch`

**Purpose:** Fetch metrics for multiple servers efficiently

**Request:**
```json
{
  "server_ids": ["postgres-tools", "bd554881-d64c-45be-a05e-49f7b802d4d8", ...]
}
```

**Expected Response:**
```json
{
  "postgres-tools": {
    "server_id": "postgres-tools",
    "active_installs": 150,
    "daily_active_users": 45,
    "monthly_active_users": 120,
    "weekly_growth": 0.15,
    "last_updated": "2025-07-05T12:00:00Z"
  },
  "bd554881-d64c-45be-a05e-49f7b802d4d8": {
    "server_id": "bd554881-d64c-45be-a05e-49f7b802d4d8",
    "active_installs": 75,
    "daily_active_users": 20,
    "monthly_active_users": 60,
    "weekly_growth": 0.08,
    "last_updated": "2025-07-05T12:00:00Z"
  }
}
```

## Authentication

The registry sync service supports Basic Authentication:

```http
Authorization: Basic base64(username:password)
```

The credentials are read from environment variables:
- `ANALYTICS_API_USERNAME`
- `ANALYTICS_API_PASSWORD`

## Elasticsearch Queries

### Example: Active Installs Count
```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "server_id": "postgres-tools" } },
        { "term": { "event_type": "installation" } },
        { "range": { "timestamp": { "gte": "now-30d" } } }
      ]
    }
  },
  "aggs": {
    "unique_installs": {
      "cardinality": {
        "field": "installation_id"
      }
    }
  }
}
```

### Example: Daily Active Users
```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "server_id": "postgres-tools" } },
        { "range": { "timestamp": { "gte": "now-1d" } } }
      ]
    }
  },
  "aggs": {
    "unique_users": {
      "cardinality": {
        "field": "user_id"
      }
    }
  }
}
```

## Testing the Integration

1. **Test Individual Endpoint:**
```bash
curl -u username:password https://plugged.in/api/v1/servers/postgres-tools/stats
```

2. **Monitor Sync Service:**
```bash
docker logs registry-extended -f | grep -i sync
```

3. **Verify MongoDB Update:**
```bash
docker exec mongodb mongosh mcp-registry --eval "db.server_stats.find({server_id: 'postgres-tools'}).pretty()"
```

## Sync Service Behavior

- Runs every 15 minutes
- Fetches all servers from MongoDB's `server_stats` collection
- For each server, calls the analytics API to get updated metrics
- Updates only the analytics fields (preserves installation counts and ratings from other sources)
- Logs successes and failures

## Error Handling

- Return proper HTTP status codes
- 200 OK for success
- 404 Not Found if server doesn't exist
- 401 Unauthorized for auth failures
- 500 Internal Server Error for server errors
- Include error messages in response body

## Performance Considerations

- The sync service processes servers with concurrency limit of 10
- Implement caching if needed to handle frequent requests
- Consider implementing the batch endpoint to reduce API calls
- Use Elasticsearch aggregations efficiently

## Next Steps

1. Implement the endpoints in pluggedin-app
2. Test with curl to verify response format
3. Configure registry with analytics URL and credentials
4. Monitor sync service logs
5. Verify data appears in VP dashboard