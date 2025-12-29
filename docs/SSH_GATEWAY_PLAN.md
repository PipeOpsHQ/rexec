# SSH Gateway Plan: terminal.shop-Style Access for Rexec

> **Goal**: Allow users to access Rexec by simply running `ssh rexec.io` — no CLI install required, instant TUI dashboard in any terminal.

---

## 1. Concept Overview

### What is terminal.shop?

terminal.shop is a coffee shop accessible via SSH. Users run `ssh terminal.shop` and get a beautiful TUI to browse and purchase coffee — no signup, no app install.

### What We're Building

An SSH server that presents the Rexec TUI dashboard when users connect. This enables:

- **Zero-install access**: Just `ssh rexec.io` from any machine
- **Zero-config auth**: Server accepts ANY SSH key — no registration required
- **Universal compatibility**: Works from any terminal (Linux, macOS, Windows, mobile)
- **Instant onboarding**: New users can explore immediately, link account later
- **Secure terminal bridging**: Connect to containers/agents directly through SSH

---

## 2. Architecture

```
┌─────────────────┐         ┌─────────────────────────────────────────┐
│   User's Term   │   SSH   │              SSH Gateway                │
│                 │ ──────► │  ┌─────────────────────────────────┐    │
│  ssh rexec.io   │         │  │  wish Server (port 22/2222)     │    │
└─────────────────┘         │  │                                 │    │
                            │  │  ┌───────────────────────────┐  │    │
                            │  │  │  Bubble Tea TUI Session   │  │    │
                            │  │  │  - Dashboard              │  │    │
                            │  │  │  - Terminal list          │  │    │
                            │  │  │  - Snippets               │  │    │
                            │  │  │  - Agent management       │  │    │
                            │  │  └───────────────────────────┘  │    │
                            │  └─────────────────────────────────┘    │
                            │                    │                    │
                            │                    ▼                    │
                            │  ┌─────────────────────────────────┐    │
                            │  │  Rexec API (internal)           │    │
                            │  │  - Auth, Containers, Agents     │    │
                            │  └─────────────────────────────────┘    │
                            │                    │                    │
                            │                    ▼                    │
                            │  ┌─────────────────────────────────┐    │
                            │  │  Container/Agent WebSocket      │    │
                            │  │  Terminal Bridge                │    │
                            │  └─────────────────────────────────┘    │
                            └─────────────────────────────────────────┘
```

---

## 3. Tech Stack

| Component       | Library                                       | Purpose                               |
| --------------- | --------------------------------------------- | ------------------------------------- |
| SSH Server      | `github.com/charmbracelet/wish`               | Go SSH server framework by Charm      |
| TUI Framework   | `github.com/charmbracelet/bubbletea`          | Already in use for rexec-tui          |
| Styling         | `github.com/charmbracelet/lipgloss`           | Already in use                        |
| Auth            | `github.com/charmbracelet/wish/accesscontrol` | SSH key + password auth               |
| Terminal Bridge | Custom PTY relay                              | Bridge SSH session to container/agent |

---

## 4. Authentication Methods

### 4.1 Zero-Config Access (Like terminal.shop)

**Key insight**: terminal.shop accepts ANY SSH public key without pre-registration. The server simply accepts whatever key the user offers and uses the key fingerprint as a unique identifier.

```bash
# Just connect - no setup required!
ssh rexec.io

# Server accepts your existing SSH key automatically
# Your key fingerprint becomes your identity
# First connection = new guest session
# Subsequent connections = recognized returning user
```

**How it works**:

1. User runs `ssh rexec.io`
2. SSH client offers user's default public key (e.g., `~/.ssh/id_ed25519.pub`)
3. Server **accepts any valid key** (no pre-registration required)
4. Server extracts key fingerprint (e.g., `SHA256:abc123...`)
5. Fingerprint lookup:
   - Known fingerprint → Load existing user session
   - Unknown fingerprint → Create guest session, prompt to link account

```go
// wish middleware - accept all keys
func publicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
    // Accept ALL valid public keys
    // Store fingerprint in context for session lookup
    fingerprint := gossh.FingerprintSHA256(key)
    ctx.SetValue("fingerprint", fingerprint)
    return true  // Always accept
}
```

### 4.2 Account Linking (Optional)

Users can optionally link their SSH key to a Rexec account for persistent data:

```bash
ssh rexec.io
# In TUI: Settings → Link Account → Enter email/token
# Now this SSH key is associated with your Rexec account
```

Or via the web UI / CLI:

```bash
# Get your fingerprint
ssh-keygen -lf ~/.ssh/id_ed25519.pub

# Link via CLI
rexec ssh-key link --fingerprint "SHA256:abc123..."
```

### 4.3 Token-Based Authentication (Fallback)

For environments without SSH keys or for explicit account access:

```bash
ssh token@rexec.io
# Password prompt accepts API token

# Or with username:
ssh myusername@rexec.io
# Password prompt for token/password
```

### 4.4 Direct Terminal Access

```bash
# Jump directly to a specific terminal
ssh terminal-abc123@rexec.io

# Or agent:
ssh agent-myserver@rexec.io

# These require the SSH key to be linked to an account with access
```

---

## 5. Implementation Phases

### Phase 1: Basic SSH Server + TUI (2-3 weeks)

**New binary**: `cmd/rexec-ssh/main.go`

**Tasks**:

- [ ] Set up wish SSH server with host key generation
- [ ] Integrate existing Bubble Tea TUI from `cmd/rexec-tui`
- [ ] Implement guest access (no auth, read-only demo)
- [ ] Basic connection handling and session management
- [ ] Health check endpoint for load balancers

**Deliverable**: Users can `ssh guest@rexec.io` and see a demo dashboard

### Phase 2: Authentication (1-2 weeks)

**Tasks**:

- [ ] SSH public key registration API (`POST /api/ssh-keys`)
- [ ] SSH key lookup middleware in wish
- [ ] Token-based password authentication
- [ ] Session creation and JWT bridging to API
- [ ] Rate limiting and brute-force protection

**Database additions**:

```sql
-- SSH fingerprints - can exist without a linked user (guest sessions)
CREATE TABLE ssh_fingerprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint TEXT NOT NULL UNIQUE,          -- SHA256:abc123...
    user_id UUID REFERENCES users(id),         -- NULL for unlinked guests
    public_key TEXT,                           -- Optional, for display
    key_type TEXT,                             -- ed25519, rsa, etc.
    first_seen_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    last_ip TEXT,
    session_count INT DEFAULT 0
);

-- Guest session data (for unlinked fingerprints)
CREATE TABLE ssh_guest_sessions (
    fingerprint_id UUID REFERENCES ssh_fingerprints(id),
    preferences JSONB DEFAULT '{}',
    demo_terminals_created INT DEFAULT 0,
    PRIMARY KEY (fingerprint_id)
);
```

**Deliverable**: Anyone can `ssh rexec.io` and get a session immediately

### Phase 3: Terminal Bridging (2-3 weeks)

**Tasks**:

- [ ] WebSocket → PTY bridge for container terminals
- [ ] Agent terminal bridging through SSH
- [ ] Terminal size (PTY window) synchronization
- [ ] MFA challenge integration in TUI
- [ ] Session recording for SSH sessions
- [ ] Graceful disconnection handling

**Deliverable**: Full terminal access through SSH gateway

### Phase 4: Advanced Features (2 weeks)

**Tasks**:

- [ ] Direct terminal access via username (`ssh terminal-ID@rexec.io`)
- [ ] SSH tunneling for port forwarding
- [ ] SCP/SFTP support for file transfers
- [ ] Multi-session support (tmux-style)
- [ ] Session sharing (pair programming)

**Deliverable**: Feature parity with web terminal + SSH-specific extras

### Phase 5: Production Hardening (1-2 weeks)

**Tasks**:

- [ ] Audit logging for all SSH connections
- [ ] Connection limits per user
- [ ] Geographic access controls (optional)
- [ ] DDoS protection at edge
- [ ] Metrics and alerting (Prometheus/Grafana)
- [ ] Host key rotation strategy

**Deliverable**: Production-ready SSH gateway

---

## 6. File Structure

```
cmd/
├── rexec-ssh/
│   ├── main.go           # SSH server entry point
│   ├── auth.go           # Authentication handlers
│   ├── session.go        # Session management
│   └── bridge.go         # Terminal bridging
│
internal/
├── ssh/
│   ├── server.go         # wish server setup
│   ├── middleware.go     # Auth, logging, rate limiting
│   ├── keys.go           # SSH key management
│   └── tui/
│       ├── app.go        # Main TUI application (refactored from rexec-tui)
│       ├── dashboard.go  # Dashboard view
│       ├── terminals.go  # Terminal list/management
│       ├── agents.go     # Agent management
│       └── styles.go     # Shared lipgloss styles
```

---

## 7. TUI Views & Navigation

```
### Linked User Dashboard
┌────────────────────────────────────────────────────────────────────┐
│  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗                         │
│  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝                         │
│  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║                              │
│  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║                              │
│  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗                         │
│  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝  Terminal Cloud         │
├────────────────────────────────────────────────────────────────────┤
│  Welcome, user@example.com                          Pro Plan │ ◉  │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  [1] 📦 Terminals (5)      Active containers and sessions         │
│  [2] 🤖 Agents (3)         Remote servers and VMs                 │
│  [3] 📝 Snippets (12)      Saved commands and scripts             │
│  [4] ➕ Create             Launch new terminal                    │
│  [5] ⚙️  Settings           Account and preferences                │
│  [q] 🚪 Quit               Exit SSH session                       │
│                                                                    │
├────────────────────────────────────────────────────────────────────┤
│  ↑/↓ Navigate  •  Enter Select  •  ? Help  •  q Quit              │
└────────────────────────────────────────────────────────────────────┘

### Guest Dashboard (First Visit / Unlinked Key)
┌────────────────────────────────────────────────────────────────────┐
│  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗                         │
│  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝                         │
│  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║                              │
│  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║                              │
│  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗                         │
│  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝  Terminal Cloud         │
├────────────────────────────────────────────────────────────────────┤
│  👋 Welcome, Guest!                    SSH Key: ed25519 SHA256:xY │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │ 🔗 Link your account for full access: terminals, agents,   │   │
│  │    snippets, and more. Press [L] to link now.              │   │
│  └────────────────────────────────────────────────────────────┘   │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  [1] 🎮 Try Demo Terminal   Launch a temporary sandbox            │
│  [2] 👀 Explore Features    See what Rexec can do                 │
│  [L] 🔗 Link Account        Connect to your Rexec account         │
│  [S] 📝 Sign Up             Create a new account                  │
│  [q] 🚪 Quit                Exit SSH session                      │
│                                                                    │
├────────────────────────────────────────────────────────────────────┤
│  ↑/↓ Navigate  •  Enter Select  •  ? Help  •  q Quit              │
└────────────────────────────────────────────────────────────────────┘
```

---

## 8. Configuration

### Environment Variables

```bash
# SSH Gateway config
SSH_GATEWAY_PORT=2222           # SSH listen port (22 requires root)
SSH_GATEWAY_HOST_KEY=/etc/rexec/ssh_host_key
SSH_GATEWAY_API_URL=http://localhost:8080
SSH_GATEWAY_MAX_CONNECTIONS=1000
SSH_GATEWAY_SESSION_TIMEOUT=24h
SSH_GATEWAY_IDLE_TIMEOUT=30m

# Rate limiting
SSH_GATEWAY_RATE_LIMIT=10       # connections per minute per IP
SSH_GATEWAY_BURST=5
```

### Host Key Generation

```bash
# Generate on first run or via setup script
ssh-keygen -t ed25519 -f /etc/rexec/ssh_host_key -N ""
```

---

## 9. API Additions

### SSH Key Management

```
POST   /api/ssh-keys           # Register new SSH key
GET    /api/ssh-keys           # List user's SSH keys
DELETE /api/ssh-keys/:id       # Remove SSH key
```

### SSH Session Endpoints (internal)

```
POST   /api/internal/ssh/auth        # Validate SSH key/token
POST   /api/internal/ssh/session     # Create session for SSH user
GET    /api/internal/ssh/terminals   # List terminals for SSH session
```

---

## 10. Security Considerations

| Concern              | Mitigation                                                    |
| -------------------- | ------------------------------------------------------------- |
| Brute force          | Rate limiting, fail2ban integration, progressive delays       |
| Anonymous abuse      | Guest sessions have limits (e.g., 1 demo terminal, no agents) |
| Key theft            | Keys are fingerprinted; users can revoke; last-used tracking  |
| Session hijacking    | Sessions tied to SSH connection; auto-expire on disconnect    |
| Resource exhaustion  | Connection limits, idle timeouts, memory caps per session     |
| Privilege escalation | SSH session inherits user permissions; no sudo/root           |
| Audit trail          | All connections logged with IP, key fingerprint, actions      |

---

## 11. Deployment

### Docker Compose Addition

```yaml
rexec-ssh:
  build:
    context: .
    dockerfile: Dockerfile.ssh
  ports:
    - "2222:2222"
  volumes:
    - ./ssh_host_key:/etc/rexec/ssh_host_key:ro
  environment:
    - SSH_GATEWAY_API_URL=http://rexec-api:8080
  depends_on:
    - rexec-api
```

### DNS Setup

```
rexec.io.       IN A     <gateway-ip>
ssh.rexec.io.   IN CNAME rexec.io.
```

### Port 22 Access

For `ssh rexec.io` (default port 22):

- Run gateway on 2222, use iptables/nftables redirect
- Or use a reverse proxy (nginx stream, HAProxy)
- Or run as root (not recommended in production)

---

## 12. Example Session Flow

### First-Time User (No Account)

```bash
$ ssh rexec.io
# 1. SSH handshake - client offers ~/.ssh/id_ed25519.pub
# 2. Server ACCEPTS the key (any valid key works)
# 3. Server extracts fingerprint: SHA256:xYz789...
# 4. Fingerprint lookup → not found → new guest session
# 5. Create ssh_fingerprints record
# 6. Initialize TUI with guest dashboard
# 7. User sees: "Welcome! Link your account for full access"
# 8. Guest can explore demo, create 1 trial terminal

# User decides to link account:
# 9. TUI: Settings → Link Account → enters email
# 10. Email verification sent
# 11. User clicks link → fingerprint now linked to account
# 12. Next SSH connection = full access
```

### Returning User (Linked Account)

```bash
$ ssh rexec.io
# 1. SSH handshake - server accepts key
# 2. Fingerprint lookup → found → user_id = abc123
# 3. Load user's session, permissions, preferences
# 4. TUI shows personalized dashboard with all terminals/agents
# 5. User selects a terminal...
# 6. TUI connects to container WebSocket
# 7. PTY bridge established
# 8. User is now in container shell

# User presses Ctrl+] to return to dashboard
# 9. Bridge disconnects, back to TUI

# User presses 'q'
# 10. SSH session ends gracefully
```

### Returning Guest (Unlinked but Recognized)

```bash
$ ssh rexec.io
# 1. Fingerprint lookup → found but user_id = NULL
# 2. Load guest session preferences
# 3. Resume where they left off
# 4. Still prompted to link account for full access
```

---

## 13. Success Metrics

| Metric                     | Target              |
| -------------------------- | ------------------- |
| Time to first dashboard    | < 2 seconds         |
| Terminal connect latency   | < 500ms             |
| Concurrent SSH sessions    | 1000+               |
| Auth success rate          | > 99.9%             |
| Session recording coverage | 100% (when enabled) |

---

## 14. Future Enhancements

- **Mobile SSH**: Test with Termux, Blink, Prompt
- **tmux Integration**: `ssh rexec.io -t tmux attach`
- **Notifications**: Push alerts to SSH session
- **Multiplexing**: Multiple terminals in one SSH session
- **Git over SSH**: `git clone ssh://rexec.io/snippets/my-repo`

---

## 15. Dependencies to Add

```go
// go.mod additions
require (
    github.com/charmbracelet/wish v1.4.0
    github.com/charmbracelet/wish/accesscontrol v0.0.0
    github.com/charmbracelet/wish/bubbletea v0.0.0
    github.com/charmbracelet/wish/logging v0.0.0
    golang.org/x/crypto v0.45.0  // already present
)
```

---

## 16. Quick Start (Development)

```bash
# 1. Generate test host key
ssh-keygen -t ed25519 -f ./dev_host_key -N ""

# 2. Build and run
make ssh-gateway
./bin/rexec-ssh --host-key=./dev_host_key --port=2222

# 3. Test connection
ssh -p 2222 guest@localhost

# 4. Test with your SSH key
ssh -p 2222 localhost
```

---

## References

- [Charm wish library](https://github.com/charmbracelet/wish)
- [terminal.shop source](https://github.com/charmbracelet/terminal-shop) (if public)
- [Bubble Tea documentation](https://github.com/charmbracelet/bubbletea)
- [SSH protocol RFC 4252](https://tools.ietf.org/html/rfc4252)
