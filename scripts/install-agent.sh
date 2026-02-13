#!/bin/bash
# Rexec Agent Installer
# Usage: curl -fsSL https://rexec.sh/install-agent.sh | bash -s -- --token YOUR_TOKEN
#
# This script installs the Rexec agent on your server, allowing it to appear
# as a terminal in your Rexec dashboard.

set -e

# Configuration
REXEC_API="${REXEC_API:-https://rexec.sh}"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/rexec"
SERVICE_DIR="/etc/systemd/system"
REPO="rexec/rexec"
USE_SUDO=""
SERVICE_INSTALL=true

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Parse arguments
TOKEN=""
NAME=""
LABELS=""
AGENT_ID=""
UNINSTALL=false
AGENT_SHELL=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --token|-t)
            TOKEN="$2"
            shift 2
            ;;
        --agent-id|-i)
            AGENT_ID="$2"
            shift 2
            ;;
        --name|-n)
            NAME="$2"
            shift 2
            ;;
        --labels|-l)
            LABELS="$2"
            shift 2
            ;;
        --api)
            REXEC_API="$2"
            shift 2
            ;;
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        --help|-h)
            echo "Rexec Agent Installer"
            echo ""
            echo "Usage: curl -fsSL https://rexec.sh/install-agent.sh | bash -s -- [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --token, -t TOKEN      Agent registration token (required for install)"
            echo "  --agent-id, -i ID      Pre-registered agent ID (optional)"
            echo "  --name, -n NAME        Custom name for this agent (default: hostname)"
            echo "  --labels, -l LABELS    Comma-separated labels (e.g., 'prod,web,us-east')"
            echo "  --api URL              Rexec API URL (default: https://rexec.sh)"
            echo "  --uninstall            Uninstall the agent and remove all files"
            echo "  --help, -h             Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

print_banner() {
    echo -e "${CYAN}${BOLD}"
    echo "██████╗ ███████╗██╗  ██╗███████╗ ██████╗"
    echo "██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝"
    echo "██████╔╝█████╗   ╚███╔╝ █████╗  ██║     "
    echo "██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║     "
    echo "██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗"
    echo "╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝"
    echo -e "${NC}"
    echo -e "${BOLD}Agent Installer - Connect Any Server${NC}"
    echo ""
}

check_privileges() {
    # Running as root - full system install
    if [ "$EUID" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
        CONFIG_DIR="/etc/rexec"
        SERVICE_INSTALL=true
        USE_SUDO=""
        echo -e "${GREEN}Running as root - full system install${NC}"
        return 0
    fi

    # Check if sudo is available and user can use it
    if command -v sudo &> /dev/null; then
        # Try non-interactive sudo check
        if sudo -n true 2>/dev/null; then
            echo -e "${GREEN}Running as non-root with passwordless sudo${NC}"
            INSTALL_DIR="/usr/local/bin"
            CONFIG_DIR="/etc/rexec"
            SERVICE_INSTALL=true
            USE_SUDO="sudo"
            return 0
        fi
        
        # sudo exists but may need password - test if it works
        echo -e "${YELLOW}Checking sudo access (may prompt for password)...${NC}"
        if sudo true 2>/dev/null; then
            echo -e "${GREEN}Running as non-root with sudo access${NC}"
            INSTALL_DIR="/usr/local/bin"
            CONFIG_DIR="/etc/rexec"
            SERVICE_INSTALL=true
            USE_SUDO="sudo"
            return 0
        fi
    fi

    # No root/sudo - offer user-local installation
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}  No root or sudo access detected${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "Options:"
    echo -e "  1. Run with sudo: ${CYAN}sudo bash -s -- --token YOUR_TOKEN${NC}"
    echo -e "  2. Install to user directory (limited functionality)"
    echo ""
    echo -e "${BOLD}Proceeding with user-local installation...${NC}"
    echo ""
    
    INSTALL_DIR="$HOME/.local/bin"
    CONFIG_DIR="$HOME/.config/rexec"
    SERVICE_INSTALL=false
    USE_SUDO=""
    
    # Ensure directories exist
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    
    # Add to PATH hint
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        echo -e "${YELLOW}Note: Add ~/.local/bin to your PATH:${NC}"
        echo -e "  ${CYAN}echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc${NC}"
        echo ""
    fi
    
    return 0
}

check_token() {
    if [ -z "$TOKEN" ]; then
        echo -e "${RED}Error: Registration token is required${NC}"
        echo ""
        echo "Get your token from:"
        echo "  1. Login to https://rexec.sh"
        echo "  2. Go to Settings > Agents"
        echo "  3. Click 'Add Agent' to generate a token"
        echo ""
        echo "Then run:"
        echo "  curl -fsSL https://rexec.sh/install-agent.sh | sudo bash -s -- --token YOUR_TOKEN"
        exit 1
    fi
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            echo "Rexec agent currently supports Linux and macOS only."
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="armv7"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    echo -e "${GREEN}Detected platform: ${PLATFORM}${NC}"
}

detect_init_system() {
    if command -v systemctl &> /dev/null && [ -d /run/systemd/system ]; then
        INIT_SYSTEM="systemd"
    elif command -v launchctl &> /dev/null && [ "$OS" = "darwin" ]; then
        INIT_SYSTEM="launchd"
    elif [ -f /etc/init.d/cron ] || [ -d /etc/init.d ]; then
        INIT_SYSTEM="sysvinit"
    else
        INIT_SYSTEM="none"
    fi
    echo -e "${GREEN}Init system: ${INIT_SYSTEM}${NC}"
}

get_latest_version() {
    echo -e "${CYAN}Fetching latest version...${NC}"
    # Try GitHub releases first
    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        # Try rexec API for version
        VERSION=$(curl -s "${REXEC_API}/api/version" 2>/dev/null | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -z "$VERSION" ]; then
        VERSION="v1.0.0"
    fi
    
    echo -e "${GREEN}Latest version: ${VERSION}${NC}"
}

download_agent() {
    SUFFIX="${PLATFORM}"
    DOWNLOAD_TEMP_DIR=$(mktemp -d)
    AGENT_PATH="${DOWNLOAD_TEMP_DIR}/rexec-agent"

    echo -e "${CYAN}Downloading rexec-agent...${NC}" >&2
    
    # Try GitHub releases first (most reliable for releases)
    AGENT_URL="https://github.com/${REPO}/releases/download/${VERSION}/rexec-agent-${SUFFIX}"
    if curl -fsSL "$AGENT_URL" -o "$AGENT_PATH" 2>/dev/null; then
        # Verify it's a binary, not an HTML error page (check for absence of html tag)
        if ! grep -q '<html' "$AGENT_PATH"; then
            echo -e "${GREEN}Downloaded from GitHub releases${NC}" >&2
            chmod +x "$AGENT_PATH"
            echo "$DOWNLOAD_TEMP_DIR"
            return 0
        fi
    fi
    
    # Try rexec.sh (primary domain)
    AGENT_URL="https://rexec.sh/downloads/rexec-agent-${SUFFIX}"
    if curl -fsSL "$AGENT_URL" -o "$AGENT_PATH" 2>/dev/null; then
        # Verify it's a binary, not an HTML error page (check for absence of html tag)
        if ! grep -q '<html' "$AGENT_PATH"; then
            echo -e "${GREEN}Downloaded from rexec.sh${NC}" >&2
            chmod +x "$AGENT_PATH"
            echo "$DOWNLOAD_TEMP_DIR"
            return 0
        fi
    fi
    
    # Try rexec.pipeops.io (fallback)
    AGENT_URL="https://rexec.pipeops.io/downloads/rexec-agent-${SUFFIX}"
    if curl -fsSL "$AGENT_URL" -o "$AGENT_PATH" 2>/dev/null; then
        # Verify it's a binary, not an HTML error page (check for absence of html tag)
        if ! grep -q '<html' "$AGENT_PATH"; then
            echo -e "${GREEN}Downloaded from rexec.pipeops.io${NC}" >&2
            chmod +x "$AGENT_PATH"
            echo "$DOWNLOAD_TEMP_DIR"
            return 0
        fi
    fi
    
    # If all else fails, provide build instructions
    echo "" >&2
    echo -e "${RED}═══════════════════════════════════════════════════════════${NC}" >&2
    echo -e "${RED}  Agent binary not available for download${NC}" >&2
    echo -e "${RED}═══════════════════════════════════════════════════════════${NC}" >&2
    echo "" >&2
    echo -e "${YELLOW}The agent binary hasn't been released yet for ${PLATFORM}.${NC}" >&2
    echo "" >&2
    echo -e "${BOLD}Option 1: Build from source${NC}" >&2
    echo "" >&2
    echo "  # Install Go 1.21+ if not installed" >&2
    echo "  git clone https://github.com/${REPO}.git" >&2
    echo "  cd rexec" >&2
    echo "  go build -o rexec-agent ./cmd/rexec-agent" >&2
    echo "  sudo mv rexec-agent /usr/local/bin/" >&2
    echo "  sudo chmod +x /usr/local/bin/rexec-agent" >&2
    echo "" >&2
    echo -e "${BOLD}Option 2: Wait for release${NC}" >&2
    echo "  Check: https://github.com/${REPO}/releases" >&2
    echo "" >&2
    rm -rf "$DOWNLOAD_TEMP_DIR"
    exit 1
}

install_agent() {
    TEMP_DIR=$1

    echo -e "${CYAN}Installing agent to ${INSTALL_DIR}...${NC}"
    $USE_SUDO mv "${TEMP_DIR}/rexec-agent" "${INSTALL_DIR}/rexec-agent"
    rm -rf "$TEMP_DIR"

    # Create config directory
    $USE_SUDO mkdir -p "$CONFIG_DIR"
}

detect_shell() {
    # Prefer a fully-featured interactive shell so arrow keys/history work.
    # Use $SHELL only if it’s executable and not plain sh.
    if [ -n "${SHELL:-}" ] && [ -x "${SHELL}" ]; then
        local base
        base="$(basename "${SHELL}")"
        if [ "${base}" != "sh" ]; then
            AGENT_SHELL="${SHELL}"
            return
        fi
    fi

    for candidate in /bin/zsh /usr/bin/zsh /bin/bash /usr/bin/bash /usr/local/bin/bash /bin/sh /usr/bin/sh; do
        if [ -x "${candidate}" ]; then
            AGENT_SHELL="${candidate}"
            return
        fi
    done

    AGENT_SHELL="/bin/sh"
}

register_agent() {
    echo -e "${CYAN}Registering agent with Rexec...${NC}"
    
    # Generate agent name from hostname if not provided
    if [ -z "$NAME" ]; then
        NAME=$(hostname -s 2>/dev/null || hostname)
    fi
    
    # If token is already an API token (starts with rexec_), use it directly
    if [[ "$TOKEN" == rexec_* ]]; then
        echo -e "${GREEN}Using API token for authentication${NC}"
        # If no agent_id, we need to register
        if [ -z "$AGENT_ID" ]; then
            # Register new agent with API token
            REGISTER_RESPONSE=$(curl -s -X POST "${REXEC_API}/api/agents/register" \
                -H "Authorization: Bearer ${TOKEN}" \
                -H "Content-Type: application/json" \
                -d "{\"name\": \"${NAME}\", \"os\": \"$(uname -s | tr '[:upper:]' '[:lower:]')\", \"arch\": \"${ARCH}\", \"shell\": \"${AGENT_SHELL}\", \"hostname\": \"$(hostname)\"}")
            
            # Extract agent ID from response - try jq first, fallback to grep
            if command -v jq &> /dev/null; then
                AGENT_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.id // empty')
                NEW_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.token // empty')
            else
                AGENT_ID=$(echo "$REGISTER_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
                NEW_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"token":"rexec_[^"]*"' | head -1 | cut -d'"' -f4)
            fi
            
            if [ -z "$AGENT_ID" ]; then
                echo -e "${RED}Failed to register agent. Response: ${REGISTER_RESPONSE}${NC}"
                exit 1
            fi
            
            # Check if a new token was returned (use it if so)
            if [ -n "$NEW_TOKEN" ] && [[ "$NEW_TOKEN" == rexec_* ]]; then
                TOKEN="$NEW_TOKEN"
                echo -e "${GREEN}Received new API token for agent${NC}"
            fi
            
            echo -e "${GREEN}Agent registered with ID: ${AGENT_ID}${NC}"
        fi
        return 0
    fi
    
    # Token is a JWT (doesn't start with rexec_)
    # We MUST register to get an API token, otherwise it will expire
    if [ -n "$AGENT_ID" ]; then
        echo -e "${YELLOW}Warning: You provided a JWT token with an existing agent ID.${NC}"
        echo -e "${YELLOW}JWT tokens expire after 24 hours. We recommend using an API token.${NC}"
        echo -e "${YELLOW}Generate one at: ${REXEC_API}/account/api${NC}"
        echo ""
        echo -e "${CYAN}Attempting to register a new agent and get a permanent API token...${NC}"
        # Clear AGENT_ID to force registration
        AGENT_ID=""
    fi
    
    # Register new agent - this returns an API token that doesn't expire
    REGISTER_RESPONSE=$(curl -s -X POST "${REXEC_API}/api/agents/register" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"${NAME}\", \"os\": \"$(uname -s | tr '[:upper:]' '[:lower:]')\", \"arch\": \"${ARCH}\", \"shell\": \"${AGENT_SHELL}\", \"hostname\": \"$(hostname)\"}")
    
    # Debug: show response (mask token if present)
    if [ -n "${DEBUG:-}" ]; then
        echo "DEBUG: Registration response: $(echo "$REGISTER_RESPONSE" | sed 's/rexec_[a-f0-9]*/rexec_***MASKED***/g')"
    fi
    
    # Check for error in response
    if echo "$REGISTER_RESPONSE" | grep -q '"error"'; then
        ERROR_MSG=$(echo "$REGISTER_RESPONSE" | grep -o '"error":"[^"]*"' | cut -d'"' -f4)
        echo -e "${RED}Registration failed: ${ERROR_MSG}${NC}"
        echo -e "${RED}Make sure your token is valid and not expired.${NC}"
        exit 1
    fi
    
    # Extract agent ID from response - try jq first, fallback to grep
    if command -v jq &> /dev/null; then
        AGENT_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.id // empty')
        NEW_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.token // empty')
    else
        AGENT_ID=$(echo "$REGISTER_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        NEW_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"token":"rexec_[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    
    if [ -z "$AGENT_ID" ]; then
        echo -e "${RED}Failed to register agent. Response: ${REGISTER_RESPONSE}${NC}"
        echo -e "${RED}Make sure your token is valid and not expired.${NC}"
        exit 1
    fi
    
    if [ -n "$NEW_TOKEN" ] && [[ "$NEW_TOKEN" == rexec_* ]]; then
        TOKEN="$NEW_TOKEN"
        echo -e "${GREEN}✓ Received permanent API token (never expires)${NC}"
    else
        echo -e "${YELLOW}Warning: No API token returned. Using original token (may expire).${NC}"
        echo -e "${YELLOW}Generate an API token at: ${REXEC_API}/account/api${NC}"
    fi
    
    echo -e "${GREEN}✓ Agent registered: ${AGENT_ID}${NC}"
}

create_config() {
    echo -e "${CYAN}Creating configuration...${NC}"

    # Generate agent name from hostname if not provided
    if [ -z "$NAME" ]; then
        NAME=$(hostname -s 2>/dev/null || hostname)
    fi

    # Get system info
    SYSTEM_INFO=$(uname -a)
    IP_ADDR=$(hostname -I 2>/dev/null | awk '{print $1}' || curl -s ifconfig.me 2>/dev/null || echo "unknown")

    # Determine working directory and log file based on install type
    if [ "$SERVICE_INSTALL" = true ]; then
        WORKING_DIR="/root"
        LOG_FILE="/var/log/rexec-agent.log"
    else
        WORKING_DIR="$HOME"
        LOG_FILE="$HOME/.config/rexec/agent.log"
    fi

    # Create config file (use sudo if needed for system installs)
    if [ -n "$USE_SUDO" ]; then
        $USE_SUDO tee "${CONFIG_DIR}/agent.yaml" > /dev/null << EOF
# Rexec Agent Configuration
# Generated: $(date -Iseconds)

# API endpoint
api_url: ${REXEC_API}

# Agent identification
token: ${TOKEN}
agent_id: ${AGENT_ID}
name: ${NAME}
labels:
$(echo "$LABELS" | tr ',' '\n' | sed 's/^/  - /' | grep -v '^  - $')

# System information (auto-detected)
system:
  platform: ${PLATFORM}
  hostname: $(hostname)
  ip: ${IP_ADDR}

# Connection settings
reconnect_interval: 5s
heartbeat_interval: 30s

# Self-update (opt-in)
# auto_update: false

# Shell configuration
shell: ${AGENT_SHELL}
working_dir: ${WORKING_DIR}

# Logging
log_level: info
log_file: ${LOG_FILE}
EOF
        $USE_SUDO chmod 600 "${CONFIG_DIR}/agent.yaml"
    else
        cat > "${CONFIG_DIR}/agent.yaml" << EOF
# Rexec Agent Configuration
# Generated: $(date -Iseconds)

# API endpoint
api_url: ${REXEC_API}

# Agent identification
token: ${TOKEN}
agent_id: ${AGENT_ID}
name: ${NAME}
labels:
$(echo "$LABELS" | tr ',' '\n' | sed 's/^/  - /' | grep -v '^  - $')

# System information (auto-detected)
system:
  platform: ${PLATFORM}
  hostname: $(hostname)
  ip: ${IP_ADDR}

# Connection settings
reconnect_interval: 5s
heartbeat_interval: 30s

# Self-update (opt-in)
# auto_update: false

# Shell configuration
shell: ${AGENT_SHELL}
working_dir: ${WORKING_DIR}

# Logging
log_level: info
log_file: ${LOG_FILE}
EOF
        chmod 600 "${CONFIG_DIR}/agent.yaml"
    fi
    
    echo -e "${GREEN}Configuration saved to ${CONFIG_DIR}/agent.yaml${NC}"
}

setup_systemd() {
    echo -e "${CYAN}Setting up systemd service...${NC}"

    $USE_SUDO tee "${SERVICE_DIR}/rexec-agent.service" > /dev/null << EOF
[Unit]
Description=Rexec Agent - Cloud Terminal Connection
Documentation=https://rexec.sh/agents
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/rexec-agent --config ${CONFIG_DIR}/agent.yaml start
Restart=always
RestartSec=10
User=root
StandardOutput=journal
StandardError=journal
SyslogIdentifier=rexec-agent

# Security hardening
NoNewPrivileges=false
ProtectSystem=false
ProtectHome=false

# Environment
Environment=REXEC_API=${REXEC_API}
Environment=REXEC_TOKEN=${TOKEN}
Environment=REXEC_CONFIG=${CONFIG_DIR}/agent.yaml

# Note: No watchdog - agent has its own health checks via WebSocket ping/pong

[Install]
WantedBy=multi-user.target
EOF

    $USE_SUDO systemctl daemon-reload
    $USE_SUDO systemctl enable rexec-agent
    # Restart if already running so updated token/config take effect
    if $USE_SUDO systemctl is-active --quiet rexec-agent; then
        $USE_SUDO systemctl restart rexec-agent
    else
        $USE_SUDO systemctl start rexec-agent
    fi

    echo -e "${GREEN}Systemd service installed and started${NC}"
}

setup_launchd() {
    echo -e "${CYAN}Setting up launchd service...${NC}"

    PLIST_PATH="/Library/LaunchDaemons/io.pipeops.rexec-agent.plist"

    $USE_SUDO tee "$PLIST_PATH" > /dev/null << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.pipeops.rexec-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/rexec-agent</string>
        <string>--config</string>
        <string>${CONFIG_DIR}/agent.yaml</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
        <key>NetworkState</key>
        <true/>
    </dict>
    <key>ThrottleInterval</key>
    <integer>10</integer>
    <key>StandardOutPath</key>
    <string>/var/log/rexec-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/rexec-agent.error.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>REXEC_API</key>
        <string>${REXEC_API}</string>
        <key>REXEC_TOKEN</key>
        <string>${TOKEN}</string>
    </dict>
</dict>
</plist>
EOF

    # Reload if already installed so updated token/config take effect
    $USE_SUDO launchctl stop io.pipeops.rexec-agent 2>/dev/null || true
    $USE_SUDO launchctl unload "$PLIST_PATH" 2>/dev/null || true
    $USE_SUDO launchctl load "$PLIST_PATH"
    $USE_SUDO launchctl start io.pipeops.rexec-agent

    echo -e "${GREEN}Launchd service installed and started${NC}"
}

setup_sysvinit() {
    echo -e "${CYAN}Setting up init.d service...${NC}"

    $USE_SUDO tee "/etc/init.d/rexec-agent" > /dev/null << 'INITSCRIPT'
#!/bin/bash
### BEGIN INIT INFO
# Provides:          rexec-agent
# Required-Start:    $network $remote_fs
# Required-Stop:     $network $remote_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Rexec Agent
# Description:       Rexec Agent - Cloud Terminal Connection
### END INIT INFO

DAEMON=/usr/local/bin/rexec-agent
DAEMON_ARGS="--config /etc/rexec/agent.yaml start"
PIDFILE=/var/run/rexec-agent.pid
LOGFILE=/var/log/rexec-agent.log

case "$1" in
    start)
        echo "Starting rexec-agent..."
        start-stop-daemon --start --background --make-pidfile --pidfile $PIDFILE \
            --exec $DAEMON -- $DAEMON_ARGS >> $LOGFILE 2>&1
        ;;
    stop)
        echo "Stopping rexec-agent..."
        start-stop-daemon --stop --pidfile $PIDFILE
        rm -f $PIDFILE
        ;;
    restart)
        $0 stop
        sleep 1
        $0 start
        ;;
    status)
        if [ -f $PIDFILE ] && kill -0 $(cat $PIDFILE) 2>/dev/null; then
            echo "rexec-agent is running"
        else
            echo "rexec-agent is not running"
            exit 1
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac
exit 0
INITSCRIPT

    $USE_SUDO chmod +x /etc/init.d/rexec-agent
    $USE_SUDO update-rc.d rexec-agent defaults 2>/dev/null || $USE_SUDO chkconfig --add rexec-agent 2>/dev/null || true
    if $USE_SUDO /etc/init.d/rexec-agent status >/dev/null 2>&1; then
        $USE_SUDO /etc/init.d/rexec-agent restart
    else
        $USE_SUDO /etc/init.d/rexec-agent start
    fi

    echo -e "${GREEN}Init.d service installed and started${NC}"
}

setup_service() {
    # Skip service setup for user-local installs
    if [ "$SERVICE_INSTALL" = false ]; then
        echo -e "${YELLOW}Skipping service setup (user-local installation)${NC}"
        echo ""
        echo -e "To run the agent manually:"
        echo -e "  ${CYAN}rexec-agent --config ${CONFIG_DIR}/agent.yaml start${NC}"
        echo ""
        echo -e "To run in background:"
        echo -e "  ${CYAN}nohup rexec-agent --config ${CONFIG_DIR}/agent.yaml start > ${CONFIG_DIR}/agent.log 2>&1 &${NC}"
        return
    fi

    case "$INIT_SYSTEM" in
        systemd)
            setup_systemd
            ;;
        launchd)
            setup_launchd
            ;;
        sysvinit)
            setup_sysvinit
            ;;
        *)
            echo -e "${YELLOW}No supported init system found. Agent installed but not running as service.${NC}"
            echo "To run manually: rexec-agent --config ${CONFIG_DIR}/agent.yaml start"
            ;;
    esac
}

verify_installation() {
    echo ""
    echo -e "${CYAN}Verifying installation...${NC}"
    
    sleep 2

    # For user-local installs, check in ~/.local/bin
    if [ -f "${INSTALL_DIR}/rexec-agent" ]; then
        echo -e "${GREEN}${BOLD}✓ Agent binary installed${NC}"
    elif command -v rexec-agent &> /dev/null; then
        echo -e "${GREEN}${BOLD}✓ Agent binary installed${NC}"
    else
        echo -e "${RED}✗ Agent binary not found in PATH${NC}"
    fi

    if [ -f "${CONFIG_DIR}/agent.yaml" ]; then
        echo -e "${GREEN}${BOLD}✓ Configuration created${NC}"
    else
        echo -e "${RED}✗ Configuration not found${NC}"
    fi

    # Skip service check for user-local installs
    if [ "$SERVICE_INSTALL" = false ]; then
        echo -e "${YELLOW}! Service setup skipped (user-local install)${NC}"
        return
    fi

    # Check service status
    case "$INIT_SYSTEM" in
        systemd)
            if $USE_SUDO systemctl is-active --quiet rexec-agent; then
                echo -e "${GREEN}${BOLD}✓ Service running${NC}"
            else
                echo -e "${RED}✗ Service not running${NC}"
                echo "Check logs: journalctl -u rexec-agent -f"
            fi
            ;;
        launchd)
            if $USE_SUDO launchctl list | grep -q io.pipeops.rexec-agent; then
                echo -e "${GREEN}${BOLD}✓ Service running${NC}"
            else
                echo -e "${RED}✗ Service not running${NC}"
                echo "Check logs: tail -f /var/log/rexec-agent.log"
            fi
            ;;
        *)
            if pgrep -x rexec-agent > /dev/null; then
                echo -e "${GREEN}${BOLD}✓ Agent process running${NC}"
            else
                echo -e "${YELLOW}! Agent process not detected${NC}"
            fi
            ;;
    esac
}

uninstall_agent() {
    echo -e "${CYAN}${BOLD}Uninstalling Rexec Agent...${NC}"
    echo ""
    
    # Stop and disable services
    case "$INIT_SYSTEM" in
        systemd)
            echo -e "${CYAN}Stopping systemd service...${NC}"
            systemctl stop rexec-agent 2>/dev/null || true
            systemctl disable rexec-agent 2>/dev/null || true
            rm -f "${SERVICE_DIR}/rexec-agent.service"
            systemctl daemon-reload
            echo -e "${GREEN}✓ Systemd service removed${NC}"
            ;;
        launchd)
            echo -e "${CYAN}Stopping launchd service...${NC}"
            launchctl stop io.pipeops.rexec-agent 2>/dev/null || true
            launchctl unload /Library/LaunchDaemons/io.pipeops.rexec-agent.plist 2>/dev/null || true
            rm -f /Library/LaunchDaemons/io.pipeops.rexec-agent.plist
            echo -e "${GREEN}✓ Launchd service removed${NC}"
            ;;
        sysvinit)
            echo -e "${CYAN}Stopping init.d service...${NC}"
            /etc/init.d/rexec-agent stop 2>/dev/null || true
            update-rc.d -f rexec-agent remove 2>/dev/null || chkconfig --del rexec-agent 2>/dev/null || true
            rm -f /etc/init.d/rexec-agent
            echo -e "${GREEN}✓ Init.d service removed${NC}"
            ;;
        *)
            # Kill any running agent process
            pkill -f rexec-agent 2>/dev/null || true
            echo -e "${GREEN}✓ Agent process stopped${NC}"
            ;;
    esac
    
    # Remove binary
    if [ -f "${INSTALL_DIR}/rexec-agent" ]; then
        rm -f "${INSTALL_DIR}/rexec-agent"
        echo -e "${GREEN}✓ Agent binary removed${NC}"
    fi
    
    # Remove config directory
    if [ -d "$CONFIG_DIR" ]; then
        rm -rf "$CONFIG_DIR"
        echo -e "${GREEN}✓ Configuration removed${NC}"
    fi
    
    # Remove log files
    rm -f /var/log/rexec-agent.log 2>/dev/null || true
    rm -f /var/log/rexec-agent.error.log 2>/dev/null || true
    
    echo ""
    echo -e "${GREEN}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}${BOLD}  Rexec Agent Uninstalled Successfully!${NC}"
    echo -e "${GREEN}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "The agent has been removed from this server."
    echo -e "To reinstall, run:"
    echo -e "  ${CYAN}curl -fsSL https://rexec.sh/install-agent.sh | sudo bash -s -- --token YOUR_TOKEN${NC}"
    echo ""
}

show_next_steps() {
    echo ""
    echo -e "${GREEN}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}${BOLD}  Installation Complete!${NC}"
    echo -e "${GREEN}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BOLD}Your server '${NAME}' should now appear in your Rexec dashboard.${NC}"
    echo ""
    
    # User-local install instructions
    if [ "$SERVICE_INSTALL" = false ]; then
        echo -e "${YELLOW}Note: User-local installation - no system service created${NC}"
        echo ""
        echo -e "${BOLD}To start the agent:${NC}"
        echo -e "  ${CYAN}${INSTALL_DIR}/rexec-agent --config ${CONFIG_DIR}/agent.yaml start${NC}"
        echo ""
        echo -e "${BOLD}To run in background:${NC}"
        echo -e "  ${CYAN}nohup ${INSTALL_DIR}/rexec-agent --config ${CONFIG_DIR}/agent.yaml start > ${CONFIG_DIR}/agent.log 2>&1 &${NC}"
        echo ""
        echo -e "${BOLD}View logs:${NC}"
        echo -e "  ${CYAN}tail -f ${CONFIG_DIR}/agent.log${NC}"
        echo ""
        echo -e "${BOLD}Dashboard:${NC} ${CYAN}https://rexec.sh${NC}"
        echo -e "${BOLD}Documentation:${NC} ${CYAN}https://rexec.sh/agents${NC}"
        echo ""
        return
    fi

    echo -e "${BOLD}Useful commands:${NC}"
    echo ""
    case "$INIT_SYSTEM" in
        systemd)
            echo -e "  View logs:        ${CYAN}journalctl -u rexec-agent -f${NC}"
            echo -e "  Check status:     ${CYAN}systemctl status rexec-agent${NC}"
            echo -e "  Restart agent:    ${CYAN}systemctl restart rexec-agent${NC}"
            echo -e "  Stop agent:       ${CYAN}systemctl stop rexec-agent${NC}"
            ;;
        launchd)
            echo -e "  View logs:        ${CYAN}tail -f /var/log/rexec-agent.log${NC}"
            echo -e "  Restart agent:    ${CYAN}launchctl stop io.pipeops.rexec-agent && launchctl start io.pipeops.rexec-agent${NC}"
            ;;
        *)
            echo -e "  View logs:        ${CYAN}tail -f /var/log/rexec-agent.log${NC}"
            echo -e "  Restart agent:    ${CYAN}/etc/init.d/rexec-agent restart${NC}"
            ;;
    esac
    echo ""
    echo -e "${BOLD}To uninstall:${NC}"
    echo -e "  ${CYAN}curl -fsSL https://rexec.sh/install-agent.sh | sudo bash -s -- --uninstall${NC}"
    echo ""
    echo -e "${BOLD}Dashboard:${NC} ${CYAN}https://rexec.sh${NC}"
    echo -e "${BOLD}Documentation:${NC} ${CYAN}https://rexec.sh/agents${NC}"
    echo ""
}

main() {
    print_banner
    check_privileges
    # Detect platform first so init detection can rely on $OS.
    detect_platform
    detect_shell
    detect_init_system
    
    # Handle uninstall
    if [ "$UNINSTALL" = true ]; then
        uninstall_agent
        exit 0
    fi
    
    check_token
    get_latest_version
    TEMP_DIR=$(download_agent)
    install_agent "$TEMP_DIR"
    register_agent
    create_config
    setup_service
    verify_installation
    show_next_steps
}

main "$@"
