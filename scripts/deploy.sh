#!/bin/bash
set -e

# Deployment script for MCP Registry with Traefik
# This script performs zero-downtime deployment with health checks and rollback capability

COMPOSE_PROJECT_NAME="registry"
HEALTH_CHECK_URL="${REGISTRY_HEALTH_URL:-https://registry.plugged.in/v0/health}"
DEPLOY_TIMEOUT="${DEPLOY_TIMEOUT:-300}"
BACKUP_TAG="backup-$(date +%Y%m%d-%H%M%S)"

# Validate required environment variables
if [ -z "$HEALTH_CHECK_URL" ]; then
    echo "Error: REGISTRY_HEALTH_URL environment variable is required"
    exit 1
fi

echo "🚀 Starting deployment..."

# Function to check service health
check_health() {
    local service=$1
    local max_attempts=30
    local attempt=1
    
    echo "Checking health of $service..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$HEALTH_CHECK_URL" > /dev/null; then
            echo "✅ Health check passed"
            return 0
        fi
        
        echo "Attempt $attempt/$max_attempts failed, retrying..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "❌ Health check failed after $max_attempts attempts"
    return 1
}

# Function to backup current state
backup_current() {
    echo "📦 Creating backup of current deployment..."
    
    # Tag current images as backup
    docker tag registry-extended:latest registry-extended:$BACKUP_TAG || true
    docker tag registry:latest registry:$BACKUP_TAG || true
    
    # Save current compose files
    cp docker-compose.yml docker-compose.yml.backup || true
    cp docker-compose.override.yml docker-compose.override.yml.backup || true
    cp docker-compose.extended-override.yml docker-compose.extended-override.yml.backup || true
}

# Function to rollback
rollback() {
    echo "⚠️  Rolling back to previous version..."
    
    # Stop current containers
    docker compose -f docker-compose.yml -f docker-compose.extended-override.yml down
    
    # Restore backup images
    docker tag registry-extended:$BACKUP_TAG registry-extended:latest || true
    docker tag registry:$BACKUP_TAG registry:latest || true
    
    # Restore compose files
    mv docker-compose.yml.backup docker-compose.yml || true
    mv docker-compose.override.yml.backup docker-compose.override.yml || true
    mv docker-compose.extended-override.yml.backup docker-compose.extended-override.yml || true
    
    # Start services with extended configuration
    docker compose -f docker-compose.yml -f docker-compose.extended-override.yml up -d
    
    echo "✅ Rollback completed"
}

# Main deployment process
main() {
    # Step 1: Backup current state
    backup_current
    
    # Step 2: Pull latest changes
    echo "📥 Pulling latest configuration..."
    
    # Step 3: Build extended image
    echo "🔨 Building extended registry image..."
    docker build -t registry-extended:latest -f Dockerfile .
    
    # Step 4: Update Traefik if needed
    echo "🔄 Updating Traefik..."
    docker compose -f docker-compose.proxy.yml up -d
    sleep 5
    
    # Step 5: Deploy registry with rolling update
    echo "🔄 Deploying extended registry service with stats..."
    
    # Use extended configuration for deployment
    docker compose -f docker-compose.yml -f docker-compose.extended-override.yml up -d --no-deps --scale registry=2 registry
    
    # Wait for new container to be healthy
    if ! check_health "registry"; then
        echo "❌ New deployment failed health check"
        rollback
        exit 1
    fi
    
    # Step 6: Remove old container
    echo "🧹 Cleaning up old containers..."
    docker compose -f docker-compose.yml -f docker-compose.extended-override.yml up -d --no-deps --scale registry=1 registry
    
    # Step 7: Clean up backup images (keep last 3)
    echo "🧹 Cleaning up old backup images..."
    # Cleanup extended registry backups
    docker images --format "table {{.Repository}}:{{.Tag}}" | grep "^registry-extended:backup-" | tail -n +4 | while read -r image; do
        docker rmi "$image" || true
    done
    
    # Cleanup registry backups
    docker images --format "table {{.Repository}}:{{.Tag}}" | grep "^registry:backup-" | tail -n +4 | while read -r image; do
        docker rmi "$image" || true
    done
    
    # Step 8: Prune unused resources
    docker system prune -f --volumes
    
    echo "✅ Deployment completed successfully!"
    echo "📊 Stats endpoints available at: https://registry.plugged.in/vp/*"
}

# Trap errors and rollback if needed
trap 'if [ $? -ne 0 ]; then rollback; fi' EXIT

# Run main deployment
main

# Remove trap on success
trap - EXIT