# Deployment Setup Guide

This guide explains how to set up auto-deployment for the MCP Registry.

## Prerequisites

1. A server with Docker and Docker Compose installed
2. SSH access to the server
3. A domain pointing to your server
4. GitHub repository admin access

## GitHub Secrets Configuration

Go to your repository Settings → Secrets and variables → Actions, and add:

### Required Secrets

1. **DEPLOY_HOST**
   - Your server's hostname or IP address
   - Example: `registry.plugged.in` or `192.168.1.100`

2. **DEPLOY_USER**
   - SSH username for deployment
   - Example: `pluggedin`

3. **DEPLOY_KEY**
   - SSH private key for authentication
   - Generate with: `ssh-keygen -t ed25519 -f deploy_key`
   - Copy the content of `deploy_key` (not `deploy_key.pub`)

4. **DEPLOY_PATH**
   - Absolute path to the application directory on server
   - Example: `/home/pluggedin/registry`

## Server Setup

1. **Add SSH Public Key**
   ```bash
   # On your server, add the public key to authorized_keys
   echo "your-public-key-content" >> ~/.ssh/authorized_keys
   ```

2. **Create Application Directory**
   ```bash
   mkdir -p /home/pluggedin/registry
   cd /home/pluggedin/registry
   ```

3. **Initial Setup**
   ```bash
   # Clone your fork
   git clone https://github.com/VeriTeknik/registry.git .
   
   # Create necessary files
   touch acme.json
   chmod 600 acme.json
   
   # Start services
   docker compose -f docker-compose.proxy.yml up -d
   docker compose up -d
   ```

## Deployment Workflow

### Automatic Deployment

Deployments happen automatically when:
- Code is pushed to the `main` branch
- A PR is merged to `main`
- The sync-upstream workflow merges changes

### Manual Deployment

Trigger manual deployment:
1. Go to Actions → Deploy to Production
2. Click "Run workflow"
3. Enter a deployment reason
4. Click "Run workflow"

### Deployment Process

1. **Tests Run**: Unit and integration tests must pass
2. **Docker Build**: New image is built
3. **Deployment**: Image deployed to server
4. **Health Check**: Service health is verified
5. **Rollback**: Automatic rollback if health check fails

## Monitoring Deployments

### Check Deployment Status

1. Go to the Actions tab in GitHub
2. Click on the running/completed workflow
3. View logs for each step

### Server Logs

```bash
# Check Traefik logs
docker logs traefik

# Check registry logs
docker logs registry

# Check all services
docker compose logs -f
```

### Health Endpoint

Monitor service health:
```bash
curl https://registry.plugged.in/v0/health
```

## Troubleshooting

### SSH Connection Failed

1. Verify DEPLOY_HOST is correct
2. Check DEPLOY_USER has SSH access
3. Ensure DEPLOY_KEY is the private key content
4. Test manually: `ssh -i deploy_key user@host`

### Health Check Failed

1. Check if services are running: `docker ps`
2. View logs: `docker logs registry`
3. Verify domain DNS is configured
4. Check Traefik routing: `curl http://localhost:8080/api/http/routers`

### Rollback Failed

If automatic rollback fails:
```bash
# SSH to server
cd /home/pluggedin/registry

# Manual rollback
docker compose down
docker tag registry:backup-{timestamp} registry:latest
docker compose up -d
```

## Security Considerations

1. **SSH Key**: Use a dedicated deploy key with limited permissions
2. **Secrets**: Never commit secrets to the repository
3. **Firewall**: Only expose ports 80 and 443
4. **Updates**: Keep Docker and system packages updated

## Upstream Sync

The repository automatically syncs with upstream daily:
- Sync workflow runs at 2 AM UTC
- Creates PR if changes detected
- Manual sync: Actions → Sync Fork with Upstream → Run workflow

When conflicts occur:
1. The PR will be created as a draft
2. Manually resolve conflicts
3. Run tests
4. Merge when ready