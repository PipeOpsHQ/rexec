# Rexec — Terminal as a Service

Rexec gives “10x” engineers an instantly-available, network-isolated Linux terminal in the cloud with first-class container management, SSH access, and Stripe-backed billing.

---

## Why Rexec

- **Container lifecycle on demand** – Create, start, stop, and delete per-user sandboxes powered by Docker.
- **First-class terminal UX** – Real-time streams over WebSockets with `xterm.js`, JetBrains Mono typography, and macOS-style chrome.
- **Persistent developer context** – Named volumes mounted at `/home/user` so work survives restarts.
- **Security-aware defaults** – JWT auth, rate limiting, no-new-privileges Docker profiles.
- **Payments built in** – Stripe subscriptions, checkout, and customer portal flows wired into the API.
- **Ops visibility** – Health checks, stats endpoints, container cleanup workers, and structured logging.

---

## Architecture at a Glance

```
[Browser UI] ←→ [Gin API] ←→ [PostgreSQL]
      │                       │
      └── WebSocket / REST ───┤
                              ↓
                        [Container Manager]
                              │
                        [Docker Engine]
```

| Layer       | Tech                                                  |
| ----------- | ----------------------------------------------------- |
| Frontend    | HTML/CSS/JS, `xterm.js`, Inter + JetBrains Mono fonts |
| API         | Go 1.22+, Gin, JWT, custom middleware                 |
| Workers     | Container cleanup service, Stripe webhook handler     |
| Persistence | PostgreSQL (users/containers)                         |
| Billing     | Stripe SDK (`stripe-go`)                              |
| Containers  | Docker SDK with Ubuntu/Debian/Alpine/Fedora images    |

---

## Quickstart

### Prerequisites

- Go 1.22 or newer
- Docker Engine with socket access
- `make`, `git`, `bash`

### Local build & run

```bash
git clone https://github.com/rexec/rexec.git
cd rexec
# create a .env with at least JWT_SECRET (see Configuration)
make setup              # installs deps, pulls base images, prepares volumes
make run                # builds + starts the API on :8080
```

The server will look for the UI in `./web`. Point your browser at `http://localhost:8080/`.

### Docker Compose

```bash
cd docker
docker compose up --build
```

This spins up the API and PostgreSQL with volumes persisted via Docker.

---

## Configuration

| Variable                  | Default / Notes                                                |
| ------------------------- | -------------------------------------------------------------- |
| `PORT`                    | `8080` – API listen port                                       |
| `DATABASE_URL`            | `postgres://rexec:rexec@localhost:5432/rexec?sslmode=disable`  |
| `JWT_SECRET`              | **Required** – used for signing auth tokens                    |
| `STRIPE_SECRET_KEY`       | Enables billing endpoints when set                             |
| `STRIPE_WEBHOOK_SECRET`   | Required for webhook verification                              |
| `STRIPE_PRICE_PRO`        | Stripe price ID for the Pro plan                               |
| `STRIPE_PRICE_ENTERPRISE` | Price ID for Enterprise tier                                   |
| `WEB_DIR`                 | Directory containing `index.html` (defaults to `./web`)        |
| `GIN_MODE`                | `release` or `debug` – influences container cleanup thresholds |

### S3 Storage for Recordings (Optional)

Store terminal recordings in S3-compatible storage for horizontal scaling:

| Variable               | Description                                              |
| ---------------------- | -------------------------------------------------------- |
| `S3_BUCKET`            | S3 bucket name (enables S3 storage when set)             |
| `S3_REGION`            | AWS region (default: `us-east-1`)                        |
| `S3_ENDPOINT`          | Custom endpoint for S3-compatible services (MinIO, etc.) |
| `S3_ACCESS_KEY_ID`     | AWS access key ID                                        |
| `S3_SECRET_ACCESS_KEY` | AWS secret access key                                    |
| `S3_PREFIX`            | Key prefix for recordings (e.g., `recordings/`)          |
| `S3_FORCE_PATH_STYLE`  | Set to `true` for MinIO and some S3-compatible services  |

---

## Make Targets

| Command             | Purpose                                         |
| ------------------- | ----------------------------------------------- |
| `make build`        | Compile the API binary into `bin/rexec`         |
| `make run`          | Build and start the API locally                 |
| `make dev`          | Hot-reload via `air` (auto-installs if missing) |
| `make test`         | Run Go test suite                               |
| `make docker-build` | Build production Docker image                   |
| `make docker-run`   | Bring up the stack via Compose                  |
| `make images`       | Build custom user images with SSH baked in      |

---

## UI Vision – “Compact, Premium Terminal Control Room”

The refreshed interface takes cues from PipeOps Load Tester while doubling down on developer ergonomics:

- **Dark titanium palette** with neon violet accents and subtle glass highlights.
- **Tight grid system**: 12px baseline rhythm, 8–12px radius cards, condensed inter-card spacing for a dashboard feel.
- **Hero panel**: Minimal copy, a live terminal preview, and CTA buttons with soft glow hover states.
- **Dashboard cards**: Status light, container meta (image, uptime, CPU/RAM sparkline), and inline action buttons.
- **Terminal workspace**: macOS-style chrome (● ● ●), breadcrumb header, SSH quick connect, and contextual status pills (Running / Sleeping / Building).
- **Command palette** (⌘K / Ctrl+K): Jump to containers, trigger actions, or copy SSH instructions without leaving the keyboard.
- **Modals & Forms**: Two-column layout for advanced settings, inline validation, and monospace previews of `docker run` equivalents.
- **Toasts & Activity Feed**: Slide-in notifications for lifecycle events plus a right-rail log of API actions for auditability.
- **Visual feedback for errors** (e.g., JSON parse issues) with precise callouts so backend misconfigurations are obvious.

The result should _feel_ like a premium IDE sidebar rather than a generic admin dashboard—fast, minimal, and optimized for people who live in terminals.

---

## Operational Tips

1. **Return JSON everywhere** – Ensure API errors aren’t responded with HTML (prevents “Unexpected token `<`” in the UI).
2. **Keep Docker tidy** – The cleanup service removes idle containers; tune `CleanupConfig` for staging vs. production.
3. **Stripe webhooks** – Run `stripe listen --forward-to localhost:8080/api/billing/webhook` during local billing work.
4. **SSH keys** – Users can upload keys, sync into containers, and retrieve connection commands via `/api/ssh/*` endpoints.

---

## Roadmap

- Command palette + keyboard shortcuts baked into the UI.
- Metrics overlays (CPU, memory, bandwidth per container) with mini charts.
- Multi-region container pools.
- “Snapshots” for one-click environment cloning.
- Optional SSO (SAML/OIDC) for enterprise teams.

Rexec is already production-capable; polish the UI, harden the API responses, and invite your favorite power users to try the new experience.

---

## FAQ

### Is Rexec a Virtual Machine?

**No.** Rexec is not a VM. It's a **Terminal as a Service** platform that gives you instant access to containerized Linux environments or your own machines via the agent. Think of it as a cloud-based terminal multiplexer, not a hypervisor.

### What's the difference between Rexec containers and the Agent?

| Feature        | Container                               | Agent                                                   |
| -------------- | --------------------------------------- | ------------------------------------------------------- |
| **What it is** | A Docker container provisioned by Rexec | A lightweight binary running on your existing machine   |
| **Setup**      | One click in the dashboard              | Install agent, register, done                           |
| **Resources**  | Dedicated, isolated container           | Uses your machine's resources                           |
| **Use case**   | Disposable dev environments, sandboxes  | Access your laptop, servers, Raspberry Pi from anywhere |
| **Network**    | Isolated container network              | Your machine's full network                             |

### Can I access my own laptop/server remotely?

**Yes!** This is exactly what the **Rexec Agent** is for. Install the agent on any machine, and you get instant terminal access from the Rexec dashboard—no SSH port exposure, no VPN, no complex setup.

```bash
# Install and register your machine in 30 seconds
curl -sSL https://rexec.pipeops.io/install-agent.sh | bash
rexec-agent register --name "my-laptop"
rexec-agent start --daemon
```

Now you can access your laptop's terminal from any browser, phone, or AI CLI tool.

### Why would I use Rexec instead of SSH?

- **No inbound ports** – The agent connects outbound, so no firewall holes needed
- **Works from anywhere** – Access from browser, mobile, or AI coding tools
- **No VPN required** – Direct access without complex network setup
- **Unified dashboard** – Manage all your machines (containers + agents) in one place
- **AI-native** – Built for integration with AI coding assistants

### Can I use Rexec with AI coding tools like Claude, Cursor, or Windsurf?

**Absolutely.** Rexec was designed with AI-native workflows in mind. Register your dev machine as an agent, and you can continue building in your environment from any AI CLI without opening your laptop.

### What happens if my connection drops?

The agent automatically reconnects with exponential backoff. Your terminal sessions are maintained, and you can resume right where you left off.

### Is it secure?

- All traffic is encrypted via WSS (WebSocket Secure)
- No inbound ports required on your machines
- JWT-based authentication
- Tokens can be rotated anytime
- See [SECURITY.md](docs/SECURITY.md) for details

### What operating systems are supported?

**Containers:** Ubuntu, Debian, Alpine, Fedora (pre-built images)

**Agent:** Linux (amd64, arm64), macOS (amd64, arm64), Windows (experimental)

---

# rexec

- Subtle addition - Keep existing design, just add a small AI tools section below
  - Hero variation - Add a secondary tagline mentioning AI tools
  - Separate page - Create a dedicated /ai-tools or /vibe-coder page
  - Feature cards - Add AI tools as additional feature cards in existing grid
