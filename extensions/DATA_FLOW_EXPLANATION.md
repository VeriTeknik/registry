# Data Flow Explanation: How Stats & Analytics Data Gets Populated

## Overview
The VP Stats & Analytics system collects data through multiple channels and stores it in various MongoDB collections. Here's how data flows through the system and why collections might appear empty initially.

## Data Population Methods

### 1. User-Initiated Actions (Primary Source)
These actions directly populate the MongoDB collections:

#### Installation Tracking
```
User clicks Install → POST /vp/servers/{id}/install → MongoDB
```
- **Updates**: `stats` collection (install_count++)
- **Creates**: `activity_events` entry (type: "install")
- **Updates**: `analytics_metrics` (total_installs++)

#### Rating/Feedback Submission
```
User submits rating → POST /vp/servers/{id}/rate → MongoDB
```
- **Creates**: `feedback` entry with rating & comment
- **Updates**: `stats` collection (rating aggregates)
- **Creates**: `activity_events` entry (type: "rating")

#### Search Actions
```
User searches → Internal tracking → MongoDB
```
- **Updates**: `search_analytics` collection
- **Creates**: `activity_events` entry (type: "search")
- **Tracks**: Conversion if search leads to install

### 2. API Usage Tracking (Automatic)
The system automatically tracks API usage:

```
Any API call → Middleware → MongoDB
```
- **Updates**: `api_calls` collection
- **Aggregates**: Response times, error rates
- **Updates**: `analytics_metrics` (total_api_calls++)

### 3. External Analytics Sync (Optional)
If configured with external analytics service:

```
External Analytics API → Sync Service → MongoDB
```
- **Runs**: Every 15 minutes
- **Updates**: `stats` collection with:
  - active_installs
  - daily_active_users
  - weekly_growth_rate
- **Preserves**: Local install counts

## Why Collections Might Be Empty

### Initial State
When first deployed, all collections are empty because:
1. No user actions have occurred yet
2. No API calls have been tracked
3. External sync hasn't run (if configured)

### Data Accumulation Timeline

**Immediate** (within seconds):
- API calls start populating `api_calls`
- Any user action creates `activity_events`

**Short-term** (within minutes):
- First installations populate `stats`
- Search queries populate `search_analytics`
- Dashboard metrics start showing data

**Medium-term** (within hours):
- Feedback/ratings accumulate
- Time series data builds up
- Trending calculations become meaningful

**Long-term** (within days):
- Growth metrics become accurate
- Milestones are achieved
- Historical trends emerge

## Testing Data Population

### Manual Testing
1. **Track an installation**:
   ```bash
   curl -X POST https://registry.plugged.in/vp/servers/test-server/install \
     -H "Content-Type: application/json" \
     -d '{"platform": "test", "version": "1.0.0"}'
   ```

2. **Submit feedback** (requires auth):
   ```bash
   curl -X POST https://registry.plugged.in/vp/servers/test-server/rate \
     -H "Authorization: Bearer YOUR_GITHUB_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"rating": 5, "comment": "Test feedback"}'
   ```

3. **Check results**:
   ```bash
   curl https://registry.plugged.in/vp/servers/test-server/stats
   ```

### Automated Population
For development/testing, you can:
1. Create a seeder script to populate test data
2. Use the activity simulator
3. Import historical data

## Data Relationships

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   User      │────▶│ VP Endpoints │────▶│  MongoDB    │
│  Actions    │     │              │     │ Collections │
└─────────────┘     └──────────────┘     └─────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │ Cache Layer  │
                    │ (5 min TTL)  │
                    └──────────────┘

┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  External   │────▶│ Sync Service │────▶│   Stats     │
│ Analytics   │     │ (15 min)     │     │ Collection  │
└─────────────┘     └──────────────┘     └─────────────┘
```

## Verification Steps

1. **Check if data is being written**:
   - Monitor MongoDB logs
   - Check application logs for errors
   - Verify network connectivity

2. **Validate data flow**:
   - Test each endpoint manually
   - Check MongoDB directly
   - Review cache behavior

3. **Debug empty collections**:
   - Ensure MongoDB connection is active
   - Check for validation errors
   - Verify correct database/collection names
   - Review user permissions

## Common Issues & Solutions

### Issue: Stats show 0 despite user actions
**Cause**: Cache not invalidated
**Solution**: Clear cache or wait for TTL expiry

### Issue: External analytics not syncing
**Cause**: Analytics API not configured
**Solution**: Set `MCP_REGISTRY_ANALYTICS_URL` environment variable

### Issue: Feedback not appearing
**Cause**: Authentication required
**Solution**: Ensure valid GitHub token is provided

### Issue: Activity feed empty
**Cause**: No recent activity
**Solution**: Trigger some test actions to populate

## Best Practices

1. **Development Environment**:
   - Use seed data for consistent testing
   - Implement data generators
   - Lower cache TTL for faster feedback

2. **Production Environment**:
   - Monitor data flow continuously
   - Set up alerts for anomalies
   - Regular backups of analytics data

3. **Data Integrity**:
   - Validate all inputs
   - Use transactions where appropriate
   - Implement idempotency for critical operations