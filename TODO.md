# MCP Registry - TODO List

Generated: 2025-07-02

## 📊 Current Status

### ✅ Completed
- Analytics infrastructure deployed (Elasticsearch, Kibana, Redis)
- MongoDB sync service running and syncing data
- Analytics API deployed and accessible
- Basic authentication implemented for Kibana and Analytics API
- Data Views created in Kibana (servers*, events*, metrics*, feedback*)
- Traefik configured for routing (SSL pending due to rate limits)

### ⚠️ In Progress
- SSL certificates for kibana.plugged.in and analytics.plugged.in (rate limited)
- Analytics integration into main registry application
- Kibana dashboard creation

### 📁 Uncommitted Changes
```
M analytics/docker-compose.yml          # Added authentication
M docker-compose.proxy.yml              # Removed traefik.plugged.in
M traefik.yml                          # Removed global HTTPS redirect
?? internal/analytics/                  # New analytics client
?? internal/middleware/                 # New middleware directory
```

## 🚨 High Priority Tasks

### 1. Commit Current Changes
```bash
# Analytics infrastructure
git add analytics/
git commit -m "feat: Add analytics infrastructure with Elasticsearch, Kibana, and Redis

- Deploy Elasticsearch for analytics data storage
- Add Kibana for data visualization
- Include Redis for caching
- Add MongoDB sync service for data synchronization
- Implement basic auth for security"

# Analytics client
git add internal/analytics/
git commit -m "feat: Add analytics client for event tracking

- Create client for tracking server views, searches, and publishes
- Implement async event sending
- Add structured event types"

# Traefik updates
git add docker-compose.proxy.yml traefik.yml analytics/docker-compose.yml
git commit -m "fix: Update Traefik configuration for analytics services

- Remove traefik.plugged.in subdomain requirement
- Fix HTTP challenge routing for Let's Encrypt
- Add authentication middleware for analytics services"
```

### 2. Complete Analytics Integration

#### Add to main.go or config:
```go
// Add to internal/config/config.go
AnalyticsURL string `envconfig:"ANALYTICS_URL" default:""`

// Initialize in main application
analyticsClient := analytics.NewClient(cfg.AnalyticsURL)
```

#### Track in handlers:

**File: internal/api/handlers/v0/servers.go**
- Line 28-97: Add tracking for server list views
- Line 99-134: Add tracking for individual server views

**File: internal/api/handlers/v0/publish.go**
- Line 18-133: Track publish events (new vs update)

**File: extensions/vp/handlers/servers.go**
- Line 28-126: Track enhanced API usage with filters

### 3. Test Analytics Pipeline
```bash
# Test event tracking
curl -X POST https://analytics.plugged.in/api/v1/track \
  -H "Content-Type: application/json" \
  -u admin:o6FdPN55UJLuP0 \
  -d '{
    "event_type": "view",
    "server_id": "test-server",
    "client_id": "test-client"
  }'

# Check if events appear in Elasticsearch
curl -u admin:o6FdPN55UJLuP0 \
  https://analytics.plugged.in/api/v1/servers/test-server/stats
```

## 📋 Medium Priority Tasks

### 4. Create Kibana Dashboards

Access: https://kibana.plugged.in (admin/o6FdPN55UJLuP0)

#### Dashboard 1: Server Overview
- Total servers count (Metric visualization)
- Servers by category (Pie chart)
- New servers over time (Line chart)
- Top 10 most viewed servers (Data table)

#### Dashboard 2: API Analytics
- Requests by endpoint (Bar chart)
- Error rate over time (Line chart)
- Response time percentiles (Line chart)
- Active users count (Metric)

#### Dashboard 3: Search Analytics
- Popular search terms (Tag cloud)
- Searches with no results (Data table)
- Search volume over time (Area chart)

### 5. Update Documentation

#### Update ANALYTICS_DEPLOYMENT.md:
- Change status from "Ready to Deploy" to "Running"
- Add authentication credentials
- Update access URLs
- Add dashboard creation guide

#### Update FORK-MODIFICATIONS.md:
- Add analytics infrastructure section
- Document new internal/analytics directory
- List all analytics-related files

#### Create ANALYTICS_INTEGRATION.md:
- How to add new event types
- Analytics client usage examples
- Dashboard creation guide
- Troubleshooting common issues

### 6. Production Hardening

```bash
# Set up Elasticsearch index lifecycle
PUT _ilm/policy/analytics-policy
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": {
            "max_size": "50GB",
            "max_age": "30d"
          }
        }
      },
      "delete": {
        "min_age": "90d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

## 🔮 Future Enhancements

### 7. Advanced Analytics Features
- [ ] User session tracking
- [ ] Funnel analysis for user journeys
- [ ] A/B testing framework
- [ ] Real-time analytics dashboard
- [ ] Geographic analytics with IP geolocation

### 8. Security Enhancements
- [ ] API key authentication for programmatic access
- [ ] Rate limiting per user/IP
- [ ] Audit logging for all actions
- [ ] Encrypted data at rest in Elasticsearch

### 9. Performance Optimizations
- [ ] Implement caching layer with Redis
- [ ] Batch event processing
- [ ] Optimize Elasticsearch queries
- [ ] Add CDN for static assets

### 10. Monitoring & Alerts
- [ ] Set up Prometheus metrics
- [ ] Configure Grafana dashboards
- [ ] Create PagerDuty alerts for service health
- [ ] Implement SLO/SLA tracking

## 🔧 SSL Certificate Resolution

Wait until rate limit expires or try alternative approaches:

```bash
# Check rate limit expiry
docker logs traefik | grep rateLimited

# Alternative: Use Cloudflare for SSL
# Alternative: Try DNS challenge instead of HTTP
# Alternative: Use Let's Encrypt staging for testing
```

## 📝 Environment Variables Needed

Add to production environment:
```bash
MCP_REGISTRY_ANALYTICS_URL=http://analytics-api:8081
MCP_REGISTRY_ANALYTICS_ENABLED=true
```

## 🎯 Quick Reference

### Service URLs
- Registry: https://registry.plugged.in
- Kibana: https://kibana.plugged.in (admin/o6FdPN55UJLuP0)
- Analytics API: https://analytics.plugged.in (admin/o6FdPN55UJLuP0)

### Health Checks
```bash
# Check all services
docker ps | grep -E "registry|analytics|kibana|elasticsearch|redis|mongo"

# Analytics API health
curl -u admin:o6FdPN55UJLuP0 https://analytics.plugged.in/api/v1/health

# Elasticsearch health
curl http://localhost:9200/_cluster/health?pretty
```

### Useful Commands
```bash
# View analytics logs
docker compose -f analytics/docker-compose.yml logs -f

# Restart analytics services
docker compose -f analytics/docker-compose.yml restart

# Check sync status
docker logs analytics-sync --tail 50
```

---
Last Updated: 2025-07-02 03:45 UTC
Next Review: After completing high priority tasks