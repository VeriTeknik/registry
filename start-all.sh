#!/bin/bash

# Start all registry services in the correct order

cd /home/pluggedin/registry

echo "Starting Traefik proxy..."
docker compose -f docker-compose.proxy.yml up -d

echo "Waiting for Traefik to be ready..."
sleep 5

echo "Starting Registry and MongoDB..."
# Use no-ports compose file to avoid port conflicts with Traefik
# Use extended-override to get the registry with /vp endpoints
docker compose -f docker-compose-noports.yml -f docker-compose.extended-override.yml up -d

echo "All services started!"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"