# Stats Extension Implementation Summary

## Overview
This document summarizes the implementation of the stats extension system for the MCP Registry, which adds community server support with source tracking.

## Key Features Implemented

### 1. Source-Based Stats Tracking
- Added `Source` field to `ServerStats` model with two possible values:
  - `REGISTRY`: For servers published through the registry
  - `COMMUNITY`: For community-contributed servers
- Implemented compound key indexing on `(server_id, source)` for unique stats per source
- All database operations now support source-aware queries

### 2. Stats Endpoints with Source Support

#### Installation Tracking
- **Endpoint**: `POST /vp/servers/{id}/install`
- **Body**: 
  ```json
  {
    "source": "REGISTRY" | "COMMUNITY",  // optional, defaults to REGISTRY
    "user_id": "string",
    "version": "string",
    "platform": "string"
  }
  ```

#### Rating Submission
- **Endpoint**: `POST /vp/servers/{id}/rate`
- **Body**:
  ```json
  {
    "rating": 1-5,
    "source": "REGISTRY" | "COMMUNITY"  // optional, defaults to REGISTRY
  }
  ```

#### Stats Retrieval
- **Endpoint**: `GET /vp/servers/{id}/stats`
- **Query Parameters**:
  - `source`: Filter by source (REGISTRY/COMMUNITY)
  - `aggregated=true`: Get combined stats from all sources

#### Global Stats
- **Endpoint**: `GET /vp/stats/global`
- **Query Parameters**:
  - `source`: Filter by source (REGISTRY/COMMUNITY/ALL)

#### Leaderboards
- **Endpoint**: `GET /vp/stats/leaderboard`
- **Query Parameters**:
  - `type`: installs/rating/trending
  - `source`: Filter by source (REGISTRY/COMMUNITY/ALL)
  - `limit`: Number of results (default: 10)

### 3. Server Listing with Sorting
- **Endpoint**: `GET /vp/servers`
- **Query Parameters**:
  - `sort`: Sort by installs/rating/trending
  - `source`: Filter by source (REGISTRY/COMMUNITY)
  - `limit`: Number of results (default: 100, max: 1000)

### 4. Stats Transfer During Claiming
- **Endpoint**: `POST /vp/servers/{id}/claim`
- When `transfer_stats: true` is set, stats are transferred from COMMUNITY to REGISTRY source
- Original community stats are preserved with claim metadata for audit trail

### 5. Database Migration
- Automatic migration adds `source: "REGISTRY"` to all existing stats entries
- Runs on startup to ensure backward compatibility

## Implementation Details

### Database Schema
```javascript
{
  server_id: "string",
  source: "REGISTRY" | "COMMUNITY",
  installation_count: number,
  rating: number,
  rating_count: number,
  active_installs: number,
  daily_active_users: number,
  monthly_active_users: number,
  last_updated: Date,
  // Claim tracking
  claimed_from: "string",
  claimed_at: Date
}
```

### Indexes
1. Compound unique index: `(server_id, source)`
2. Single field indexes: `server_id`, `source`
3. Compound sorting indexes: `(source, installation_count)`, `(source, rating)`

### Caching Strategy
- In-memory cache with 5-minute TTL
- Cache invalidation on stats updates
- Separate cache keys for different sources and aggregated views

## API Examples

### Track Installation for Community Server
```bash
curl -X POST https://registry.plugged.in/vp/servers/my-server/install \
  -H "Content-Type: application/json" \
  -d '{"source": "COMMUNITY"}'
```

### Get Aggregated Stats
```bash
curl https://registry.plugged.in/vp/servers/my-server/stats?aggregated=true
```

### Get Top Rated Registry Servers
```bash
curl https://registry.plugged.in/vp/stats/leaderboard?type=rating&source=REGISTRY&limit=20
```

### List Servers Sorted by Installs
```bash
curl https://registry.plugged.in/vp/servers?sort=installs&limit=50
```

## Future Enhancements
1. Add tests for source tracking and sorting functionality
2. Implement proper trending algorithm based on growth rate
3. Add more detailed analytics integration
4. Support for additional sources beyond REGISTRY and COMMUNITY