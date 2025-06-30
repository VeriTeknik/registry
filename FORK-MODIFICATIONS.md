# Fork Modifications

This document tracks all custom modifications made to the VeriTeknik/registry fork of modelcontextprotocol/registry.

## Overview

This fork is deployed at https://registry.plugged.in with the following infrastructure modifications:

- **Traefik Reverse Proxy**: Handles SSL termination and routing
- **Enhanced Security**: No direct port exposure, all traffic through Traefik
- **Auto-deployment**: GitHub Actions workflows for CI/CD
- **Environment Configuration**: Support for GitHub OAuth and custom settings

## File Modifications

### Modified Files

1. **`.gitignore`**
   - Added `acme.json` to ignore Let's Encrypt certificates

### Added Files

1. **`docker-compose.override.yml`**
   - Configures Traefik labels for the registry service
   - Removes direct port mappings for security
   - Adds Traefik network configuration

2. **`docker-compose.proxy.yml`**
   - Defines Traefik reverse proxy service
   - Configures SSL with Let's Encrypt
   - Sets up automatic HTTPS redirect

3. **`traefik.yml`**
   - Traefik static configuration
   - Defines entry points, providers, and certificate resolvers
   - Security headers and rate limiting middleware

4. **`docker-compose-noports.yml`**
   - Copy of docker-compose.yml without port mappings
   - Used in production to avoid port conflicts with Traefik
   - Allows upstream docker-compose.yml to remain unchanged

5. **`.github/workflows/sync-upstream.yml`**
   - Automated daily sync with upstream repository
   - Creates PR when upstream changes are detected
   - Handles merge conflicts gracefully

6. **`.github/workflows/deploy.yml`**
   - Auto-deployment pipeline triggered on main branch updates
   - Includes testing, building, deployment, and rollback
   - Health checks to ensure successful deployment

7. **`scripts/deploy.sh`**
   - Zero-downtime deployment script
   - Health checking and automatic rollback
   - Docker image backup management

8. **`CLAUDE.md`**
   - Documentation for AI assistants working on the codebase
   - Common commands and architecture overview

9. **Startup Scripts**
   - `registry-services.service`: Systemd service for auto-start
   - `install-startup.sh`: Installation script for systemd service
   - `start-all.sh`: Manual startup script

## Environment Variables

The following environment variables are used in production:

```bash
MCP_REGISTRY_DATABASE_URL      # MongoDB connection string
MCP_REGISTRY_ENVIRONMENT       # Deployment environment
MCP_REGISTRY_GITHUB_CLIENT_ID  # GitHub OAuth client ID
MCP_REGISTRY_GITHUB_CLIENT_SECRET # GitHub OAuth client secret
```

## Deployment Architecture

```
Internet → Traefik (SSL) → Registry Service → MongoDB
              ↓
         Let's Encrypt
```

## Maintaining Fork Sync

1. **Upstream Remote**: Configured as `upstream` pointing to modelcontextprotocol/registry
2. **Automated Sync**: Daily GitHub Action checks for upstream changes
3. **Conflict Resolution**: Manual intervention required for conflicts
4. **Clean Separation**: All customizations in override files to minimize conflicts

## GitHub Secrets Required

For auto-deployment to work, configure these secrets in GitHub:

- `DEPLOY_HOST`: Your server hostname
- `DEPLOY_USER`: SSH username for deployment
- `DEPLOY_KEY`: SSH private key for authentication
- `DEPLOY_PATH`: Path to the application on server

## Rollback Procedure

If deployment fails:
1. Automatic rollback triggered by deploy.yml workflow
2. Manual rollback: SSH to server and run `docker tag registry:backup-{timestamp} registry:latest`
3. Restart services: `docker compose down && docker compose up -d`

## Local Development

To run locally with Traefik:
```bash
# Start Traefik
docker compose -f docker-compose.proxy.yml up -d

# Start Registry
docker compose up -d
```

Without Traefik (development):
```bash
# Temporarily restore ports in docker-compose.yml
# Then run:
docker compose up -d
```