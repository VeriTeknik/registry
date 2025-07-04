#!/bin/bash
set -e

echo "Building extended registry with stats support..."

# Build the extended Docker image
docker build -t registry-extended:latest -f extensions/Dockerfile .

echo "Build completed successfully!"
echo ""
echo "To run the extended registry with stats support:"
echo "  docker compose -f docker-compose.yml -f docker-compose.extended-override.yml up -d"
echo ""
echo "Stats endpoints will be available at:"
echo "  - /vp/servers - Enhanced server listing with stats"
echo "  - /vp/servers/{id}/stats - Individual server statistics"  
echo "  - /vp/stats/global - Global registry statistics"
echo "  - /vp/stats/leaderboard - Top servers by various metrics"