#!/bin/bash

# Test script for analytics endpoints
# This script tests the new analytics and dashboard endpoints

API_BASE="http://localhost:8080"

echo "=== Testing Analytics Endpoints ==="
echo

# Test Dashboard Metrics
echo "1. Testing Dashboard Metrics (GET /vp/analytics/dashboard)"
curl -s "$API_BASE/vp/analytics/dashboard?period=day" | jq '.' || echo "Dashboard endpoint failed"
echo
echo "---"

# Test Analytics with all includes
echo "2. Testing Full Analytics (GET /vp/analytics)"
curl -s "$API_BASE/vp/analytics?period=week&include_activity=true&include_trending=true&include_categories=true&include_search=true" | jq '.' || echo "Analytics endpoint failed"
echo
echo "---"

# Test Activity Feed
echo "3. Testing Activity Feed (GET /vp/analytics/activity)"
curl -s "$API_BASE/vp/analytics/activity?limit=10" | jq '.' || echo "Activity feed failed"
echo
echo "---"

# Test Growth Metrics
echo "4. Testing Growth Metrics (GET /vp/analytics/growth)"
echo "   - Installs growth:"
curl -s "$API_BASE/vp/analytics/growth?metric=installs&period=week" | jq '.' || echo "Growth metrics failed"
echo
echo "   - Users growth:"
curl -s "$API_BASE/vp/analytics/growth?metric=users&period=week" | jq '.' || echo "Growth metrics failed"
echo
echo "   - API calls growth:"
curl -s "$API_BASE/vp/analytics/growth?metric=api_calls&period=week" | jq '.' || echo "Growth metrics failed"
echo
echo "---"

# Test API Metrics
echo "5. Testing API Metrics (GET /vp/analytics/api-metrics)"
curl -s "$API_BASE/vp/analytics/api-metrics?limit=10" | jq '.' || echo "API metrics failed"
echo
echo "---"

# Test Search Analytics
echo "6. Testing Search Analytics (GET /vp/analytics/search)"
curl -s "$API_BASE/vp/analytics/search?limit=10" | jq '.' || echo "Search analytics failed"
echo
echo "---"

# Test Time Series
echo "7. Testing Time Series (GET /vp/analytics/time-series)"
curl -s "$API_BASE/vp/analytics/time-series?interval=hour" | jq '.' || echo "Time series failed"
echo
echo "---"

# Test Hot Servers
echo "8. Testing Hot Servers (GET /vp/analytics/hot)"
curl -s "$API_BASE/vp/analytics/hot?limit=5" | jq '.' || echo "Hot servers failed"
echo
echo "---"

# Create some test activity to populate analytics
echo "9. Creating test activity..."

# Track some API calls
echo "   - Simulating API calls..."
for i in {1..5}; do
    curl -s "$API_BASE/vp/servers" > /dev/null
    curl -s "$API_BASE/vp/stats/global" > /dev/null
done

# Track an installation
SERVER_ID="mcp-server-postgres"
echo "   - Tracking installation for $SERVER_ID..."
curl -s -X POST "$API_BASE/vp/servers/$SERVER_ID/install" \
  -H "Content-Type: application/json" \
  -d '{"source": "REGISTRY"}' | jq '.'

echo
echo "---"

# Re-test dashboard to see updated metrics
echo "10. Re-testing Dashboard Metrics after activity"
curl -s "$API_BASE/vp/analytics/dashboard?period=day" | jq '.' || echo "Dashboard endpoint failed"

echo
echo "=== Analytics Testing Complete ==="