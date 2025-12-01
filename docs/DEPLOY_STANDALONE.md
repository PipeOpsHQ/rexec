# Standalone Deployment (All-in-One)

This guide explains how to deploy Rexec in "Standalone Mode". In this configuration, the Rexec API and a **Rootless Docker Daemon** run together inside a single container.

This removes the need for a separate remote Docker host or mounting the host's Docker socket.

## Architecture

```
[ Container (Rexec API + Rootless Docker) ]
      │
      └─> [PostgreSQL / Redis]
```

## When to use this?

- **Simplicity**: You want a single artifact to deploy.
- **Development**: You want to test Rexec without setting up SSH keys or external servers.
- **Self-Hosted**: You are running on a VPS or bare metal and want strong isolation between Rexec and the host system.

## Prerequisites

- Docker installed on your build machine.
- A PostgreSQL database.

## Building the Image

This method uses a special Dockerfile: `Dockerfile.standalone`.

```bash
docker build -f Dockerfile.standalone -t rexec-standalone .
```

## Running Locally

You can run the standalone image locally to test it. Note that even though it is "rootless" inside, the container itself often requires the `--privileged` flag to function correctly (specifically for mounting filesystems and cgroups), depending on your host kernel version.

```bash
docker run --privileged \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/rexec" \
  -e JWT_SECRET="your-secret" \
  rexec-standalone
```

## Using Docker Compose

For a complete local environment with PostgreSQL and Redis, use the provided compose file:

```bash
cd docker
docker compose -f docker-compose.standalone.yml up --build
```

## Deploying to PaaS (Railway, Fly.io, etc.)

Many modern PaaS providers support running Docker-in-Docker (DinD).

### Railway.app

1.  **New Project** -> **GitHub Repo**.
2.  **Settings** -> **Build**:
    - Set **Dockerfile Path** to `Dockerfile.standalone`.
3.  **Variables**:
    - `DATABASE_URL`: Connect to a Postgres service.
    - `JWT_SECRET`: Random string.
    - `PORT`: `8080`.
4.  **Service Settings**:
    - Enable **Privileged Mode** (often required for nested Docker functionality).

### Fly.io

Fly.io supports this natively via their Firecracker microVMs.

1.  Create `fly.toml`.
2.  Update the `[build]` section:
    ```toml
    [build]
      dockerfile = "Dockerfile.standalone"
    ```
3.  Deploy.

## How it Works

1.  The image is based on `docker:27-dind-rootless`.
2.  It creates a non-root user (`rootless`, UID 1000).
3.  The entrypoint script (`start-standalone.sh`) starts the Docker daemon in the background.
4.  Once the daemon is ready, it starts the Rexec API.
5.  Rexec connects to the local daemon via the unix socket at `/run/user/1000/docker.sock`.

## Limitations

- **Performance**: Running Docker inside Docker (even rootless) has a slight storage performance overhead due to nested overlayfs.
- **Ephemeral Storage**: If the container restarts, all user containers and volumes created _inside_ it are lost unless you mount a persistent volume to `/var/lib/rexec`.
