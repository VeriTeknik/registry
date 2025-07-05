# VP Stats & Analytics Frontend Integration Guide

## Table of Contents
1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Core Stats API](#core-stats-api)
4. [Analytics Dashboard API](#analytics-dashboard-api)
5. [Feedback System](#feedback-system)
6. [Activity & Real-time Features](#activity--real-time-features)
7. [Search Analytics](#search-analytics)
8. [Performance & Caching](#performance--caching)
9. [Error Handling](#error-handling)
10. [TypeScript/JavaScript Examples](#typescriptjavascript-examples)
11. [React Integration Examples](#react-integration-examples)
12. [Testing](#testing)

## Overview

The VP (v-plugged) Stats & Analytics system provides comprehensive metrics and analytics for MCP servers. It tracks installations, usage, ratings, and provides real-time analytics dashboards.

### Base URL
```
https://registry.plugged.in/vp
```

### Key Features
- Real-time installation and usage tracking
- User feedback and ratings system
- Analytics dashboards with trends
- Activity feeds and hot servers
- Search analytics and conversion tracking
- Growth metrics and momentum analysis

## Authentication

Most read endpoints are public. Write operations require authentication:

```javascript
// For authenticated requests (feedback, claims)
const headers = {
  'Authorization': 'Bearer YOUR_GITHUB_TOKEN',
  'Content-Type': 'application/json'
};
```

## Core Stats API

### 1. Get Server Stats
Retrieve statistics for a specific server.

**Endpoint:** `GET /vp/servers/{server_id}/stats`

**Query Parameters:**
- `source` (optional): `REGISTRY` | `COMMUNITY` | `ALL` (default: `REGISTRY`)
- `aggregated` (optional): `true` to get combined stats from all sources

**Example Request:**
```javascript
// Get stats for a specific server
const response = await fetch('https://registry.plugged.in/vp/servers/my-server-id/stats?source=REGISTRY');
const stats = await response.json();
```

**Response:**
```json
{
  "server_id": "my-server-id",
  "source": "REGISTRY",
  "stats": {
    "install_count": 1523,
    "average_rating": 4.7,
    "rating_count": 89,
    "daily_active_users": 234,
    "weekly_growth_rate": 12.5,
    "last_updated": "2025-07-05T12:34:56Z"
  },
  "trending": {
    "rank": 5,
    "momentum": 1.8,
    "category": "hot"
  }
}
```

### 2. Track Installation
Record an installation event.

**Endpoint:** `POST /vp/servers/{server_id}/install`

**Request Body:**
```json
{
  "platform": "vscode",
  "version": "1.2.3",
  "source": "marketplace",
  "metadata": {
    "user_id": "anonymous-uuid",
    "country": "US"
  }
}
```

**Example:**
```javascript
await fetch('https://registry.plugged.in/vp/servers/my-server-id/install', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    platform: 'vscode',
    version: '1.2.3'
  })
});
```

### 3. List Servers with Stats
Get all servers with their stats.

**Endpoint:** `GET /vp/servers`

**Query Parameters:**
- `limit` (optional): 1-100 (default: 20)
- `offset` (optional): For pagination
- `sort` (optional): `installs` | `rating` | `trending` | `newest`
- `category` (optional): Filter by category

**Response:**
```json
{
  "servers": [
    {
      "id": "server-1",
      "name": "PostgreSQL Tools",
      "description": "Comprehensive PostgreSQL management tools",
      "stats": {
        "install_count": 5234,
        "average_rating": 4.8,
        "rating_count": 234
      },
      "author": {
        "name": "John Doe",
        "github": "johndoe"
      }
    }
  ],
  "total": 156,
  "has_more": true
}
```

### 4. Global Statistics
Get overall registry statistics.

**Endpoint:** `GET /vp/stats/global`

**Response:**
```json
{
  "total_servers": 342,
  "total_installs": 1234567,
  "total_users": 89012,
  "average_rating": 4.3,
  "categories": {
    "database": 45,
    "ai": 89,
    "development": 120
  }
}
```

## Analytics Dashboard API

### 1. Dashboard Metrics
Get comprehensive dashboard metrics with trends.

**Endpoint:** `GET /vp/analytics/dashboard`

**Query Parameters:**
- `period`: `day` | `week` | `month` | `year` (default: `day`)

**Response:**
```json
{
  "total_installs": {
    "value": 15234,
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
    "trend_direction": "up",
    "comparison_period": "vs yesterday"
  },
  "new_servers_today": 12,
  "install_velocity": 234.5,
  "top_rated_count": 89,
  "search_success_rate": 76.3,
  "install_trend": [120, 145, 132, 156, 189, 201, 234],
  "hottest_server": {
    "server_id": "trending-server",
    "server_name": "AI Assistant",
    "value": "45.2/hr",
    "label": "installs/hour"
  }
}
```

### 2. Growth Metrics
Track growth for specific metrics.

**Endpoint:** `GET /vp/analytics/growth`

**Query Parameters:**
- `metric`: `installs` | `users` | `api_calls` | `servers` | `ratings` | `searches`
- `period`: `day` | `week` | `month` | `year`

**Response:**
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

### 3. Time Series Data
Get historical data for charts.

**Endpoint:** `GET /vp/analytics/time-series`

**Query Parameters:**
- `start`: ISO 8601 date
- `end`: ISO 8601 date
- `interval`: `hour` | `day` | `week` | `month`

**Example:**
```javascript
const params = new URLSearchParams({
  start: '2025-06-01T00:00:00Z',
  end: '2025-07-01T00:00:00Z',
  interval: 'day'
});

const response = await fetch(`https://registry.plugged.in/vp/analytics/time-series?${params}`);
```

### 4. Hot/Trending Servers
Get servers with sudden activity spikes.

**Endpoint:** `GET /vp/analytics/hot`

**Query Parameters:**
- `limit`: 1-50 (default: 10)

**Response:**
```json
{
  "servers": [
    {
      "server_id": "hot-server-1",
      "server_name": "New AI Tool",
      "install_velocity": 145.2,
      "momentum_change": 280.5,
      "trend_category": "viral",
      "stats": {
        "installs_24h": 3489,
        "installs_prev_24h": 234
      }
    }
  ],
  "count": 3
}
```

## Feedback System

### 1. Submit Feedback/Rating
**Endpoint:** `POST /vp/servers/{server_id}/rate`

**Headers:** Requires authentication

**Request Body:**
```json
{
  "rating": 5,
  "comment": "Excellent tool! Works perfectly with my workflow.",
  "version": "1.2.3",
  "platform": "vscode"
}
```

### 2. Get Server Feedback
**Endpoint:** `GET /vp/servers/{server_id}/feedback`

**Query Parameters:**
- `limit`: 1-100 (default: 20)
- `offset`: For pagination
- `sort`: `newest` | `oldest` | `highest` | `lowest`
- `source`: `REGISTRY` | `COMMUNITY`

**Response:**
```json
{
  "feedback": [
    {
      "id": "feedback-123",
      "user_id": "user-456",
      "rating": 5,
      "comment": "Great extension!",
      "created_at": "2025-07-05T10:30:00Z",
      "helpful_count": 12,
      "version": "1.2.3"
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
  }
}
```

### 3. Check User's Rating
**Endpoint:** `GET /vp/servers/{server_id}/rating/{user_id}`

**Response:**
```json
{
  "has_rated": true,
  "rating": 5,
  "feedback_id": "feedback-123",
  "created_at": "2025-07-05T10:30:00Z"
}
```

## Activity & Real-time Features

### 1. Activity Feed
Get real-time activity stream.

**Endpoint:** `GET /vp/analytics/activity`

**Query Parameters:**
- `limit`: 1-100 (default: 20)
- `type`: Filter by event type (`install` | `rating` | `search` | `server_added`)

**Response:**
```json
{
  "activity": [
    {
      "id": "event-123",
      "type": "install",
      "timestamp": "2025-07-05T12:34:56Z",
      "server_id": "postgres-tools",
      "server_name": "PostgreSQL Tools",
      "metadata": {
        "platform": "vscode",
        "version": "1.2.3"
      }
    },
    {
      "id": "event-124",
      "type": "rating",
      "timestamp": "2025-07-05T12:35:00Z",
      "server_id": "ai-assistant",
      "server_name": "AI Assistant",
      "metadata": {
        "rating": 5,
        "user_id": "anonymous-uuid"
      }
    }
  ],
  "count": 2
}
```

### 2. Leaderboard
Get top-performing servers.

**Endpoint:** `GET /vp/stats/leaderboard`

**Query Parameters:**
- `type`: `installs` | `rating` | `growth` | `trending`
- `limit`: 1-100 (default: 10)
- `source`: `REGISTRY` | `COMMUNITY` | `ALL`

### 3. Trending Servers
**Endpoint:** `GET /vp/stats/trending`

**Query Parameters:**
- `limit`: 1-50 (default: 10)
- `source`: `REGISTRY` | `COMMUNITY` | `ALL`

## Search Analytics

### 1. Top Searches
Get most popular search terms.

**Endpoint:** `GET /vp/analytics/search`

**Response:**
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

## Performance & Caching

### Caching Strategy
Most endpoints support caching. Check response headers:

```javascript
// Check cache headers
const response = await fetch(url);
const cacheControl = response.headers.get('Cache-Control');
const etag = response.headers.get('ETag');

// Use conditional requests
const cachedResponse = await fetch(url, {
  headers: {
    'If-None-Match': etag
  }
});
```

### Rate Limiting
- Public endpoints: 100 requests per minute
- Authenticated endpoints: 500 requests per minute
- Bulk operations: 10 requests per minute

Check headers for rate limit info:
```javascript
const remaining = response.headers.get('X-RateLimit-Remaining');
const reset = response.headers.get('X-RateLimit-Reset');
```

## Error Handling

### Error Response Format
```json
{
  "success": false,
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

### Common Error Codes
- `400`: Invalid parameters
- `401`: Authentication required
- `403`: Forbidden
- `404`: Server not found
- `429`: Rate limit exceeded
- `500`: Internal server error

### Retry Strategy
```javascript
async function fetchWithRetry(url, options = {}, retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      const response = await fetch(url, options);
      
      if (response.status === 429) {
        // Rate limited - wait and retry
        const retryAfter = response.headers.get('Retry-After') || 60;
        await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
        continue;
      }
      
      if (!response.ok && i < retries - 1) {
        // Exponential backoff for other errors
        await new Promise(resolve => setTimeout(resolve, Math.pow(2, i) * 1000));
        continue;
      }
      
      return response;
    } catch (error) {
      if (i === retries - 1) throw error;
      await new Promise(resolve => setTimeout(resolve, Math.pow(2, i) * 1000));
    }
  }
}
```

## TypeScript/JavaScript Examples

### API Client Class
```typescript
interface ServerStats {
  install_count: number;
  average_rating: number;
  rating_count: number;
  daily_active_users?: number;
  weekly_growth_rate?: number;
}

interface DashboardMetrics {
  total_installs: MetricWithTrend;
  total_api_calls: MetricWithTrend;
  active_users: MetricWithTrend;
  server_health: MetricWithTrend;
}

interface MetricWithTrend {
  value: number | string;
  trend: number;
  trend_direction: 'up' | 'down' | 'stable';
  comparison_period: string;
}

class VPStatsClient {
  private baseUrl = 'https://registry.plugged.in/vp';
  private cache = new Map<string, { data: any; timestamp: number }>();
  private cacheTTL = 5 * 60 * 1000; // 5 minutes

  async getServerStats(serverId: string, source = 'REGISTRY'): Promise<ServerStats> {
    const cacheKey = `stats:${serverId}:${source}`;
    const cached = this.getFromCache(cacheKey);
    if (cached) return cached;

    const response = await fetch(`${this.baseUrl}/servers/${serverId}/stats?source=${source}`);
    if (!response.ok) throw new Error(`Failed to fetch stats: ${response.statusText}`);
    
    const data = await response.json();
    this.setCache(cacheKey, data.stats);
    return data.stats;
  }

  async getDashboardMetrics(period = 'day'): Promise<DashboardMetrics> {
    const response = await fetch(`${this.baseUrl}/analytics/dashboard?period=${period}`);
    if (!response.ok) throw new Error(`Failed to fetch dashboard: ${response.statusText}`);
    return response.json();
  }

  async trackInstall(serverId: string, platform: string, version?: string): Promise<void> {
    await fetch(`${this.baseUrl}/servers/${serverId}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ platform, version })
    });
  }

  async submitRating(
    serverId: string, 
    rating: number, 
    comment: string, 
    authToken: string
  ): Promise<void> {
    const response = await fetch(`${this.baseUrl}/servers/${serverId}/rate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`
      },
      body: JSON.stringify({ rating, comment })
    });
    
    if (!response.ok) {
      throw new Error(`Failed to submit rating: ${response.statusText}`);
    }
  }

  private getFromCache(key: string): any | null {
    const cached = this.cache.get(key);
    if (!cached) return null;
    
    if (Date.now() - cached.timestamp > this.cacheTTL) {
      this.cache.delete(key);
      return null;
    }
    
    return cached.data;
  }

  private setCache(key: string, data: any): void {
    this.cache.set(key, { data, timestamp: Date.now() });
  }
}
```

## React Integration Examples

### Stats Dashboard Component
```jsx
import React, { useState, useEffect } from 'react';
import { VPStatsClient } from './vpStatsClient';

const client = new VPStatsClient();

function ServerStatsDashboard({ serverId }) {
  const [stats, setStats] = useState(null);
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const [statsData, metricsData] = await Promise.all([
          client.getServerStats(serverId),
          client.getDashboardMetrics()
        ]);
        setStats(statsData);
        setMetrics(metricsData);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }

    fetchData();
    const interval = setInterval(fetchData, 60000); // Refresh every minute
    return () => clearInterval(interval);
  }, [serverId]);

  if (loading) return <div>Loading stats...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div className="stats-dashboard">
      <div className="server-stats">
        <h3>Server Statistics</h3>
        <div className="stat-grid">
          <StatCard
            label="Installations"
            value={stats.install_count}
            trend={stats.weekly_growth_rate}
          />
          <StatCard
            label="Rating"
            value={`${stats.average_rating}/5`}
            subtitle={`${stats.rating_count} reviews`}
          />
          <StatCard
            label="Active Users"
            value={stats.daily_active_users || 'N/A'}
          />
        </div>
      </div>

      <div className="global-metrics">
        <h3>Registry Overview</h3>
        <MetricCard metric={metrics.total_installs} label="Total Installs" />
        <MetricCard metric={metrics.active_users} label="Active Users" />
        <MetricCard metric={metrics.server_health} label="Server Health" />
      </div>
    </div>
  );
}

function StatCard({ label, value, trend, subtitle }) {
  return (
    <div className="stat-card">
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      {trend && (
        <div className={`stat-trend ${trend > 0 ? 'positive' : 'negative'}`}>
          {trend > 0 ? '↑' : '↓'} {Math.abs(trend)}%
        </div>
      )}
      {subtitle && <div className="stat-subtitle">{subtitle}</div>}
    </div>
  );
}

function MetricCard({ metric, label }) {
  return (
    <div className="metric-card">
      <div className="metric-label">{label}</div>
      <div className="metric-value">{metric.value}</div>
      <div className={`metric-trend ${metric.trend_direction}`}>
        {metric.trend_direction === 'up' ? '↑' : metric.trend_direction === 'down' ? '↓' : '→'}
        {Math.abs(metric.trend)}% {metric.comparison_period}
      </div>
    </div>
  );
}
```

### Activity Feed Hook
```javascript
import { useState, useEffect, useCallback } from 'react';

export function useActivityFeed(limit = 20) {
  const [activities, setActivities] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchActivities = useCallback(async () => {
    try {
      const response = await fetch(
        `https://registry.plugged.in/vp/analytics/activity?limit=${limit}`
      );
      const data = await response.json();
      setActivities(data.activity || []);
    } catch (error) {
      console.error('Failed to fetch activities:', error);
    } finally {
      setLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    fetchActivities();
    const interval = setInterval(fetchActivities, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [fetchActivities]);

  return { activities, loading, refresh: fetchActivities };
}

// Usage
function ActivityFeed() {
  const { activities, loading } = useActivityFeed(10);

  if (loading) return <div>Loading activity...</div>;

  return (
    <div className="activity-feed">
      {activities.map((activity) => (
        <ActivityItem key={activity.id} activity={activity} />
      ))}
    </div>
  );
}
```

### Installation Tracking
```javascript
// Track installations with error handling
export async function trackInstallation(serverId, platform = 'web') {
  try {
    await fetch(`https://registry.plugged.in/vp/servers/${serverId}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        platform,
        version: process.env.REACT_APP_VERSION,
        source: 'marketplace'
      })
    });
  } catch (error) {
    console.error('Failed to track installation:', error);
    // Queue for retry or send to analytics fallback
  }
}

// Use in install button
function InstallButton({ serverId }) {
  const [installing, setInstalling] = useState(false);

  const handleInstall = async () => {
    setInstalling(true);
    try {
      // Your installation logic here
      await installServer(serverId);
      
      // Track the installation
      await trackInstallation(serverId, 'web');
      
      // Show success
      toast.success('Successfully installed!');
    } catch (error) {
      toast.error('Installation failed');
    } finally {
      setInstalling(false);
    }
  };

  return (
    <button onClick={handleInstall} disabled={installing}>
      {installing ? 'Installing...' : 'Install'}
    </button>
  );
}
```

## Testing

### Mock VP API for Development
```javascript
// mockVPApi.js
export class MockVPApi {
  constructor() {
    this.data = {
      stats: new Map(),
      feedback: new Map(),
      activities: []
    };
  }

  async getServerStats(serverId) {
    await this.delay(100);
    return {
      install_count: Math.floor(Math.random() * 10000),
      average_rating: (Math.random() * 2 + 3).toFixed(1),
      rating_count: Math.floor(Math.random() * 500),
      daily_active_users: Math.floor(Math.random() * 1000)
    };
  }

  async getDashboardMetrics() {
    await this.delay(150);
    return {
      total_installs: {
        value: 123456,
        trend: (Math.random() * 20 - 10).toFixed(1),
        trend_direction: Math.random() > 0.5 ? 'up' : 'down',
        comparison_period: 'vs yesterday'
      },
      // ... other metrics
    };
  }

  async trackInstall(serverId) {
    await this.delay(50);
    const stats = this.data.stats.get(serverId) || { installs: 0 };
    stats.installs++;
    this.data.stats.set(serverId, stats);
    
    this.data.activities.unshift({
      id: Date.now().toString(),
      type: 'install',
      timestamp: new Date().toISOString(),
      server_id: serverId
    });
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Use in development
const api = process.env.NODE_ENV === 'development' 
  ? new MockVPApi() 
  : new VPStatsClient();
```

### Integration Tests
```javascript
import { render, screen, waitFor } from '@testing-library/react';
import { ServerStatsDashboard } from './ServerStatsDashboard';
import { VPStatsClient } from './vpStatsClient';

jest.mock('./vpStatsClient');

describe('ServerStatsDashboard', () => {
  beforeEach(() => {
    VPStatsClient.mockClear();
  });

  test('displays server statistics', async () => {
    const mockStats = {
      install_count: 1234,
      average_rating: 4.5,
      rating_count: 89
    };

    VPStatsClient.prototype.getServerStats = jest.fn()
      .mockResolvedValue(mockStats);

    render(<ServerStatsDashboard serverId="test-server" />);

    await waitFor(() => {
      expect(screen.getByText('1234')).toBeInTheDocument();
      expect(screen.getByText('4.5/5')).toBeInTheDocument();
      expect(screen.getByText('89 reviews')).toBeInTheDocument();
    });
  });

  test('handles API errors gracefully', async () => {
    VPStatsClient.prototype.getServerStats = jest.fn()
      .mockRejectedValue(new Error('API Error'));

    render(<ServerStatsDashboard serverId="test-server" />);

    await waitFor(() => {
      expect(screen.getByText(/Error: API Error/)).toBeInTheDocument();
    });
  });
});
```

## Best Practices

1. **Always handle errors gracefully** - Users should see meaningful messages
2. **Implement retry logic** for transient failures
3. **Cache responses** to reduce API calls and improve performance
4. **Use conditional requests** with ETags when supported
5. **Track client-side errors** for debugging
6. **Batch API calls** when possible using Promise.all()
7. **Show loading states** for all async operations
8. **Implement optimistic updates** for better UX
9. **Use TypeScript** for better type safety and IDE support
10. **Monitor rate limits** and implement backoff strategies

## Migration from Basic Stats

If migrating from the basic stats endpoints to enhanced analytics:

1. **Update endpoints**:
   - `/v0/servers` → `/vp/servers` (includes stats)
   - Basic counts → `/vp/analytics/dashboard` (comprehensive metrics)

2. **Enhanced data**:
   - Simple install count → Trending, velocity, growth metrics
   - Basic rating → Rating distribution, feedback comments
   - View count → API calls, active users, engagement metrics

3. **New capabilities**:
   - Real-time activity feeds
   - Growth analytics
   - Search conversion tracking
   - Hot/trending servers

## Support

For issues or questions:
- GitHub Issues: https://github.com/modelcontextprotocol/registry/issues
- API Status: Check response headers for service health
- Rate Limit Issues: Implement exponential backoff
- Data Discrepancies: Stats are eventually consistent (5-minute cache)