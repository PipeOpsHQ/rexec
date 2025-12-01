# Deploying Rexec on PaaS (Remote Docker)

This guide explains how to deploy Rexec on Platform-as-a-Service (PaaS) providers like Heroku, Railway, Render, or Fly.io, where running a privileged Docker daemon or mounting the host Docker socket is not possible or recommended.

Instead of running Docker locally, Rexec can connect to a remote Docker engine via SSH.

## Architecture

```
[PaaS Container (Rexec)]  ----(SSH)---->  [Remote VPS (Docker Engine)]
      │                                          │
      └─> [PostgreSQL / Redis]                   └─> [User Containers]
```

## Prerequisites

1.  **A PaaS account** (Railway, Render, Heroku, etc.) to host the Rexec API.
2.  **A Linux VPS** (DigitalOcean, Hetzner, AWS EC2, etc.) with Docker installed to host the user containers.
3.  **PostgreSQL & Redis** (managed or self-hosted).

## Step 1: Prepare the Remote Docker Host

1.  **Install Docker** on your VPS.
2.  **Create a dedicated user** for Rexec (optional but recommended):
    ```bash
    useradd -m -s /bin/bash rexec
    usermod -aG docker rexec
    ```
3.  **Generate an SSH Key Pair** on your local machine (do not use a passphrase):
    ```bash
    ssh-keygen -t ed25519 -f rexec_key -C "rexec-paas"
    ```
4.  **Add the Public Key** to the remote host:
    ```bash
    # On the remote host, as the 'rexec' user
    mkdir -p ~/.ssh
    chmod 700 ~/.ssh
    echo "content-of-rexec_key.pub" >> ~/.ssh/authorized_keys
    chmod 600 ~/.ssh/authorized_keys
    ```
5.  **Verify Connection**:
    Ensure you can SSH from your local machine using the private key:
    ```bash
    ssh -i rexec_key rexec@your-vps-ip docker info
    ```

## Step 2: Configure the PaaS Application

Deploy the Rexec Docker image to your PaaS provider. You will need to set the following environment variables:

### Required Environment Variables

| Variable | Value | Description |
|----------|-------|-------------|
| `DOCKER_HOST` | `ssh://rexec@your-vps-ip` | Connection string for the remote Docker engine. |
| `SSH_PRIVATE_KEY` | `-----BEGIN OPENSSH PRIVATE KEY...` | The content of the `rexec_key` file you generated. |
| `JWT_SECRET` | `your-secret` | Secret for signing auth tokens. |
| `DATABASE_URL` | `postgres://...` | Connection string for PostgreSQL. |

### Optional Environment Variables

| Variable | Value | Description |
|----------|-------|-------------|
| `REDIS_URL` | `redis://...` | Connection string for Redis (recommended). |
| `PORT` | `8080` | The port the API listens on (PaaS usually sets this automatically). |

## Step 3: Deploy

1.  **Build and Push** the Rexec image (or use the official one if available).
2.  **Set the Environment Variables** in your PaaS dashboard.
    *   *Note*: When pasting the `SSH_PRIVATE_KEY`, ensure all newlines are preserved.
3.  **Start the Service**.

Rexec will automatically:
1.  Detect the `SSH_PRIVATE_KEY`.
2.  Configure the SSH client inside the container.
3.  Connect to the remote Docker engine defined in `DOCKER_HOST`.

## Security Considerations

*   **Network Isolation**: Ensure your Remote Docker Host firewall allows SSH connections (port 22) from the PaaS IP addresses (or `0.0.0.0/0` if IPs are dynamic, secured by the SSH key).
*   **User Permissions**: The `rexec` user on the remote host has access to the Docker socket, which is effectively root access on that host. Treat this host as a dedicated infrastructure component.
*   **Resource Limits**: Configure Docker on the remote host to limit resources per container to prevent one user from consuming all CPU/RAM.
