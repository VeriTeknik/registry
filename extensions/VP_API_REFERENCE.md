# VP (v-plugged) API Reference

## Version: 1.0.0
Last Updated: 2025-07-05

## Overview

The VP API provides enhanced statistics and analytics for the MCP Registry. This document is the authoritative reference for all VP endpoints.

**Base URL**: `https://registry.plugged.in/vp`

## Table of Contents

1. [Authentication](#authentication)
2. [Server Statistics](#server-statistics)
3. [Analytics Dashboard](#analytics-dashboard)
4. [Feedback System](#feedback-system)
5. [Activity & Discovery](#activity--discovery)
6. [Error Handling](#error-handling)
7. [Rate Limiting](#rate-limiting)

## Authentication

Most read endpoints are public. Write operations require GitHub authentication:

| Endpoint | Method | Auth Required |
|----------|---------|--------------|
| `/servers/*` | GET | No |
| `/servers/{id}/install` | POST | No |
| `/servers/{id}/rate` | POST | **Yes** |
| `/servers/{id}/claim` | POST | **Yes** |
| `/analytics/*` | GET | No |
| `/admin/*` | GET | **Yes** |

**Authentication Header**:
```
Authorization: Bearer YOUR_GITHUB_TOKEN
```

## Server Statistics

### Get All Servers with Stats
```
GET /vp/servers
```

**Query Parameters**:
- `limit` (int): 1-100, default: 20
- `offset` (int): For pagination, default: 0
- `category` (string): Filter by category
- `featured` (boolean): Show only featured servers
- `source` (string): `REGISTRY` | `COMMUNITY` | `ALL`

**Response**:
```json
{
  "servers": [
    {
      "id": "postgres-tools",
      "name": "PostgreSQL Tools",
      "description": "Comprehensive PostgreSQL management",
      "category": "database",
      "repository_url": "https://github.com/user/repo",
      "author": {
        "name": "John Doe",
        "github": "johndoe"
      },
      "stats": {
        "install_count": 5234,
        "average_rating": 4.8,
        "rating_count": 123
      }
    }
  ],
  "total": 342,
  "limit": 20,
  "offset": 0,
  "has_more": true
}
```

### Get Server Statistics
```
GET /vp/servers/{server_id}/stats
```

**Query Parameters**:
- `source` (string): `REGISTRY` | `COMMUNITY` | `ALL` (default: `REGISTRY`)
- `aggregated` (boolean): Combine stats from all sources

**Response**:
```json
{
  "server_id": "postgres-tools",
  "source": "REGISTRY",
  "install_count": 5234,
  "average_rating": 4.8,
  "rating_count": 123,
  "view_count": 12456,
  "daily_active_users": 432,
  "weekly_growth_rate": 12.5,
  "last_updated": "2025-07-05T12:34:56Z"
}
```

### Track Installation
```
POST /vp/servers/{server_id}/install
```

**Request Body**:
```json
{
  "platform": "vscode",    // Required: vscode, web, cli, etc.
  "version": "1.2.3",      // Optional: version installed
  "source": "marketplace"  // Optional: installation source
}
```

**Response**:
```json
{
  "success": true,
  "install_count": 5235
}
```

## Analytics Dashboard

### Dashboard Overview
```
GET /vp/analytics/dashboard
```

**Query Parameters**:
- `period` (string): `day` | `week` | `month` | `year` (default: `day`)

**Response**:
```json
{
  "total_installs": {
    "value": 123456,
    "trend": 12.5,
    "trend_direction": "up",
    "comparison_period": "vs yesterday"
  },
  "total_api_calls": {
    "value": 892341,
    "trend": 8.3,
    "trend_direction": "up",
    "comparison_period": "vs yesterday"
  },
  "active_users": {
    "value": 4521,
    "trend": -2.1,
    "trend_direction": "down",
    "comparison_period": "vs yesterday"
  },
  "server_health": {
    "value": "99.9%",
    "trend": 0.1,
    "trend_direction": "stable",
    "comparison_period": "vs yesterday"
  },
  "new_servers_today": 12,
  "install_velocity": 234.5,
  "top_rated_count": 89,
  "search_success_rate": 76.3,
  "install_trend": [120, 145, 132, 156, 189, 201, 234]
}
```

### Growth Metrics
```
GET /vp/analytics/growth
```

**Query Parameters**:
- `metric` (string, required): `installs` | `users` | `api_calls` | `servers` | `ratings` | `searches`
- `period` (string, required): `day` | `week` | `month` | `year`

**Response**:
```json
{
  "metric": "installs",
  "period": "week",
  "current_period_start": "2025-06-29T00:00:00Z",
  "previous_period_start": "2025-06-22T00:00:00Z",
  "current_value": 8234,
  "previous_value": 6123,
  "absolute_change": 2111,
  "growth_rate": 34.5,
  "momentum": 12.3,
  "trend": "accelerating",
  "data_points": [
    {"timestamp": "2025-06-29T00:00:00Z", "value": 1023},
    {"timestamp": "2025-06-30T00:00:00Z", "value": 1156}
  ]
}
```

### Activity Feed
```
GET /vp/analytics/activity
```

**Query Parameters**:
- `limit` (int): 1-100 (default: 20)
- `type` (string): `install` | `rating` | `search` | `server_added`

**Response**:
```json
{
  "activity": [
    {
      "id": "evt_123",
      "type": "install",
      "timestamp": "2025-07-05T12:34:56Z",
      "server_id": "postgres-tools",
      "server_name": "PostgreSQL Tools",
      "metadata": {
        "platform": "vscode",
        "version": "1.2.3"
      }
    }
  ],
  "count": 20
}
```

### Hot/Trending Servers
```
GET /vp/analytics/hot
```

**Query Parameters**:
- `limit` (int): 1-50 (default: 10)

**Response**:
```json
{
  "servers": [
    {
      "server_id": "new-ai-tool",
      "server_name": "AI Assistant Pro",
      "install_velocity": 145.2,
      "momentum_change": 280.5,
      "trend_category": "viral",
      "stats": {
        "installs_24h": 3489,
        "installs_prev_24h": 234
      }
    }
  ],
  "count": 1
}
```

### Search Analytics
```
GET /vp/analytics/search
```

**Query Parameters**:
- `limit` (int): 1-100 (default: 20)

**Response**:
```json
{
  "top_searches": [
    {
      "search_term": "postgres",
      "count": 1234,
      "avg_results_count": 8.5,
      "click_through_rate": 0.73,
      "conversion_rate": 0.21,
      "installs_from_search": 259,
      "last_searched": "2025-07-05T12:34:56Z"
    }
  ],
  "total_searches": 45678,
  "overall_success_rate": 68.5
}
```

### Time Series Data
```
GET /vp/analytics/time-series
```

**Query Parameters**:
- `start` (string, ISO 8601): Start date
- `end` (string, ISO 8601): End date
- `interval` (string): `hour` | `day` | `week` | `month`

**Response**:
```json
{
  "data": [
    {
      "timestamp": "2025-07-01T00:00:00Z",
      "installs": 234,
      "active_users": 1234,
      "api_calls": 5678,
      "new_servers": 2
    }
  ],
  "start": "2025-07-01T00:00:00Z",
  "end": "2025-07-05T00:00:00Z",
  "interval": "day",
  "count": 5
}
```

## Feedback System

### Submit Rating/Feedback
```
POST /vp/servers/{server_id}/rate
```

**Headers**: `Authorization: Bearer YOUR_GITHUB_TOKEN`

**Request Body**:
```json
{
  "rating": 5,              // Required: 1-5
  "comment": "Great tool!", // Optional: feedback comment
  "version": "1.2.3",       // Optional: version being rated
  "platform": "vscode"      // Optional: platform used
}
```

**Response**:
```json
{
  "success": true,
  "feedback_id": "fb_123456",
  "average_rating": 4.8,
  "total_ratings": 124
}
```

### Get Server Feedback
```
GET /vp/servers/{server_id}/feedback
```

**Query Parameters**:
- `limit` (int): 1-100 (default: 20)
- `offset` (int): For pagination
- `sort` (string): `newest` | `oldest` | `highest` | `lowest`
- `source` (string): `REGISTRY` | `COMMUNITY`

**Response**:
```json
{
  "feedback": [
    {
      "id": "fb_123",
      "user_id": "usr_456",
      "rating": 5,
      "comment": "Excellent extension!",
      "created_at": "2025-07-05T10:30:00Z",
      "helpful_count": 12,
      "version": "1.2.3",
      "platform": "vscode"
    }
  ],
  "total_count": 89,
  "average_rating": 4.7,
  "rating_distribution": {
    "5": 67,
    "4": 15,
    "3": 5,
    "2": 1,
    "1": 1
  },
  "has_more": true
}
```

### Get User's Rating
```
GET /vp/servers/{server_id}/feedback/user
```

**Headers**: `Authorization: Bearer YOUR_GITHUB_TOKEN`

**Response**:
```json
{
  "has_rated": true,
  "rating": 5,
  "feedback_id": "fb_123",
  "created_at": "2025-07-05T10:30:00Z"
}
```

## Activity & Discovery

### Global Statistics
```
GET /vp/stats/global
```

**Query Parameters**:
- `source` (string): `REGISTRY` | `COMMUNITY` | `ALL`

**Response**:
```json
{
  "total_servers": 342,
  "total_installs": 1234567,
  "total_users": 89012,
  "total_ratings": 4521,
  "average_rating": 4.3,
  "categories": {
    "database": 45,
    "ai": 89,
    "development": 120,
    "other": 88
  }
}
```

### Leaderboard
```
GET /vp/stats/leaderboard
```

**Query Parameters**:
- `type` (string): `installs` | `rating` | `growth` | `trending`
- `limit` (int): 1-100 (default: 10)
- `source` (string): `REGISTRY` | `COMMUNITY` | `ALL`

**Response**:
```json
{
  "type": "installs",
  "servers": [
    {
      "rank": 1,
      "server_id": "postgres-tools",
      "server_name": "PostgreSQL Tools",
      "value": 12456,
      "change": 2
    }
  ],
  "updated_at": "2025-07-05T12:00:00Z"
}
```

### Trending Servers
```
GET /vp/stats/trending
```

**Query Parameters**:
- `limit` (int): 1-50 (default: 10)
- `source` (string): `REGISTRY` | `COMMUNITY` | `ALL`

**Response**:
```json
{
  "servers": [
    {
      "server_id": "new-tool",
      "server_name": "New Amazing Tool",
      "install_velocity": 45.2,
      "momentum": 2.3,
      "trend_score": 89.5,
      "stats": {
        "installs_today": 234,
        "installs_yesterday": 102
      }
    }
  ],
  "period": "24h",
  "updated_at": "2025-07-05T12:00:00Z"
}
```

### Recent Servers
```
GET /vp/servers/recent
```

**Query Parameters**:
- `limit` (int): 1-50 (default: 10)
- `days` (int): Look back period (default: 7)
- `source` (string): `REGISTRY` | `COMMUNITY`

**Response**:
```json
{
  "servers": [
    {
      "server": {
        "id": "new-server",
        "name": "New Server",
        "description": "Just added",
        "category": "development",
        "author": {
          "name": "Jane Doe",
          "github": "janedoe"
        }
      },
      "added_at": "2025-07-05T10:00:00Z",
      "stats": {
        "install_count": 12,
        "average_rating": 0,
        "rating_count": 0
      }
    }
  ],
  "count": 1
}
```

## Error Handling

All error responses follow this format:

```json
{
  "success": false,
  "error": "Human-readable error message",
  "details": {
    "field": "Additional context if applicable"
  }
}
```

### Common HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Missing or invalid auth token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Server or resource not found |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error |

## Rate Limiting

Rate limits are applied per IP address or auth token:

| Endpoint Type | Limit | Window |
|--------------|-------|---------|
| Public Read | 100 | 1 minute |
| Authenticated | 500 | 1 minute |
| Write Operations | 50 | 1 minute |

**Rate Limit Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1625500000
Retry-After: 60 (only on 429 responses)
```

## Caching

Responses include cache headers where appropriate:

```
Cache-Control: public, max-age=300
ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4"
```

For conditional requests:
```
If-None-Match: "33a64df551425fcc55e4d42a148795d9f25f89d4"
```

## Notes

1. **Source Parameter**: Always uppercase (`REGISTRY`, `COMMUNITY`, `ALL`)
2. **Timestamps**: All timestamps are in UTC ISO 8601 format
3. **IDs**: Server IDs are case-sensitive strings
4. **Pagination**: Use `limit` and `offset` for pagination
5. **Sorting**: Default sort is typically by relevance or most recent