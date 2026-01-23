# Rexec - Terminal as a Service

[![GitHub stars](https://img.shields.io/github/stars/PipeOpsHQ/rexec?style=social)](https://github.com/PipeOpsHQ/rexec)
[![GitHub forks](https://img.shields.io/github/forks/PipeOpsHQ/rexec?style=social)](https://github.com/PipeOpsHQ/rexec/fork)
[![GitHub issues](https://img.shields.io/github/issues/PipeOpsHQ/rexec)](https://github.com/PipeOpsHQ/rexec/issues)
[![GitHub license](https://img.shields.io/github/license/PipeOpsHQ/rexec)](https://github.com/PipeOpsHQ/rexec/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/PipeOpsHQ/rexec)](https://go.dev/)
[![Docker Pulls](https://img.shields.io/docker/pulls/pipeopshq/rexec)](https://ghcr.io/pipeopshq/rexec)

Rexec is an open-source platform that gives you instantly-available, network-isolated Linux terminals in the cloud, or lets you connect your own machines to a unified dashboard. Built for developers who need on-demand environments and secure remote access.

---

## Features

*   **Instant Cloud Terminals**: Create, start, and destroy disposable Linux sandboxes in seconds (powered by Docker).
*   **Connect Any Machine (BYOS)**: Install the lightweight Rexec Agent on your laptop, server, or Raspberry Pi to access it securely from the browser without VPNs or SSH port exposure.
*   **First-Class Terminal UX**: Real-time WebSocket streaming with `xterm.js`, JetBrains Mono fonts, and a native-feeling UI.
*   **Secure by Default**: JWT authentication, MFA support, audit logging, and isolated container networking.
*   **Collaboration**: Share terminal sessions for pair programming or debugging.
*   **Session Recording**: Record and replay terminal sessions for documentation or audit trails.

---

## Quick Start

### Self-Hosting with Docker Compose

The easiest way to run Rexec is with Docker Compose.

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/rexec/rexec.git
    cd rexec/docker
    ```

2.  **Start the stack:**
    ```bash
    docker compose up --build
    ```

3.  **Access the UI:**
    Open your browser to `http://localhost:8080`.

    *   **Username:** `admin`
    *   **Password:** `admin` (Change this immediately in production!)

### Manual Installation (Development)

**Prerequisites:** Go 1.22+, Docker Engine, Node.js (for frontend)

1.  **Clone and Setup:**
    ```bash
    git clone https://github.com/rexec/rexec.git
    cd rexec
    make setup
    ```

2.  **Run the API:**
    ```bash
    make run
    ```

3.  **Run the Frontend:**
    ```bash
    cd frontend
    npm install
    npm run dev
    ```

---

## Connecting Your Machines (Agent)

Rexec allows you to connect your own infrastructure (laptops, VMs, bare metal) to the dashboard using the Rexec Agent.

1.  Log in to your Rexec instance.
2.  Go to **Settings > Agents**.
3.  Click **Add Agent** to generate an installation command.
4.  Run the command on your target machine:

```bash
# Example command (get your specific token from the dashboard)
curl -fsSL http://localhost:8080/install-agent.sh | sudo bash -s -- --token YOUR_TOKEN
```

The agent establishes a secure outbound WebSocket connection to your Rexec server. No firewall changes or inbound ports required.

---

## Configuration

Rexec is configured via environment variables. Create a `.env` file in the root directory or pass them to Docker.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | API listen port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://...` |
| `JWT_SECRET` | **Required.** Secret for signing auth tokens | (Random if unset) |
| `GIN_MODE` | Web framework mode (`debug` or `release`) | `debug` |
| `S3_BUCKET` | S3 bucket for storing session recordings | (Optional) |
| `S3_REGION` | S3 region | `us-east-1` |
| `S3_ENDPOINT` | Custom S3 endpoint (e.g., MinIO) | (Optional) |

See `.env.example` for a full list of options.

---

## Architecture

```
[Browser UI] ←(WebSocket)→ [Rexec API] ←→ [PostgreSQL]
                                │
                                ├── [Container Manager] ──→ [Docker Engine]
                                │
                                └── [Agent Handler] ←(WebSocket)→ [Remote Agents]
```

*   **Frontend:** Svelte, xterm.js, Tailwind CSS
*   **Backend:** Go (Gin), Gorilla WebSocket
*   **Database:** PostgreSQL
*   **Runtime:** Docker (for cloud terminals)

---

## SDKs

Integrate Rexec into your applications with our official SDKs:

### Go SDK

```bash
go get github.com/PipeOpsHQ/rexec-go
```

```go
client := rexec.NewClient("https://your-instance.com", "your-token")
container, _ := client.Containers.Create(ctx, &rexec.CreateContainerRequest{
    Image: "ubuntu:24.04",
})
term, _ := client.Terminal.Connect(ctx, container.ID)
term.Write([]byte("echo hello\n"))
```

### JavaScript/TypeScript SDK

```bash
npm install @pipeopshq/rexec
```

```typescript
const client = new RexecClient({
  baseURL: 'https://your-instance.com',
  token: 'your-token'
});
const container = await client.containers.create({ image: 'ubuntu:24.04' });
const terminal = await client.terminal.connect(container.id);
terminal.write('echo hello\n');
```

### Python SDK

```bash
pip install rexec
```

```python
async with RexecClient("https://your-instance.com", "your-token") as client:
    container = await client.containers.create(image="ubuntu:24.04")
    async with client.terminal.connect(container.id) as term:
        await term.write(b"echo hello\n")
```

### Rust SDK

```bash
cargo add rexec tokio
```

```rust
let client = RexecClient::new("https://your-instance.com", "your-token");
let container = client.containers()
    .create(CreateContainerRequest::new("ubuntu:24.04")).await?;
let mut term = client.terminal().connect(&container.id).await?;
term.write(b"echo hello\n").await?;
```

📚 [Full SDK Documentation](docs/SDK.md)

---

## Roadmap

*   [ ] **Command Palette**: `Cmd+K` navigation for power users.
*   [ ] **Metrics Overlays**: Real-time CPU/RAM usage charts per container.
*   [ ] **Multi-Region**: Support for container pools in different geographic locations.
*   [ ] **Snapshots**: One-click environment cloning.
*   [ ] **SSO**: SAML/OIDC support for enterprise teams.

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

## Community & Support

*   **GitHub**: [github.com/PipeOpsHQ/rexec](https://github.com/PipeOpsHQ/rexec)
*   **Issues**: [GitHub Issues](https://github.com/PipeOpsHQ/rexec/issues)
*   **Discussions**: [GitHub Discussions](https://github.com/PipeOpsHQ/rexec/discussions)


Built with ❤️ for the 10x engineer in everyone.
