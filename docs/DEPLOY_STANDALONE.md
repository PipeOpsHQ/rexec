# Standalone Deployment (Remote Docker)

This guide explains how to deploy Rexec with a remote Docker daemon. This is the recommended approach for production deployments.

## Architecture

```
[ Rexec Container ]  ----(TLS/TCP)---->  [ Docker Host VM ]
      │                                        │
      └─> [PostgreSQL / Redis]                 └─> [User Containers]
```

## When to use this?

- **Production**: Recommended for all production deployments
- **PaaS Platforms**: Required for Railway, PipeOps, Render, etc. (no privileged mode)
- **Scalability**: Separate Docker host can be scaled independently
- **Security**: User containers are isolated from the Rexec API

## Prerequisites

- A Linux VM for the Docker host (Hetzner, DigitalOcean, Linode, Fly.io, etc.)
- A PostgreSQL database
- Redis (optional, recommended)

## Step 1: Set Up the Docker Host

### Automated Setup (Recommended)

Run this on a fresh Ubuntu/Debian VM:

```bash
curl -fsSL https://raw.githubusercontent.com/your-repo/rexec/main/docker/docker-host/setup.sh | sudo bash
```

This script will:

1. Install Docker
2. Generate TLS certificates
3. Configure Docker daemon for secure remote access
4. Open firewall port 2376
5. Output all required environment variables

### Manual Setup

See [DEPLOY_PAAS.md](./DEPLOY_PAAS.md) for detailed manual setup instructions.

## Step 2: Build and Run Rexec

### Using Docker

```bash
# Build the image
docker build -t rexec .

# Run with remote Docker
docker run -d \
  -p 8080:8080 \
  -e DOCKER_HOST="tcp://your-docker-host:2376" \
  -e DOCKER_TLS_VERIFY="1" \
  -e DOCKER_CA_CERT="$(cat ca.pem)" \
  -e DOCKER_CLIENT_CERT="$(cat cert.pem)" \
  -e DOCKER_CLIENT_KEY="$(cat key.pem)" \
  -e DATABASE_URL="postgres://user:pass@host:5432/rexec" \
  -e JWT_SECRET="your-secret" \
  rexec
```

### Using Docker Compose

```bash
cd docker
docker compose up -d
```

Make sure to set the environment variables in `docker-compose.yml` or a `.env` file.

## Step 3: Configure Environment Variables

### Required Variables

| Variable             | Description                             |
| -------------------- | --------------------------------------- |
| `DOCKER_HOST`        | `tcp://YOUR_DOCKER_HOST_IP:2376`        |
| `DOCKER_TLS_VERIFY`  | `1` (enable TLS verification)           |
| `DOCKER_CA_CERT`     | Contents of `ca.pem` from Docker host   |
| `DOCKER_CLIENT_CERT` | Contents of `cert.pem` from Docker host |
| `DOCKER_CLIENT_KEY`  | Contents of `key.pem` from Docker host  |
| `DATABASE_URL`       | PostgreSQL connection string            |
| `JWT_SECRET`         | Secret for signing auth tokens          |

### Optional Variables

| Variable            | Default | Description                           |
| ------------------- | ------- | ------------------------------------- |
| `REDIS_URL`         | -       | Redis connection string (recommended) |
| `PORT`              | `8080`  | API server port                       |
| `GIN_MODE`          | -       | Set to `release` for production       |
| `STRIPE_SECRET_KEY` | -       | For billing features                  |

## Deploying to PaaS Platforms

### Railway

1. Create a new project from your GitHub repo
2. Set **Dockerfile Path** to `Dockerfile`
3. Add environment variables (see above)
4. Deploy

### PipeOps / Render

1. Connect your repository
2. Select Dockerfile deployment
3. Configure environment variables
4. Deploy

### Fly.io

```bash
# Create fly.toml
fly launch --no-deploy

# Set secrets
fly secrets set DOCKER_HOST="tcp://your-docker-host:2376"
fly secrets set DOCKER_TLS_VERIFY="1"
fly secrets set DOCKER_CA_CERT="$(cat ca.pem)"
fly secrets set DOCKER_CLIENT_CERT="$(cat cert.pem)"
fly secrets set DOCKER_CLIENT_KEY="$(cat key.pem)"
fly secrets set DATABASE_URL="postgres://..."
fly secrets set JWT_SECRET="..."

# Deploy
fly deploy
```

## Health Check

Verify the deployment is working:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "ok",
  "database": "connected",
  "docker": "connected"
}
```

## Troubleshooting

### Connection Refused to Docker Host

- Check firewall allows port 2376: `sudo ufw status`
- Verify Docker is listening: `sudo netstat -tlnp | grep 2376`
- Test from Docker host: `docker --tlsverify -H tcp://127.0.0.1:2376 version`

### TLS Certificate Errors

- Ensure all three certificates are set correctly
- Check certificates haven't expired: `openssl x509 -in cert.pem -noout -dates`
- Verify newlines are preserved in environment variables

### Docker Daemon Logs

On the Docker host:

```bash
sudo journalctl -u docker -f
```

## Estimated Costs

| Component      | Provider         | Cost/Month |
| -------------- | ---------------- | ---------- |
| Docker Host VM | Hetzner CX22     | ~€4.50     |
| Docker Host VM | DigitalOcean     | ~$6        |
| Docker Host VM | Fly.io           | ~$7        |
| Rexec API      | Railway/PipeOps  | ~$5-10     |
| PostgreSQL     | Railway/Supabase | $0-5       |
| Redis          | Upstash          | $0-5       |

**Total: ~$15-25/month** for a basic setup
