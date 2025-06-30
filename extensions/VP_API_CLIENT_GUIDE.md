# MCP Registry /vp API - Client Integration Guide

## Overview

The `/vp` (v-plugged) API extends the MCP Registry with advanced filtering capabilities, allowing clients to efficiently query and filter MCP servers based on various criteria.

## Base URL

```
https://registry.plugged.in/vp
```

## Available Endpoints

### 1. List Servers with Filtering

**Endpoint:** `GET /vp/servers`

**Description:** Retrieve a paginated list of MCP servers with optional filtering.

#### Query Parameters

| Parameter | Type | Required | Description | Example |
|-----------|------|----------|-------------|---------|
| `cursor` | string (UUID) | No | Pagination cursor for next page | `cursor=123e4567-e89b-12d3-a456-426614174000` |
| `limit` | integer | No | Number of results per page (1-100, default: 30) | `limit=50` |
| `name` | string | No | Filter by exact server name | `name=sqlite` |
| `repository_url` | string | No | Filter by repository URL | `repository_url=https://github.com/user/repo` |
| `repository_source` | string | No | Filter by repository source | `repository_source=github` |
| `version` | string | No | Filter by specific version | `version=1.0.0` |
| `latest` | boolean | No | Filter to show only latest versions | `latest=true` |
| `package_registry` | string | No | Filter by package registry type | `package_registry=npm` |

#### Response Format

```json
{
  "servers": [
    {
      "id": "00613acb-73e2-4f93-8b96-296df17316c8",
      "name": "sqlite",
      "description": "MCP server for SQLite databases",
      "repository": {
        "url": "https://github.com/benborla/mcp-server-sqlite",
        "source": "github",
        "id": "benborla/mcp-server-sqlite"
      },
      "version_detail": {
        "version": "0.4.2",
        "release_date": "2024-11-27T10:00:00Z",
        "is_latest": true
      }
    }
  ],
  "metadata": {
    "next_cursor": "123e4567-e89b-12d3-a456-426614174000",
    "count": 30
  }
}
```

### 2. Get Server Details

**Endpoint:** `GET /vp/servers/{id}`

**Description:** Retrieve detailed information about a specific server.

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string (UUID) | Yes | The server's unique identifier |

#### Response Format

```json
{
  "id": "00613acb-73e2-4f93-8b96-296df17316c8",
  "name": "sqlite",
  "description": "MCP server for SQLite databases",
  "repository": {
    "url": "https://github.com/benborla/mcp-server-sqlite",
    "source": "github",
    "id": "benborla/mcp-server-sqlite"
  },
  "version_detail": {
    "version": "0.4.2",
    "release_date": "2024-11-27T10:00:00Z",
    "is_latest": true
  },
  "packages": [
    {
      "registry_name": "npm",
      "name": "@benborla/mcp-server-sqlite",
      "version": "0.4.2"
    }
  ],
  "remotes": []
}
```

## Example Usage

### JavaScript/TypeScript

```typescript
// Fetch servers by package registry
async function getServersByPackageRegistry(registry: 'npm' | 'docker' | 'pypi') {
  const response = await fetch(`https://registry.plugged.in/vp/servers?package_registry=${registry}`);
  const data = await response.json();
  return data.servers;
}

// Fetch latest SQLite servers
async function getLatestSQLiteServers() {
  const response = await fetch('https://registry.plugged.in/vp/servers?name=sqlite&latest=true');
  const data = await response.json();
  return data.servers;
}

// Fetch all GitHub-hosted servers with pagination
async function getGitHubServers(cursor?: string) {
  const url = new URL('https://registry.plugged.in/vp/servers');
  url.searchParams.set('repository_source', 'github');
  url.searchParams.set('limit', '50');
  if (cursor) {
    url.searchParams.set('cursor', cursor);
  }
  
  const response = await fetch(url.toString());
  return await response.json();
}

// Get server details
async function getServerDetails(serverId: string) {
  const response = await fetch(`https://registry.plugged.in/vp/servers/${serverId}`);
  return await response.json();
}
```

### Python

```python
import requests

# Fetch latest servers
def get_latest_servers():
    response = requests.get('https://registry.plugged.in/vp/servers', params={
        'latest': 'true',
        'limit': 50
    })
    return response.json()

# Search by name
def search_servers_by_name(name):
    response = requests.get('https://registry.plugged.in/vp/servers', params={
        'name': name
    })
    return response.json()

# Get server details
def get_server_details(server_id):
    response = requests.get(f'https://registry.plugged.in/vp/servers/{server_id}')
    return response.json()
```

### cURL

```bash
# Get all npm packages
curl "https://registry.plugged.in/vp/servers?package_registry=npm"

# Get all docker packages
curl "https://registry.plugged.in/vp/servers?package_registry=docker"

# Get all pypi packages
curl "https://registry.plugged.in/vp/servers?package_registry=pypi"

# Get all latest versions
curl "https://registry.plugged.in/vp/servers?latest=true"

# Filter by name
curl "https://registry.plugged.in/vp/servers?name=sqlite"

# Filter by repository source with increased limit
curl "https://registry.plugged.in/vp/servers?repository_source=github&limit=100"

# Combine multiple filters
curl "https://registry.plugged.in/vp/servers?repository_source=github&latest=true&limit=50"

# Get server details
curl "https://registry.plugged.in/vp/servers/00613acb-73e2-4f93-8b96-296df17316c8"
```

## Integration Best Practices

### 1. Pagination Handling

Always check for `metadata.next_cursor` in the response to handle pagination:

```typescript
async function getAllServers() {
  const allServers = [];
  let cursor = null;
  
  do {
    const url = new URL('https://registry.plugged.in/vp/servers');
    if (cursor) url.searchParams.set('cursor', cursor);
    
    const response = await fetch(url.toString());
    const data = await response.json();
    
    allServers.push(...data.servers);
    cursor = data.metadata?.next_cursor;
  } while (cursor);
  
  return allServers;
}
```

### 2. Error Handling

```typescript
async function fetchServers(filters: Record<string, string>) {
  try {
    const url = new URL('https://registry.plugged.in/vp/servers');
    Object.entries(filters).forEach(([key, value]) => {
      url.searchParams.set(key, value);
    });
    
    const response = await fetch(url.toString());
    
    if (!response.ok) {
      throw new Error(`API error: ${response.status} ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Failed to fetch servers:', error);
    throw error;
  }
}
```

### 3. Caching Strategy

Consider implementing client-side caching for better performance:

```typescript
const cache = new Map();
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

async function getCachedServers(filters: Record<string, string>) {
  const cacheKey = JSON.stringify(filters);
  const cached = cache.get(cacheKey);
  
  if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
    return cached.data;
  }
  
  const data = await fetchServers(filters);
  cache.set(cacheKey, { data, timestamp: Date.now() });
  return data;
}
```

## Rate Limiting

The API currently does not enforce rate limits, but clients should implement reasonable request throttling to avoid overloading the server.

## Future Enhancements

The following features are planned for future releases:

1. **Sorting Options**: `?sort=name|downloads|rating|created|updated`
2. **Partial Name Matching**: `?name_contains=sql`
3. **Multiple Value Filters**: `?repository_source=github,gitlab`
4. **Date Range Filters**: `?created_after=2024-01-01`

## Support

For issues or questions about the API, please contact the registry maintainers or open an issue in the repository.