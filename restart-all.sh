#!/bin/bash

# Restart all registry and analytics services to pick up new environment variables

cd /home/pluggedin/registry

echo "Restarting Traefik proxy..."
docker compose -f docker-compose.proxy.yml restart

echo "Waiting for Traefik to be ready..."
sleep 3

echo "Restarting Registry and MongoDB..."
# Use no-ports compose file to avoid port conflicts with Traefik
docker compose -f docker-compose-noports.yml -f docker-compose.override.yml restart

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
echo "- Analytics API: https://analytics.plugged.in"
echo "- Kibana: https://kibana.plugged.in"
echo "- Traefik Dashboard: https://traefik.plugged.in"