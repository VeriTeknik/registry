# PluggedIn App Stats Integration Guide

This guide explains how to integrate the new stats features from the MCP Registry into the PluggedIn app.

## Overview

The stats extension adds the following capabilities to the registry:
- Installation tracking
- Server ratings
- Analytics integration
- Community server claiming with stats transfer
- Leaderboards and trending servers

All new endpoints are under the `/vp` (v-plugged) prefix to maintain compatibility with the upstream registry.

## API Migration

### 1. Update Base URLs

Replace existing v0 endpoints with vp endpoints to get stats-enhanced responses:

```typescript
// OLD
const REGISTRY_BASE_URL = 'https://registry.plugged.in/v0';

// NEW - Add VP endpoint support
const REGISTRY_BASE_URL = 'https://registry.plugged.in';
const REGISTRY_V0_URL = `${REGISTRY_BASE_URL}/v0`;
const REGISTRY_VP_URL = `${REGISTRY_BASE_URL}/vp`;
```

### 2. Update Server Listing

The `/vp/servers` endpoint returns servers with stats included:

```typescript
interface ExtendedServer extends Server {
  installation_count: number;
  rating: number;
  rating_count: number;
  active_installs?: number;
  weekly_growth?: number;
}

interface ExtendedServersResponse {
  servers: ExtendedServer[];
}

// Fetch servers with stats
async function getServers(): Promise<ExtendedServer[]> {
  const response = await fetch(`${REGISTRY_VP_URL}/servers`);
  const data: ExtendedServersResponse = await response.json();
  return data.servers;
}
```

### 3. Display Stats in UI

Update your server cards/lists to show the new stats:

```tsx
function ServerCard({ server }: { server: ExtendedServer }) {
  return (
    <div className="server-card">
      <h3>{server.name}</h3>
      <p>{server.description}</p>
      
      {/* New stats display */}
      <div className="server-stats">
        <span className="installs">
          <InstallIcon /> {server.installation_count.toLocaleString()}
        </span>
        
        {server.rating_count > 0 && (
          <span className="rating">
            <StarIcon /> {server.rating.toFixed(1)} ({server.rating_count})
          </span>
        )}
        
        {server.weekly_growth > 0 && (
          <span className="trending">
            <TrendingIcon /> +{server.weekly_growth.toFixed(0)}%
          </span>
        )}
      </div>
    </div>
  );
}
```

## Installation Tracking

Track when users install a server:

```typescript
async function trackInstallation(serverId: string, metadata?: {
  userId?: string;
  version?: string;
  platform?: string;
}) {
  try {
    await fetch(`${REGISTRY_VP_URL}/servers/${serverId}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: metadata?.userId,
        version: metadata?.version,
        platform: metadata?.platform || detectPlatform(),
        timestamp: Date.now()
      })
    });
  } catch (error) {
    console.error('Failed to track installation:', error);
    // Don't block installation on tracking failure
  }
}

// Use it when installing
async function installServer(server: ExtendedServer) {
  // Your existing install logic
  await performInstallation(server);
  
  // Track the installation
  await trackInstallation(server.id, {
    userId: getCurrentUserId(),
    version: server.version,
    platform: process.platform
  });
}
```

## Rating System

Implement a rating widget:

```tsx
function RatingWidget({ server }: { server: ExtendedServer }) {
  const [userRating, setUserRating] = useState<number | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function submitRating(rating: number) {
    setIsSubmitting(true);
    try {
      const response = await fetch(
        `${REGISTRY_VP_URL}/servers/${server.id}/rate`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ rating })
        }
      );
      
      if (response.ok) {
        const result = await response.json();
        setUserRating(rating);
        // Optionally update local server stats
        if (result.stats) {
          updateServerStats(server.id, result.stats);
        }
      }
    } catch (error) {
      console.error('Failed to submit rating:', error);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="rating-widget">
      <div className="current-rating">
        {server.rating_count > 0 ? (
          <>
            <StarIcon filled /> {server.rating.toFixed(1)}
            <span className="count">({server.rating_count} ratings)</span>
          </>
        ) : (
          <span>No ratings yet</span>
        )}
      </div>
      
      <div className="user-rating">
        <span>Rate this server:</span>
        {[1, 2, 3, 4, 5].map(star => (
          <button
            key={star}
            onClick={() => submitRating(star)}
            disabled={isSubmitting}
            className={userRating && userRating >= star ? 'filled' : ''}
          >
            <StarIcon />
          </button>
        ))}
      </div>
    </div>
  );
}
```

## Leaderboards and Trending

Add leaderboard views to your app:

```typescript
type LeaderboardType = 'installs' | 'rating' | 'trending';

async function getLeaderboard(
  type: LeaderboardType,
  limit: number = 10
): Promise<ServerStats[]> {
  const response = await fetch(
    `${REGISTRY_VP_URL}/stats/leaderboard?type=${type}&limit=${limit}`
  );
  const data = await response.json();
  return data.data;
}

async function getTrendingServers(limit: number = 20): Promise<ExtendedServer[]> {
  const response = await fetch(
    `${REGISTRY_VP_URL}/stats/trending?limit=${limit}`
  );
  const data = await response.json();
  return data.servers;
}
```

```tsx
function LeaderboardView() {
  const [type, setType] = useState<LeaderboardType>('installs');
  const [servers, setServers] = useState<ExtendedServer[]>([]);

  useEffect(() => {
    getLeaderboard(type, 20).then(setServers);
  }, [type]);

  return (
    <div className="leaderboard">
      <div className="leaderboard-tabs">
        <button onClick={() => setType('installs')} 
                className={type === 'installs' ? 'active' : ''}>
          Most Installed
        </button>
        <button onClick={() => setType('rating')}
                className={type === 'rating' ? 'active' : ''}>
          Top Rated
        </button>
        <button onClick={() => setType('trending')}
                className={type === 'trending' ? 'active' : ''}>
          Trending
        </button>
      </div>
      
      <div className="leaderboard-list">
        {servers.map((server, index) => (
          <div key={server.id} className="leaderboard-item">
            <span className="rank">#{index + 1}</span>
            <ServerCard server={server} />
          </div>
        ))}
      </div>
    </div>
  );
}
```

## Global Stats Dashboard

Show registry-wide statistics:

```typescript
interface GlobalStats {
  total_servers: number;
  total_installs: number;
  active_servers: number;
  average_rating: number;
  last_updated: string;
}

async function getGlobalStats(): Promise<GlobalStats> {
  const response = await fetch(`${REGISTRY_VP_URL}/stats/global`);
  return response.json();
}

function StatsWidget() {
  const [stats, setStats] = useState<GlobalStats | null>(null);

  useEffect(() => {
    getGlobalStats().then(setStats);
  }, []);

  if (!stats) return <div>Loading stats...</div>;

  return (
    <div className="global-stats">
      <div className="stat">
        <h4>Total Servers</h4>
        <p>{stats.total_servers.toLocaleString()}</p>
      </div>
      <div className="stat">
        <h4>Total Installs</h4>
        <p>{stats.total_installs.toLocaleString()}</p>
      </div>
      <div className="stat">
        <h4>Active Servers</h4>
        <p>{stats.active_servers.toLocaleString()}</p>
      </div>
      <div className="stat">
        <h4>Average Rating</h4>
        <p>⭐ {stats.average_rating.toFixed(1)}</p>
      </div>
    </div>
  );
}
```

## Community Server Claiming

Allow users to claim community servers:

```typescript
interface ClaimRequest {
  publish_request: PublishRequest;
  transfer_stats: boolean;
}

async function claimCommunityServer(
  serverId: string,
  publishRequest: PublishRequest,
  transferStats: boolean = true
) {
  const response = await fetch(
    `${REGISTRY_VP_URL}/servers/${serverId}/claim`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getGitHubToken()}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        publish_request: publishRequest,
        transfer_stats: transferStats
      })
    }
  );

  if (!response.ok) {
    throw new Error(`Claim failed: ${response.statusText}`);
  }

  return response.json();
}
```

## Caching Considerations

The VP endpoints include caching headers. Respect them in your app:

```typescript
// Check X-Cache header to show cache status
const response = await fetch(`${REGISTRY_VP_URL}/servers`);
const cacheStatus = response.headers.get('X-Cache'); // 'HIT' or 'MISS'

// Implement client-side caching
const cache = new Map<string, { data: any; expires: number }>();

async function fetchWithCache(url: string, ttl: number = 300000) { // 5 min default
  const cached = cache.get(url);
  if (cached && cached.expires > Date.now()) {
    return cached.data;
  }

  const response = await fetch(url);
  const data = await response.json();
  
  cache.set(url, {
    data,
    expires: Date.now() + ttl
  });
  
  return data;
}
```

## Migration Checklist

- [ ] Update API base URLs to use `/vp` endpoints
- [ ] Update TypeScript interfaces to include stats fields
- [ ] Add installation tracking to your install flow
- [ ] Implement rating widget in server details
- [ ] Add stats display to server cards/lists
- [ ] Create leaderboard/trending views
- [ ] Add global stats dashboard
- [ ] Implement community server claiming
- [ ] Test all endpoints with the new stats data
- [ ] Add appropriate error handling for stats features

## Error Handling

Stats features should fail gracefully:

```typescript
// Wrap stats calls to prevent breaking core functionality
async function getServerWithStats(serverId: string): Promise<ExtendedServer> {
  try {
    // Try VP endpoint first
    const response = await fetch(`${REGISTRY_VP_URL}/servers/${serverId}`);
    if (response.ok) {
      const data = await response.json();
      return data.server;
    }
  } catch (error) {
    console.warn('Stats endpoint failed, falling back to v0', error);
  }

  // Fallback to v0 endpoint
  const response = await fetch(`${REGISTRY_V0_URL}/servers/${serverId}`);
  const server = await response.json();
  
  // Add default stats values
  return {
    ...server,
    installation_count: 0,
    rating: 0,
    rating_count: 0
  };
}
```

## Testing

Test the integration thoroughly:

1. **Stats Display**: Verify stats show correctly for all servers
2. **Installation Tracking**: Confirm installs increment the counter
3. **Ratings**: Test rating submission and display updates
4. **Leaderboards**: Check all leaderboard types load correctly
5. **Performance**: Ensure caching works and pages load quickly
6. **Error Cases**: Test with network failures, invalid data, etc.
7. **Claiming**: Test the full claim flow with stats transfer

## Support

For questions or issues with the stats integration:
- Check the registry logs for errors
- Monitor the browser console for API failures
- Contact the registry team with specific error messages