# Deploying Rexec on PaaS (Remote Docker)

This guide explains how to deploy Rexec on Platform-as-a-Service (PaaS) providers like Railway, PipeOps, Render, or Fly.io, where running privileged containers is not possible.

Instead of running Docker locally, Rexec connects to a remote Docker daemon over TLS.

## Architecture

```
[PaaS Container (Rexec)]  ----(TLS/TCP)---->  [Remote VM (Docker Daemon)]
      │                                              │
      └─> [PostgreSQL / Redis]                       └─> [User Containers]
```

## Prerequisites

1. **A PaaS account** (Railway, PipeOps, Render, etc.) to host the Rexec API
2. **A Linux VM** (Hetzner, DigitalOcean, Linode, Fly.io, etc.) with Docker installed
3. **PostgreSQL & Redis** (managed or self-hosted)

## Step 1: Set Up the Remote Docker Host

### Option A: Automated Setup (Recommended)

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

### Option B: Manual Setup

1. **Install Docker** on your VM:

   ```bash
   curl -fsSL https://get.docker.com | sudo sh
   ```

2. **Generate TLS certificates**:

   ```bash
   sudo ./scripts/setup-docker-tls.sh YOUR_VM_PUBLIC_IP
   ```

3. **Configure Docker daemon** (`/etc/docker/daemon.json`):

   ```json
   {
     "hosts": ["unix:///var/run/docker.sock", "tcp://0.0.0.0:2376"],
     "tls": true,
     "tlscacert": "/etc/docker/certs/ca.pem",
     "tlscert": "/etc/docker/certs/server-cert.pem",
     "tlskey": "/etc/docker/certs/server-key.pem",
     "tlsverify": true
   }
   ```

4. **Create systemd override** (`/etc/systemd/system/docker.service.d/override.conf`):

   ```ini
   [Service]
   ExecStart=
   ExecStart=/usr/bin/dockerd
   ```

5. **Restart Docker**:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart docker
   ```

6. **Open firewall port**:
   ```bash
   sudo ufw allow 2376/tcp
   ```

## Step 2: Configure PaaS Environment Variables

Set these environment variables in your PaaS dashboard:

### Required Variables

| Variable             | Description                    |
| -------------------- | ------------------------------ |
| `DOCKER_HOST`        | `tcp://YOUR_VM_IP:2376`        |
| `DOCKER_TLS_VERIFY`  | `1`                            |
| `DOCKER_CA_CERT`     | Contents of `ca.pem`           |
| `DOCKER_CLIENT_CERT` | Contents of `cert.pem`         |
| `DOCKER_CLIENT_KEY`  | Contents of `key.pem`          |
| `DATABASE_URL`       | PostgreSQL connection string   |
| `JWT_SECRET`         | Secret for signing auth tokens |

### Optional Variables

| Variable            | Description                           |
| ------------------- | ------------------------------------- |
| `REDIS_URL`         | Redis connection string (recommended) |
| `PORT`              | API port (usually set by PaaS)        |
| `STRIPE_SECRET_KEY` | For billing features                  |

## Step 3: Deploy

1. Use `Dockerfile.remote` for your deployment:

   ```bash
   docker build -f Dockerfile.remote -t rexec:latest .
   ```

2. Push to your container registry or let the PaaS build it

3. Set all environment variables in the PaaS dashboard

4. Deploy and verify the health endpoint:
   ```bash
   curl https://your-app.railway.app/health
   ```

## Estimated Costs

| Component        | Provider         | Cost/Month |
| ---------------- | ---------------- | ---------- |
| Docker Host VM   | Hetzner CX22     | ~€4.50     |
| Docker Host VM   | DigitalOcean     | ~$6        |
| Docker Host VM   | Fly.io           | ~$7        |
| PaaS (Rexec API) | Railway          | ~$5-10     |
| PostgreSQL       | Railway/Supabase | $0-5       |
| Redis            | Upstash          | $0-5       |

**Total: ~$15-25/month** for a basic setup

## Security Considerations

- **TLS Required**: Always use TLS (port 2376). Never expose Docker without TLS.
- **Firewall**: Restrict port 2376 to PaaS IP ranges if possible.
- **Certificate Rotation**: Regenerate certificates periodically.
- **Resource Limits**: Configure Docker to limit container resources.
- **Monitoring**: Set up alerts for unusual Docker API activity.

## Troubleshooting

### Connection Refused

- Check firewall allows port 2376
- Verify Docker is listening: `sudo netstat -tlnp | grep 2376`

### TLS Handshake Failed

- Verify all three certificates are set correctly
- Check certificate hasn't expired: `openssl x509 -in cert.pem -noout -dates`

### Certificate Errors

- Ensure certificates are copied with newlines preserved
- Verify `DOCKER_TLS_VERIFY=1` is set

### Docker Logs

```bash
sudo journalctl -u docker -f
```
