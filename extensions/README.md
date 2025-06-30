# MCP Registry Extensions

This directory contains extensions to the MCP Registry that add additional functionality without modifying the upstream codebase.

## /vp API Extension

The `/vp` (v-plugged) API provides enhanced filtering capabilities for the registry endpoints.

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

### Architecture

```
extensions/
├── vp/
│   ├── handlers/
│   │   ├── servers.go          # Filtering logic
│   │   └── service_extension.go # Service layer extensions
│   └── router.go               # Route registration
├── router_with_vp.go           # Extended router
├── main_with_extensions.go     # Extended main entry point
├── Dockerfile                  # Docker build for extended version
└── README.md                   # This file
```