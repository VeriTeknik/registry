# PluggedIn App Stats Integration Guide

This guide explains how to integrate the new stats features from the MCP Registry into the PluggedIn app.

## Overview

The stats extension adds the following capabilities to the registry:
- Installation tracking
- Server ratings with comments (feedback system)
- Analytics integration
- Community server claiming with stats transfer
- Leaderboards and trending servers
- Recent server discovery tracking
- User feedback management

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

## Rating and Feedback System

The enhanced rating system now supports comments and user feedback tracking:

### Submit Rating with Comment

```typescript
interface RatingRequest {
  rating: number;        // 1-5
  comment?: string;      // Optional, max 1000 chars
  user_id: string;       // Track user ratings
  source?: 'REGISTRY' | 'COMMUNITY';
}

interface FeedbackResponse {
  success: boolean;
  message: string;
  feedback: ServerFeedback;
  stats: ServerStats;
}

async function submitRating(serverId: string, rating: number, comment?: string) {
  const response = await fetch(
    `${REGISTRY_VP_URL}/servers/${serverId}/rate`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        rating,
        comment,
        user_id: getCurrentUserId(),
        source: 'REGISTRY'
      })
    }
  );
  
  if (!response.ok) {
    throw new Error(`Rating failed: ${response.statusText}`);
  }
  
  return response.json();
}
```

### Display Server Feedback

```typescript
interface ServerFeedback {
  id: string;
  server_id: string;
  user_id: string;
  rating: number;
  comment?: string;
  created_at: string;
  updated_at: string;
  is_public: boolean;
}

interface FeedbackListResponse {
  feedback: ServerFeedback[];
  total_count: number;
  has_more: boolean;
}

// Get all feedback for a server
async function getServerFeedback(
  serverId: string,
  options?: {
    limit?: number;
    offset?: number;
    sort?: 'newest' | 'oldest' | 'rating_high' | 'rating_low';
    source?: 'REGISTRY' | 'COMMUNITY';
  }
): Promise<FeedbackListResponse> {
  const params = new URLSearchParams({
    limit: String(options?.limit || 20),
    offset: String(options?.offset || 0),
    sort: options?.sort || 'newest',
    source: options?.source || 'REGISTRY'
  });
  
  const response = await fetch(
    `${REGISTRY_VP_URL}/servers/${serverId}/feedback?${params}`
  );
  return response.json();
}
```

### Check User's Rating

```typescript
interface UserFeedbackResponse {
  has_rated: boolean;
  feedback?: ServerFeedback;
}

// Check if user has already rated a server
async function getUserFeedback(
  serverId: string,
  userId: string,
  source: string = 'REGISTRY'
): Promise<UserFeedbackResponse> {
  const response = await fetch(
    `${REGISTRY_VP_URL}/servers/${serverId}/rating/${userId}?source=${source}`
  );
  return response.json();
}
```

### Enhanced Rating Widget

```tsx
function RatingWidget({ server }: { server: ExtendedServer }) {
  const [userRating, setUserRating] = useState<number | null>(null);
  const [comment, setComment] = useState('');
  const [showCommentBox, setShowCommentBox] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [userFeedback, setUserFeedback] = useState<ServerFeedback | null>(null);

  // Check if user already rated
  useEffect(() => {
    getUserFeedback(server.id, getCurrentUserId()).then(response => {
      if (response.has_rated && response.feedback) {
        setUserRating(response.feedback.rating);
        setUserFeedback(response.feedback);
      }
    });
  }, [server.id]);

  async function submitRating(rating: number) {
    setIsSubmitting(true);
    try {
      const result = await submitRating(server.id, rating, comment);
      setUserRating(rating);
      setUserFeedback(result.feedback);
      setShowCommentBox(false);
      setComment('');
      
      // Update local server stats
      if (result.stats) {
        updateServerStats(server.id, result.stats);
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

## Recent Servers Discovery

Track and display recently added servers:

```typescript
interface RecentServerResponse {
  servers: Array<{
    ...ExtendedServer;
    first_seen: string;
    discovered_via: 'stats' | 'import';
  }>;
  total_count: number;
  filter: {
    source: string;
    limit: number;
    days: string;
  };
}

// Get recently discovered servers
async function getRecentServers(options?: {
  limit?: number;
  days?: number;  // Filter by last N days
  source?: 'REGISTRY' | 'COMMUNITY' | 'ALL';
}): Promise<RecentServerResponse> {
  const params = new URLSearchParams();
  if (options?.limit) params.set('limit', String(options.limit));
  if (options?.days) params.set('days', String(options.days));
  if (options?.source) params.set('source', options.source);
  
  const response = await fetch(
    `${REGISTRY_VP_URL}/servers/recent?${params}`
  );
  return response.json();
}

// Display recent servers
function RecentServersWidget() {
  const [servers, setServers] = useState<RecentServerResponse | null>(null);
  const [days, setDays] = useState<number | undefined>();

  useEffect(() => {
    getRecentServers({ limit: 10, days }).then(setServers);
  }, [days]);

  return (
    <div className="recent-servers">
      <h3>Recently Added Servers</h3>
      
      <div className="filter-buttons">
        <button onClick={() => setDays(undefined)} 
                className={!days ? 'active' : ''}>
          All Time
        </button>
        <button onClick={() => setDays(7)}
                className={days === 7 ? 'active' : ''}>
          Last Week
        </button>
        <button onClick={() => setDays(30)}
                className={days === 30 ? 'active' : ''}>
          Last Month
        </button>
      </div>

      {servers?.servers.map(server => (
        <div key={server.id} className="recent-server-item">
          <ServerCard server={server} />
          <div className="discovery-info">
            <TimeAgo date={server.first_seen} />
            {server.discovered_via === 'stats' && (
              <span className="badge">Community Discovered</span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
```

## Feedback Management

### Update/Delete User Feedback

```typescript
// Update existing feedback
async function updateFeedback(
  serverId: string,
  feedbackId: string,
  updates: {
    rating: number;
    comment?: string;
    user_id: string;
  }
): Promise<void> {
  const response = await fetch(
    `${REGISTRY_VP_URL}/servers/${serverId}/feedback/${feedbackId}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updates)
    }
  );
  
  if (!response.ok) {
    throw new Error(`Update failed: ${response.statusText}`);
  }
}

// Delete feedback
async function deleteFeedback(
  serverId: string,
  feedbackId: string,
  userId: string
): Promise<void> {
  const response = await fetch(
    `${REGISTRY_VP_URL}/servers/${serverId}/feedback/${feedbackId}?user_id=${userId}`,
    { method: 'DELETE' }
  );
  
  if (!response.ok) {
    throw new Error(`Delete failed: ${response.statusText}`);
  }
}
```

### Display Feedback List

```tsx
function FeedbackList({ serverId }: { serverId: string }) {
  const [feedback, setFeedback] = useState<FeedbackListResponse | null>(null);
  const [sort, setSort] = useState<'newest' | 'rating_high'>('newest');
  const [page, setPage] = useState(0);
  const limit = 10;

  useEffect(() => {
    getServerFeedback(serverId, {
      limit,
      offset: page * limit,
      sort
    }).then(setFeedback);
  }, [serverId, sort, page]);

  return (
    <div className="feedback-list">
      <div className="feedback-header">
        <h4>User Reviews ({feedback?.total_count || 0})</h4>
        <select value={sort} onChange={(e) => setSort(e.target.value as any)}>
          <option value="newest">Newest First</option>
          <option value="oldest">Oldest First</option>
          <option value="rating_high">Highest Rated</option>
          <option value="rating_low">Lowest Rated</option>
        </select>
      </div>

      {feedback?.feedback.map(item => (
        <div key={item.id} className="feedback-item">
          <div className="feedback-rating">
            <StarRating value={item.rating} readonly />
            <TimeAgo date={item.created_at} />
          </div>
          {item.comment && (
            <p className="feedback-comment">{item.comment}</p>
          )}
          {item.user_id === getCurrentUserId() && (
            <div className="feedback-actions">
              <button onClick={() => handleEdit(item)}>Edit</button>
              <button onClick={() => handleDelete(item)}>Delete</button>
            </div>
          )}
        </div>
      ))}

      {feedback?.has_more && (
        <button onClick={() => setPage(p => p + 1)}>Load More</button>
      )}
    </div>
  );
}
```

## Testing

Test the integration thoroughly:

1. **Stats Display**: Verify stats show correctly for all servers
2. **Installation Tracking**: Confirm installs increment the counter
3. **Ratings & Feedback**: 
   - Test rating submission with comments
   - Verify duplicate rating prevention
   - Check feedback display and pagination
   - Test update/delete operations
4. **Recent Servers**: 
   - Verify new servers appear in recent list
   - Test day filtering
   - Check first_seen timestamps
5. **Leaderboards**: Check all leaderboard types load correctly
6. **Performance**: Ensure caching works and pages load quickly
7. **Error Cases**: Test with network failures, invalid data, etc.
8. **Claiming**: Test the full claim flow with stats transfer

## New Endpoints Summary

### Feedback Endpoints
- `POST /vp/servers/{id}/rate` - Submit rating with optional comment
- `GET /vp/servers/{id}/feedback` - Get all feedback for a server
- `GET /vp/servers/{id}/rating/{user_id}` - Check if user has rated
- `PUT /vp/servers/{id}/feedback/{feedback_id}` - Update feedback
- `DELETE /vp/servers/{id}/feedback/{feedback_id}` - Delete feedback

### Recent Servers Endpoints
- `GET /vp/servers/recent` - Get recently discovered servers
- `GET /vp/admin/timeline` - Server addition timeline (coming soon)

### Existing Stats Endpoints
- `GET /vp/servers` - Servers with stats
- `GET /vp/servers/{id}` - Single server with stats
- `POST /vp/servers/{id}/install` - Track installation
- `GET /vp/servers/{id}/stats` - Detailed server stats
- `GET /vp/stats/global` - Global registry stats
- `GET /vp/stats/leaderboard` - Top servers by various metrics
- `GET /vp/stats/trending` - Trending servers
- `POST /vp/servers/{id}/claim` - Claim community server

## Support

For questions or issues with the stats integration:
- Check the registry logs for errors
- Monitor the browser console for API failures
- Use the `/vp/servers/recent` endpoint to verify new servers are being tracked
- Check feedback endpoints for proper user tracking
- Contact the registry team with specific error messages