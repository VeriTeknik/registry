#!/bin/bash
set -e

echo "Starting MCP Registry with Stats Extension..."

# Check if .env file exists, if not create from example
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        echo "Creating .env file from .env.example..."
        cp .env.example .env
        echo "Please update .env with your configuration values"
        exit 1
    fi
fi

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Build the extended image
echo "Building extended registry image..."
docker build -t registry-extended:latest -f Dockerfile .

# Start services with extended override
echo "Starting services..."
docker compose -f docker-compose.yml -f docker-compose.extended-override.yml up -d

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 5

# Check service status
echo ""
echo "Service Status:"
docker compose -f docker-compose.yml -f docker-compose.extended-override.yml ps

echo ""
echo "Registry with stats extension is now running!"
echo ""
echo "Available endpoints:"
echo "  Main registry: https://registry.plugged.in"
echo "  Stats API: https://registry.plugged.in/vp/*"
echo ""
echo "Example stats endpoints:"
echo "  - GET /vp/servers?sort=installs"
echo "  - GET /vp/servers/{id}/stats"
echo "  - POST /vp/servers/{id}/install"
echo "  - POST /vp/servers/{id}/rate"
echo "  - GET /vp/stats/global"
echo "  - GET /vp/stats/leaderboard?type=rating"
echo ""
echo "To view logs: docker compose -f docker-compose.yml -f docker-compose.extended-override.yml logs -f"