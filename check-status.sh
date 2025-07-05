#!/bin/bash

# Check status of all registry services

echo "Service Status:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Image}}" | grep -E "NAME|registry|mongodb|traefik|analytics"

echo ""
echo "Health Checks:"
echo -n "- Registry Health: "
if curl -sf https://registry.plugged.in/v0/health > /dev/null 2>&1; then
    echo "✅ OK"
else
    echo "❌ Failed"
fi

echo -n "- VP Endpoints: "
if curl -sf https://registry.plugged.in/vp/servers > /dev/null 2>&1; then
    echo "✅ OK"
else
    echo "❌ Failed"
fi

echo -n "- Feedback Endpoint: "
if curl -sf https://registry.plugged.in/vp/servers/test-server/feedback > /dev/null 2>&1; then
    echo "✅ OK"
else
    echo "❌ Failed"
fi

echo -n "- Analytics API: "
status_code=$(curl -s -o /dev/null -w "%{http_code}" https://analytics.plugged.in/ 2>/dev/null)
if [ "$status_code" = "401" ]; then
    echo "✅ OK (requires auth)"
elif [ "$status_code" = "200" ]; then
    echo "✅ OK"
else
    echo "❌ Failed (HTTP $status_code)"
fi

echo ""
echo "Endpoints:"
echo "- Registry: https://registry.plugged.in"
echo "- VP Stats: https://registry.plugged.in/vp/servers"
echo "- Feedback: https://registry.plugged.in/vp/servers/{server-id}/feedback"
echo "- Recent Servers: https://registry.plugged.in/vp/servers/recent"
echo "- Analytics: https://analytics.plugged.in"
echo "- Kibana: https://kibana.plugged.in"

echo ""
echo "Recent Server Additions:"
curl -s https://registry.plugged.in/vp/servers/recent?limit=5 | jq -r '.servers[] | "- \(.name) (\(.discovered_via)) - \(.first_seen)"' 2>/dev/null || echo "Unable to fetch recent servers"