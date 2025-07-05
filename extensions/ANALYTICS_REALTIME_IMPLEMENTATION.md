# Real-Time Analytics Implementation

## Date: 2025-07-05

## Overview

The MCP Registry VP analytics system has been refactored to provide real-time data by directly querying the analytics API instead of using a background sync service with 15-minute delays.

## Architecture Changes

### Before (Sync-based)
```
Elasticsearch → Analytics API → Sync Service (15 min) → MongoDB → VP Dashboard
```

### After (Real-time)
```
Elasticsearch → Analytics API → VP Dashboard (with MongoDB fallback)
```

## Key Changes Made

### 1. Removed Sync Service
- Deleted sync service initialization from `router.go`
- Removed `SyncService` struct and background sync logic
- Kept only `HTTPAnalyticsClient` for direct API calls

### 2. Refactored Code Structure
- Created `analytics_client.go` - HTTP client for analytics API
- Created `cache.go` - Caching service (extracted from sync.go)
- Removed `sync.go` - No longer needed

### 3. Updated Handlers
Both dashboard and activity feed handlers now:
1. Try analytics API first (if client available)
2. Fall back to MongoDB on API failure
3. Log fallback attempts for debugging

### 4. Environment Configuration
```bash
# Analytics API configuration
MCP_REGISTRY_ANALYTICS_URL=https://plugged.in/api/analytics
MCP_REGISTRY_ANALYTICS_USER=
MCP_REGISTRY_ANALYTICS_PASS=
```

## Benefits

1. **Real-time Data**: No 15-minute sync delay
2. **Simpler Architecture**: No background processes
3. **Better Reliability**: Graceful fallback to MongoDB
4. **Cleaner Code**: Single responsibility for each component

## API Integration

The analytics client expects these endpoints from pluggedin-app:

### Dashboard Metrics
```
GET /api/analytics/dashboard?period={day|week|month}
```

### Activity Feed
```
GET /api/analytics/events/recent?limit={number}
```

### Server Stats
```
GET /api/analytics/servers/{id}/stats
POST /api/analytics/servers/stats/batch
```

## Fallback Behavior

When the analytics API is unavailable (404, network error, etc.):
1. Handler logs the error: "Analytics API failed, falling back to MongoDB"
2. Queries MongoDB collections for cached/local data
3. Returns best available data to the user

## MongoDB Usage

MongoDB now stores only:
- **Local data**: User feedback, ratings, comments
- **Server metadata**: Registration info, claimed status
- **Cached data**: For performance optimization

Analytics data (installs, usage, activity) comes directly from the analytics API.

## Testing

```bash
# Test VP dashboard (should work even if analytics API is down)
curl https://registry.plugged.in/vp/analytics/dashboard

# Check logs for fallback behavior
docker logs registry-extended | grep "Analytics API failed"

# Verify no sync service is running
docker logs registry-extended | grep -i sync
```

## Future Improvements

1. Implement circuit breaker for analytics API
2. Add metrics for API success/failure rates
3. Consider read-through cache pattern
4. Add analytics API health check endpoint