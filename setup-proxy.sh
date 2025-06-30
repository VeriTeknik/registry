#!/bin/bash

# MCP Registry SSL Proxy Setup Script
# This script sets up Traefik reverse proxy with SSL for the MCP Registry

set -e

echo "🚀 Setting up SSL Proxy for MCP Registry"

# Check if running as root for Docker setup
if [[ $EUID -eq 0 ]]; then
   echo "⚠️  Don't run this script as root. Docker should be accessible to your user."
   exit 1
fi

# Configuration
DOMAIN_NAME="registry.plugged.in"
echo "🏷️  Using domain: $DOMAIN_NAME"

read -p "Enter your email for Let's Encrypt: " EMAIL

if [[ -z "$EMAIL" ]]; then
    echo "❌ Email is required for Let's Encrypt"
    exit 1
fi

echo "📝 Configuring Traefik..."

# Update traefik.yml with user's email
sed -i "s/your-email@example.com/$EMAIL/g" traefik.yml

# Update docker-compose.override.yml with user's domain
sed -i "s/registry.yourdomain.com/$DOMAIN_NAME/g" docker-compose.override.yml

# Create external network
echo "🌐 Creating Traefik network..."
docker network create traefik 2>/dev/null || echo "Network 'traefik' already exists"

# Create acme.json for Let's Encrypt
echo "🔐 Setting up SSL certificate storage..."
touch acme.json
chmod 600 acme.json

# Create traefik config directory
mkdir -p traefik-config

echo "🏗️  Starting services..."

# Stop current registry if running
docker-compose down 2>/dev/null || true

# Start Traefik first
docker-compose -f docker-compose.proxy.yml up -d

# Start registry with override
docker-compose up -d

echo "✅ Setup complete!"
echo ""
echo "🌍 Your MCP Registry will be available at:"
echo "   https://$DOMAIN_NAME"
echo ""
echo "📊 Traefik dashboard available at:"
echo "   http://localhost:8080 (local only)"
echo ""
echo "⚠️  Make sure your domain DNS points to this server's IP address"
echo "⏳ SSL certificates may take a few minutes to generate"
echo ""
echo "🔍 To check status:"
echo "   docker-compose logs -f"
echo "   docker-compose -f docker-compose.proxy.yml logs -f traefik" 