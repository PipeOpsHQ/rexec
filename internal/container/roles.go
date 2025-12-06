package container

import (
	"fmt"
)

// RoleInfo represents a user role and its configuration
type RoleInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Packages    []string `json:"packages"` // Generic package names
}

// AvailableRoles returns the list of supported roles
func AvailableRoles() []RoleInfo {
	return []RoleInfo{
		{
			ID:          "standard",
			Name:        "The Minimalist",
			Description: "I use Arch btw. Just give me a shell.",
			Icon:        "🧘",
			Packages:    []string{"zsh", "git", "curl", "wget", "vim", "nano", "htop", "jq", "neofetch", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
		},
		{
			ID:          "node",
			Name:        "10x JS Ninja",
			Description: "Ship fast, break things, npm install everything.",
			Icon:        "🚀",
			Packages:    []string{"zsh", "git", "nodejs", "npm", "yarn", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
		},
		{
			ID:          "python",
			Name:        "Data Wizard",
			Description: "Import antigravity. I speak in list comprehensions.",
			Icon:        "🧙‍♂️",
			Packages:    []string{"zsh", "git", "python3", "python3-pip", "python3-venv", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
		},
		{
			ID:          "go",
			Name:        "The Gopher",
			Description: "If err != nil { panic(err) }. Simplicity is key.",
			Icon:        "🐹",
			Packages:    []string{"zsh", "git", "make", "go", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
		},
		{
			ID:          "neovim",
			Name:        "Neovim God",
			Description: "My config is longer than your code. Mouse? What mouse?",
			Icon:        "⌨️",
			Packages:    []string{"zsh", "git", "neovim", "ripgrep", "gcc", "make", "curl", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
		},
		{
			ID:          "devops",
			Name:        "YAML Herder",
			Description: "I don't write code, I write config. Prod is my playground.",
			Icon:        "☸️",
			Packages:    []string{"zsh", "git", "docker-cli", "kubectl", "ansible", "terraform", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
		},
		{
			ID:          "overemployed",
			Name:        "Vibe Coder",
			Description: "AI-powered coding with Claude, Aider, OpenCode & more CLI tools.",
			Icon:        "🤖",
			Packages:    []string{"zsh", "git", "tmux", "python3", "python3-pip", "python3-venv", "nodejs", "npm", "curl", "wget", "htop", "vim", "neovim", "ripgrep", "fzf", "jq", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
		},
	}
}

// GenerateRoleScript generates the installation script for a specific role
func GenerateRoleScript(roleID string) (string, error) {
	var role *RoleInfo
	for _, r := range AvailableRoles() {
		if r.ID == roleID {
			role = &r
			break
		}
	}

	if role == nil {
		return "", fmt.Errorf("role not found: %s", roleID)
	}

	// Build the script reusing the package manager detection from shell_setup.go
	// We'll inject the specific packages for this role
	packages := ""
	for _, p := range role.Packages {
		packages += p + " "
	}

	script := fmt.Sprintf(`#!/bin/sh
set -e

echo "Installing tools for role: %s..."

# Wait for any existing package manager locks (max 60 seconds)
wait_for_locks() {
    local max_wait=60
    local waited=0
    
    # List of known lock files
    local locks="/var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock /lib/apk/db/lock /var/run/dnf.pid /var/run/yum.pid /var/lib/pacman/db.lck"
    
    # Helper to check if any lock exists
    check_locks() {
        for lock in $locks; do
            if [ -f "$lock" ]; then
                # If fuser exists, check if process is actually running
                if command -v fuser >/dev/null 2>&1; then
                    if fuser "$lock" >/dev/null 2>&1; then
                        return 0 # Locked
                    fi
                else
                    return 0 # File exists, assume locked (safer fallback)
                fi
            fi
        done
        return 1 # Not locked
    }

    while check_locks; do
        if [ $waited -ge $max_wait ]; then
            echo "Timeout waiting for package manager lock"
            # Try to force remove stale locks if we timed out
            echo "Attempting to clear stale locks..."
            for lock in $locks; do
                if [ -f "$lock" ]; then
                    rm -f "$lock" 2>/dev/null || true
                fi
            done
            return 0
        fi
        echo "Waiting for package manager lock release..."
        sleep 2
        waited=$((waited + 2))
    done
    return 0
}

# Function to install packages based on detected manager
install_role_packages() {
    GENERIC_PACKAGES="%s"
    PACKAGES="$GENERIC_PACKAGES"
    
    # Wait for locks before starting
    wait_for_locks || true

    if command -v apt-get >/dev/null 2>&1; then
        export DEBIAN_FRONTEND=noninteractive
        
        # Apt options for robustness
        APT_OPTS="-o DPkg::Lock::Timeout=60 -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold"
        
        # Enable universe repository for Ubuntu (needed for neovim, ripgrep, etc.)
        if grep -q "Ubuntu" /etc/issue 2>/dev/null || grep -q "Ubuntu" /etc/os-release 2>/dev/null; then
            apt-get $APT_OPTS update -qq
            apt-get $APT_OPTS install -y -qq software-properties-common >/dev/null 2>&1 || true
            add-apt-repository -y universe >/dev/null 2>&1 || true
        fi
        
        # Update package lists
        flock -w 120 /var/lib/dpkg/lock-frontend apt-get $APT_OPTS update -qq || true

        # Try bulk install first
        if ! flock -w 120 /var/lib/dpkg/lock-frontend apt-get $APT_OPTS install -y -qq $PACKAGES >/dev/null 2>&1; then
            echo "Bulk install failed, trying individual packages..."
            for pkg in $PACKAGES; do
                echo "Installing $pkg..."
                apt-get $APT_OPTS install -y -qq "$pkg" >/dev/null 2>&1 || echo "Warning: Failed to install $pkg"
            done
        fi
    elif command -v apk >/dev/null 2>&1; then
        # Alpine mapping
        PACKAGES=""
        for pkg in $GENERIC_PACKAGES; do
            case "$pkg" in
                python3-pip) PACKAGES="$PACKAGES py3-pip" ;;
                python3-venv) ;; # Included in python3 or not needed
                zsh-autosuggestions) PACKAGES="$PACKAGES zsh-autosuggestions" ;;
                zsh-syntax-highlighting) PACKAGES="$PACKAGES zsh-syntax-highlighting" ;;
                *) PACKAGES="$PACKAGES $pkg" ;;
            esac
        done

        apk update >/dev/null 2>&1
        if ! apk add --no-cache $PACKAGES >/dev/null 2>&1; then
            echo "Bulk install failed, trying individual packages..."
            for pkg in $PACKAGES; do
                apk add --no-cache "$pkg" >/dev/null 2>&1 || echo "Warning: Failed to install $pkg"
            done
        fi
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y -q $PACKAGES >/dev/null 2>&1 || {
            for pkg in $PACKAGES; do dnf install -y -q "$pkg" >/dev/null 2>&1 || true; done
        }
    elif command -v yum >/dev/null 2>&1; then
        yum install -y -q $PACKAGES >/dev/null 2>&1 || {
            for pkg in $PACKAGES; do yum install -y -q "$pkg" >/dev/null 2>&1 || true; done
        }
    elif command -v pacman >/dev/null 2>&1; then
        pacman -Sy --noconfirm $PACKAGES >/dev/null 2>&1 || {
            for pkg in $PACKAGES; do pacman -S --noconfirm "$pkg" >/dev/null 2>&1 || true; done
        }
    else
        echo "Unsupported package manager"
        # Don't exit, try to continue setup
    fi
}

install_role_packages

# Configure Zsh if installed
if command -v zsh >/dev/null 2>&1; then
    echo "Configuring zsh..."

    # Change default shell
    if [ -f /etc/passwd ]; then
        ZSH_PATH=$(which zsh)
        sed -i "s|root:.*:/bin/.*|root:x:0:0:root:/root:$ZSH_PATH|" /etc/passwd 2>/dev/null || true
    fi

    # Create minimal .zshrc
    cat > /root/.zshrc << 'ZSHRC'
export TERM=xterm-256color
export LANG=en_US.UTF-8

# Basic Config
autoload -Uz compinit && compinit
setopt HIST_IGNORE_ALL_DUPS SHARE_HISTORY
HISTFILE=~/.zsh_history
HISTSIZE=10000
SAVEHIST=10000

# Autosuggestions (detect path) - Disabled for input stability
# if [ -f /usr/share/zsh-autosuggestions/zsh-autosuggestions.zsh ]; then
#     source /usr/share/zsh-autosuggestions/zsh-autosuggestions.zsh
# elif [ -f /usr/share/zsh/plugins/zsh-autosuggestions/zsh-autosuggestions.zsh ]; then
#     source /usr/share/zsh/plugins/zsh-autosuggestions/zsh-autosuggestions.zsh
# fi

# Syntax Highlighting (detect path) - Disabled for input stability
# if [ -f /usr/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh ]; then
#     source /usr/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh
# elif [ -f /usr/share/zsh/plugins/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh ]; then
#     source /usr/share/zsh/plugins/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh
# fi

# Bind keys for autosuggestions
# bindkey '^ ' autosuggest-accept
# bindkey '^[[C' autosuggest-accept # Right arrow

# Prompt - commented out to fix xterm parsing errors
# PROMPT='%%F{cyan}%%n%%f@%%F{blue}%%m%%f %%F{yellow}%%~%%f (?:%%F{green}➜:%%F{red}➜) %%f'

# Aliases
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias gs='git status'
ZSHRC
fi

# Create rexec CLI command with subcommands
create_rexec_cli() {
    mkdir -p /root/.local/bin /home/user/.local/bin 2>/dev/null || true
    
    cat > /root/.local/bin/rexec << 'REXECCLI'
#!/bin/sh

# Rexec CLI - Terminal helper commands
VERSION="1.0.0"

show_help() {
    echo ""
    echo "\033[1;36mRexec CLI v${VERSION}\033[0m"
    echo ""
    echo "Usage: rexec <command>"
    echo ""
    echo "Commands:"
    echo "  tools     Show installed tools and their status"
    echo "  info      Show container information"
    echo "  help      Show this help message"
    echo ""
}

show_tools() {
    echo ""
    echo "\033[1;36m=== Rexec Terminal - Installed Tools ===\033[0m"
    echo ""

    ROLE_FILE="/etc/rexec/role"
    if [ -f "$ROLE_FILE" ]; then
        ROLE=$(cat "$ROLE_FILE")
        echo "\033[1;33mRole:\033[0m $ROLE"
        echo ""
    fi

    echo "\033[1;33mSystem Tools:\033[0m"
    for cmd in zsh git curl wget vim nano htop jq tmux fzf ripgrep neofetch; do
        if command -v $cmd >/dev/null 2>&1; then
            echo "  \033[32m✓\033[0m $cmd"
        fi
    done

    echo ""
    echo "\033[1;33mDevelopment:\033[0m"
    for cmd in python3 pip3 node npm yarn go rustc cargo make gcc; do
        if command -v $cmd >/dev/null 2>&1; then
            VERSION=$($cmd --version 2>/dev/null | head -1 | cut -d' ' -f2 | cut -d'v' -f2 | head -c 10)
            echo "  \033[32m✓\033[0m $cmd ${VERSION:+($VERSION)}"
        fi
    done

    echo ""
    echo "\033[1;33mAI Coding Tools:\033[0m"
    AI_FOUND=0
    for cmd in aider opencode llm interpreter claude; do
        if command -v $cmd >/dev/null 2>&1; then
            echo "  \033[32m✓\033[0m $cmd"
            AI_FOUND=1
        fi
    done
    
    # Check pip-installed tools that might not be in PATH
    if command -v pip3 >/dev/null 2>&1; then
        for pkg in aider-chat llm open-interpreter; do
            if pip3 show $pkg >/dev/null 2>&1; then
                # Only show if binary not found above
                BIN_NAME=$(echo $pkg | cut -d'-' -f1)
                if ! command -v $BIN_NAME >/dev/null 2>&1; then
                    echo "  \033[32m✓\033[0m $pkg (pip installed)"
                    AI_FOUND=1
                fi
            fi
        done
    fi
    
    if [ "$AI_FOUND" = "0" ]; then
        echo "  \033[90m(none installed)\033[0m"
    fi

    echo ""
    echo "\033[1;33mEditors:\033[0m"
    for cmd in vim nvim nano emacs code; do
        if command -v $cmd >/dev/null 2>&1; then
            echo "  \033[32m✓\033[0m $cmd"
        fi
    done

    echo ""
    echo "\033[1;33mDevOps:\033[0m"
    DEVOPS_FOUND=0
    for cmd in docker kubectl terraform ansible helm; do
        if command -v $cmd >/dev/null 2>&1; then
            echo "  \033[32m✓\033[0m $cmd"
            DEVOPS_FOUND=1
        fi
    done
    if [ "$DEVOPS_FOUND" = "0" ]; then
        echo "  \033[90m(none installed)\033[0m"
    fi

    echo ""
    echo "\033[38;5;243mRun 'rexec tools' anytime to see this list\033[0m"
    echo ""
}

show_info() {
    echo ""
    echo "\033[1;36m=== Container Information ===\033[0m"
    echo ""
    
    # OS info
    if [ -f /etc/os-release ]; then
        OS_NAME=$(grep -E "^PRETTY_NAME=" /etc/os-release 2>/dev/null | cut -d'"' -f2)
        echo "\033[1;33mOS:\033[0m $OS_NAME"
    fi
    
    # Hostname
    echo "\033[1;33mHostname:\033[0m $(hostname 2>/dev/null || echo 'unknown')"
    
    # Role
    if [ -f /etc/rexec/role ]; then
        echo "\033[1;33mRole:\033[0m $(cat /etc/rexec/role)"
    fi
    
    # Uptime
    if [ -f /proc/uptime ]; then
        UPTIME_SEC=$(cut -d. -f1 /proc/uptime)
        UPTIME_DAYS=$((UPTIME_SEC / 86400))
        UPTIME_HRS=$(((UPTIME_SEC % 86400) / 3600))
        UPTIME_MIN=$(((UPTIME_SEC % 3600) / 60))
        echo "\033[1;33mUptime:\033[0m ${UPTIME_DAYS}d ${UPTIME_HRS}h ${UPTIME_MIN}m"
    fi
    
    # Memory
    if [ -f /sys/fs/cgroup/memory.max ]; then
        MEM_LIMIT=$(cat /sys/fs/cgroup/memory.max 2>/dev/null)
        MEM_USED=$(cat /sys/fs/cgroup/memory.current 2>/dev/null || echo "0")
        if [ "$MEM_LIMIT" != "max" ] && [ "$MEM_LIMIT" -gt 0 ] 2>/dev/null; then
            MEM_LIMIT_MB=$((MEM_LIMIT / 1024 / 1024))
            MEM_USED_MB=$((MEM_USED / 1024 / 1024))
            echo "\033[1;33mMemory:\033[0m ${MEM_USED_MB}MB / ${MEM_LIMIT_MB}MB"
        fi
    fi
    
    echo ""
}

# Main command dispatch
case "$1" in
    tools)
        show_tools
        ;;
    info)
        show_info
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
REXECCLI

    chmod +x /root/.local/bin/rexec
    cp /root/.local/bin/rexec /home/user/.local/bin/rexec 2>/dev/null || true
    chmod +x /home/user/.local/bin/rexec 2>/dev/null || true
}

# Setup PATH for all roles - ensures installed tools are found
setup_path() {
    for rcfile in /root/.zshrc /root/.bashrc /root/.profile /home/user/.zshrc /home/user/.bashrc /home/user/.profile; do
        if [ -f "$rcfile" ]; then
            if ! grep -q '.local/bin' "$rcfile" 2>/dev/null; then
                echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$rcfile"
            fi
        fi
    done
    # Also set for current session
    export PATH="$HOME/.local/bin:/root/.local/bin:$PATH"
}

# Save role info
save_role_info() {
    mkdir -p /etc/rexec
    echo "%s" > /etc/rexec/role
}

create_rexec_cli
setup_path
save_role_info
echo "Role setup complete!"

# Special handling for Vibe Coder role - install AI CLI tools
if [ "%s" = "Vibe Coder" ]; then
    echo "Installing AI CLI tools..."
    
    # Ensure pip is available and upgrade it
    if command -v pip3 >/dev/null 2>&1; then
        PIP="pip3"
    elif command -v pip >/dev/null 2>&1; then
        PIP="pip"
    else
        echo "pip not found, attempting to install..."
        if command -v apt-get >/dev/null 2>&1; then
            apt-get install -y -qq python3-pip >/dev/null 2>&1 || true
        fi
        if command -v pip3 >/dev/null 2>&1; then
            PIP="pip3"
        else
            PIP=""
        fi
    fi

    if [ -n "$PIP" ]; then
        # Upgrade pip first
        $PIP install --quiet --break-system-packages --upgrade pip 2>/dev/null || $PIP install --quiet --upgrade pip 2>/dev/null || true
        
        # Install aider - the main AI pair programming tool
        echo "Installing aider (AI pair programming)..."
        $PIP install --quiet --break-system-packages aider-chat 2>/dev/null || $PIP install --quiet aider-chat 2>/dev/null || echo "  Warning: aider install failed"

        # Install llm - versatile CLI for LLMs
        echo "Installing llm (CLI for LLMs)..."
        $PIP install --quiet --break-system-packages llm 2>/dev/null || $PIP install --quiet llm 2>/dev/null || echo "  Warning: llm install failed"

        # Install Open Interpreter
        echo "Installing Open Interpreter..."
        $PIP install --quiet --break-system-packages open-interpreter 2>/dev/null || $PIP install --quiet open-interpreter 2>/dev/null || echo "  Warning: interpreter install failed"
    fi

    # Install opencode (sst/opencode) - binary release
    echo "Installing opencode (AI coding assistant)..."
    install_opencode() {
        export HOME="${HOME:-/root}"
        
        if command -v opencode >/dev/null 2>&1; then
            echo "  opencode already installed"
            return 0
        fi
        
        ARCH=$(uname -m)
        case "$ARCH" in
            x86_64|amd64)
                if ldd /bin/ls 2>/dev/null | grep -q musl; then
                    OPENCODE_ARCH="linux-x64-musl"
                else
                    OPENCODE_ARCH="linux-x64"
                fi
                ;;
            aarch64|arm64)
                if ldd /bin/ls 2>/dev/null | grep -q musl; then
                    OPENCODE_ARCH="linux-arm64-musl"
                else
                    OPENCODE_ARCH="linux-arm64"
                fi
                ;;
            *)
                echo "  Unsupported architecture: $ARCH"
                return 1
                ;;
        esac
        
        OPENCODE_VERSION=$(curl -s https://api.github.com/repos/sst/opencode/releases/latest 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
        [ -z "$OPENCODE_VERSION" ] && OPENCODE_VERSION="v1.0.133"
        
        OPENCODE_URL="https://github.com/sst/opencode/releases/download/${OPENCODE_VERSION}/opencode-${OPENCODE_ARCH}.tar.gz"
        
        mkdir -p "$HOME/.local/bin"
        curl -fsSL "$OPENCODE_URL" 2>/dev/null | tar -xzf - -C "$HOME/.local/bin" 2>/dev/null && \
            chmod +x "$HOME/.local/bin/opencode" && echo "  opencode installed" || echo "  Warning: opencode install failed"
    }
    install_opencode

    echo ""
    echo "\033[1;32m=== Vibe Coder Tools Installed ===\033[0m"
    echo ""
    echo "  \033[1;36mAI Coding:\033[0m"
    command -v aider >/dev/null 2>&1 && echo "    • aider      - AI pair programming" || echo "    • aider      - (run: pip3 install aider-chat)"
    command -v opencode >/dev/null 2>&1 && echo "    • opencode   - AI coding assistant" || true
    command -v llm >/dev/null 2>&1 && echo "    • llm        - CLI for various LLMs" || true
    command -v interpreter >/dev/null 2>&1 && echo "    • interpreter- Open Interpreter" || true
    echo ""
    echo "  \033[1;33mSetup API Keys:\033[0m"
    echo "    export ANTHROPIC_API_KEY=your-key"
    echo "    export OPENAI_API_KEY=your-key"
    echo ""
    echo "  Run '\033[1;37mrexec tools\033[0m' to see all installed tools"
    echo ""
fi
`, role.Name, packages, role.Name, role.Name)

	return script, nil
}
