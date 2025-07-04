# MCP Registry Extensions

This directory contains extensions to the MCP Registry that add additional functionality without modifying the upstream codebase.

## /vp API Extension

The `/vp` (v-plugged) API provides enhanced filtering capabilities and statistics tracking for the registry endpoints.

### Endpoints

#### GET /vp/servers

Lists servers with filtering support.

**Query Parameters:**
- `cursor` (optional): UUID for pagination
- `limit` (optional): Number of results (1-100, default 30)
- `name` (optional): Filter by exact server name
- `repository_url` (optional): Filter by repository URL
- `repository_source` (optional): Filter by repository source (e.g., "github")
- `version` (optional): Filter by specific version
- `latest` (optional): Filter by latest versions only (true/false)
- `package_registry` (optional): Filter by package registry type (npm, docker, pypi, etc.)

**Examples:**
```bash
# Get all SQLite servers
GET /vp/servers?name=sqlite

# Get all npm packages
GET /vp/servers?package_registry=npm

# Get all docker packages
GET /vp/servers?package_registry=docker

# Get latest versions only
GET /vp/servers?latest=true

# Get servers from GitHub
GET /vp/servers?repository_source=github

# Combine filters
GET /vp/servers?repository_source=github&latest=true&limit=50
```

#### GET /vp/servers/{id}

Get detailed information about a specific server (same as v0).

### Running the Extended Registry

#### Using Docker Compose:
```bash
docker compose -f docker-compose-extended.yml up --build
```

#### Using Go:
```bash
go run extensions/main_with_extensions.go
```

### Implementation Notes

1. **No Upstream Modifications**: All code is contained in the `/extensions` directory
2. **Memory Filtering**: Currently filters are applied in memory after fetching from database (not optimal for large datasets)
3. **Database Support**: The database layer supports filtering, but the service layer doesn't expose it
4. **Package Registry Filter**: Special handling for `package_registry` filter since it requires checking the full server details
5. **Future Improvements**: Could implement direct database filtering by extending the service interface

### Adding New Filters

To add new filters, modify the `buildFilters` function in `/extensions/vp/handlers/servers.go`:

```go
// Example: Add description filtering
if descriptions, ok := queryParams["description"]; ok && len(descriptions) > 0 {
    filters["description"] = descriptions[0]
}
```

## Stats Extension

The stats extension adds installation tracking, ratings, and analytics integration to the registry.

### Stats Endpoints

#### POST /vp/servers/{id}/install
Track an installation for a server.

**Request Body (optional):**
```json
{
  "user_id": "user123",
  "version": "1.2.0",
  "platform": "macos",
  "timestamp": 1234567890
}
```

#### POST /vp/servers/{id}/rate
Submit a rating for a server.

**Request Body:**
```json
{
  "rating": 4.5
}
```

#### GET /vp/servers/{id}/stats
Get statistics for a specific server.

**Response:**
```json
{
  "stats": {
    "server_id": "example-server",
    "installation_count": 1234,
    "rating": 4.5,
    "rating_count": 78,
    "active_installs": 890,
    "daily_active_users": 456,
    "monthly_active_users": 789
  }
}
```

#### GET /vp/stats/global
Get global registry statistics.

**Response:**
```json
{
  "total_servers": 250,
  "total_installs": 50000,
  "active_servers": 180,
  "average_rating": 4.2
}
```

#### GET /vp/stats/leaderboard
Get leaderboard data.

**Query Parameters:**
- `type`: Leaderboard type (installs, rating, trending)
- `limit`: Number of results (default 10, max 100)

#### GET /vp/stats/trending
Get trending servers.

**Query Parameters:**
- `limit`: Number of results (default 20, max 100)

### Server Claiming

#### POST /vp/servers/{id}/claim
Claim a community server and transfer its stats.

**Request Body:**
```json
{
  "publish_request": {
    "name": "My Server",
    "description": "Server description",
    "repository": {
      "owner": "myusername",
      "name": "myrepo"
    },
    "schema_version": "1.0.0",
    "install_type": "npm",
    "install_url": "mypackage",
    "transport": ["stdio"]
  },
  "transfer_stats": true
}
```

### Enhanced Server Responses

All `/vp/servers` endpoints now include statistics in their responses:

```json
{
  "servers": [
    {
      "id": "example-server",
      "name": "Example Server",
      "description": "...",
      "repository": {...},
      "version_detail": {...},
      "installation_count": 1234,
      "rating": 4.5,
      "rating_count": 78,
      "active_installs": 890,
      "weekly_growth": 12.5
    }
  ]
}
```

### Architecture

```
extensions/
├── stats/
│   ├── model.go                # Stats data models
│   ├── database.go             # MongoDB operations for stats
│   └── sync.go                 # Analytics sync service
├── vp/
│   ├── model/
│   │   ├── extended_server.go  # Server model with stats
│   │   └── claim.go            # Claim request models
│   ├── handlers/
│   │   ├── servers.go          # Enhanced server endpoints
│   │   ├── stats.go            # Stats-specific endpoints
│   │   └── claim.go            # Server claiming endpoint
│   └── router.go               # Route registration
├── router_with_vp.go           # Extended router setup
├── main_with_extensions.go     # Extended main entry point
├── Dockerfile                  # Docker build for extended version
└── README.md                   # This file
```

### Database Schema

The stats extension uses a separate `server_stats` collection in MongoDB:

```javascript
{
  "server_id": "unique-server-id",
  "installation_count": 1234,
  "rating": 4.5,
  "rating_count": 78,
  "active_installs": 890,
  "daily_active_users": 456,
  "monthly_active_users": 789,
  "last_updated": "2024-01-01T00:00:00Z"
}
```

### Analytics Integration

The stats extension integrates with the analytics service at `https://analytics.plugged.in`:
- Syncs active user metrics every 15 minutes
- Provides real-time installation and rating tracking
- Calculates trending servers based on growth metrics

### Caching

Stats responses are cached for 5 minutes using an in-memory cache:
- Individual server stats
- Global statistics
- Leaderboards
- Trending servers

Cache is automatically invalidated when stats are updated.