#!/bin/bash

# Start all registry services in the correct order

cd /home/pluggedin/registry

echo "Starting Traefik proxy..."
docker compose -f docker-compose.proxy.yml up -d

echo "Waiting for Traefik to be ready..."
sleep 5

echo "Starting Registry and MongoDB..."
# Stop any existing registry containers first to avoid conflicts
docker stop registry-extended 2>/dev/null || true
docker stop registry 2>/dev/null || true
docker rm registry-extended 2>/dev/null || true
docker rm registry 2>/dev/null || true

# Use no-ports compose file to avoid port conflicts with Traefik
# Use extended-override to get the registry with /vp endpoints
docker compose -f docker-compose-noports.yml -f docker-compose.extended-override.yml up -d

echo "Waiting for services to be ready..."
sleep 5

echo "All services started!"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "Services available at:"
echo "- Registry: https://registry.plugged.in"
echo "- Registry VP Stats: https://registry.plugged.in/vp/servers"
echo "- Analytics API: https://analytics.plugged.in"

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