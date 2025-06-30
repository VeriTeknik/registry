# MCP Registry Setup for plugged.in

Quick setup guide for deploying the MCP Registry at `registry.plugged.in` with SSL.

## 🚀 Quick Start

```bash
# Run the pre-configured setup
./setup-plugged-in.sh
```

This will set up:
- ✅ MCP Registry at `https://registry.plugged.in`
- ✅ Traefik dashboard at `https://traefik.plugged.in`
- ✅ Automatic SSL certificates
- ✅ Security headers and rate limiting

## 📋 Prerequisites

### 1. DNS Configuration
Ensure these DNS records point to your server:

```
A    registry.plugged.in  →  YOUR_SERVER_IP
A    traefik.plugged.in   →  YOUR_SERVER_IP
```

### 2. Firewall
```bash
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP (for SSL challenge)
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

### 3. Server Requirements
- Docker & Docker Compose installed
- Ports 80, 443 available
- At least 1GB RAM

## 🎯 Services

| Service | URL | Purpose |
|---------|-----|---------|
| **MCP Registry** | `https://registry.plugged.in` | Main API service |
| **API Health** | `https://registry.plugged.in/v0/health` | Health check |
| **API Docs** | `https://registry.plugged.in/v0/swagger/index.html` | Swagger documentation |
| **Traefik Dashboard** | `https://traefik.plugged.in` | Proxy dashboard |
| **Local Dashboard** | `http://localhost:8080` | Local Traefik access |

## 🔧 Management Commands

### Start Services
```bash
# Start everything
docker-compose up -d

# Start with proxy
docker-compose -f docker-compose.proxy.yml up -d
docker-compose up -d
```

### Monitor Logs
```bash
# Registry logs
docker-compose logs -f registry

# Traefik logs  
docker-compose -f docker-compose.proxy.yml logs -f traefik

# All services
docker-compose logs -f
```

### Health Checks
```bash
# Check registry health
curl -I https://registry.plugged.in/v0/health

# Check SSL certificate
echo | openssl s_client -connect registry.plugged.in:443 2>/dev/null | openssl x509 -noout -dates

# Check services status
docker-compose ps
```

### Restart Services
```bash
# Restart registry only
docker-compose restart registry

# Restart proxy only
docker-compose -f docker-compose.proxy.yml restart traefik

# Restart everything
docker-compose down && docker-compose up -d
```

## 🔒 Security Features

- **SSL/TLS**: Automatic Let's Encrypt certificates
- **HTTPS Redirect**: All HTTP traffic redirected to HTTPS
- **Security Headers**: HSTS, X-Frame-Options, etc.
- **Rate Limiting**: 100 req/sec average, 200 burst
- **Network Isolation**: MongoDB not exposed externally
- **Internal Access Only**: Traefik dashboard via localhost

## 🧪 Testing

### API Endpoints
```bash
# Health check
curl https://registry.plugged.in/v0/health

# List servers
curl https://registry.plugged.in/v0/servers

# Get specific server
curl https://registry.plugged.in/v0/servers/{server-id}
```

### Performance Test
```bash
# Simple load test
for i in {1..10}; do
  curl -s -o /dev/null -w "%{http_code} %{time_total}s\n" \
    https://registry.plugged.in/v0/health
done
```

## 🔧 Troubleshooting

### Common Issues

**1. SSL Certificate Not Working**
```bash
# Check Traefik logs
docker-compose -f docker-compose.proxy.yml logs traefik | grep -i error

# Verify DNS
dig registry.plugged.in
nslookup registry.plugged.in

# Test HTTP challenge
curl -I http://registry.plugged.in
```

**2. Service Not Reachable**
```bash
# Check container status
docker-compose ps

# Test internal connectivity
docker-compose exec registry curl http://localhost:8080/v0/health

# Check networks
docker network ls
docker network inspect traefik
```

**3. MongoDB Connection Issues**
```bash
# Check MongoDB logs
docker-compose logs mongodb

# Test MongoDB connectivity
docker-compose exec registry nc -zv mongodb 27017
```

### Reset Everything
```bash
# Complete reset (WARNING: Destroys data)
docker-compose down -v
docker-compose -f docker-compose.proxy.yml down -v
docker network rm traefik
sudo rm -rf .db/ acme.json traefik-config/

# Then re-run setup
./setup-plugged-in.sh
```

## 📊 Production Tips

### 1. Backup Important Files
```bash
# Create backup
tar -czf mcp-registry-backup-$(date +%Y%m%d).tar.gz \
  acme.json \
  traefik.yml \
  docker-compose*.yml \
  .db/
```

### 2. Monitor Disk Space
```bash
# Check disk usage
df -h
docker system df

# Clean up old containers/images
docker system prune -a
```

### 3. Update Services
```bash
# Update registry
docker-compose pull
docker-compose up -d

# Update Traefik
docker-compose -f docker-compose.proxy.yml pull
docker-compose -f docker-compose.proxy.yml up -d
```

## 📞 Quick Commands Reference

```bash
# Setup
./setup-plugged-in.sh

# Start
docker-compose up -d

# Monitor
docker-compose logs -f

# Test
curl -I https://registry.plugged.in/v0/health

# Stop
docker-compose down

# Restart
docker-compose restart
```

---

**Server:** `registry.plugged.in`  
**Dashboard:** `traefik.plugged.in`  
**Local Dashboard:** `localhost:8080` 