#!/bin/bash

# Restart all registry and analytics services to pick up new environment variables

cd /home/pluggedin/registry

echo "Restarting Traefik proxy..."
docker compose -f docker-compose.proxy.yml restart

echo "Waiting for Traefik to be ready..."
sleep 3

echo "Rebuilding extended registry image..."
docker build -t registry-extended:latest -f extensions/Dockerfile .

echo "Restarting Registry and MongoDB..."
# Stop and remove any existing registry containers first
docker stop registry-extended 2>/dev/null || true
docker stop registry 2>/dev/null || true
docker rm registry-extended 2>/dev/null || true
docker rm registry 2>/dev/null || true

# Use no-ports compose file to avoid port conflicts with Traefik
# Use extended-override to get the registry with /vp endpoints
docker compose -f docker-compose-noports.yml -f docker-compose.extended-override.yml down
docker compose -f docker-compose-noports.yml -f docker-compose.extended-override.yml up -d

echo "Waiting for Registry to be ready..."
sleep 5

echo "Restarting Analytics services..."
docker compose -f analytics/docker-compose.yml restart

echo "All services restarted!"
echo ""
echo "Service Status:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "Services should be available at:"
echo "- Registry: https://registry.plugged.in"
echo "- Registry VP Stats: https://registry.plugged.in/vp/servers"
echo "- Analytics API: https://analytics.plugged.in"
echo "- Kibana: https://kibana.plugged.in"
echo "- Traefik Dashboard: https://traefik.plugged.in"

echo ""
echo "Checking service health..."
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