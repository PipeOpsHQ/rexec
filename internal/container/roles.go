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

# Wait for any existing apt/dpkg locks (max 60 seconds)
wait_for_apt_lock() {
    local max_wait=60
    local waited=0
    
    # Check if fuser is available
    if ! command -v fuser >/dev/null 2>&1; then
        echo "fuser not found, using simple file check..."
        while [ -f /var/lib/dpkg/lock-frontend ] || [ -f /var/lib/dpkg/lock ] || [ -f /var/lib/apt/lists/lock ]; do
            if [ $waited -ge $max_wait ]; then
                echo "Timeout waiting for apt lock"
                return 1
            fi
            echo "Waiting for apt lock release..."
            sleep 2
            waited=$((waited + 2))
        done
        return 0
    fi

    while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 || fuser /var/lib/apt/lists/lock >/dev/null 2>&1; do
        if [ $waited -ge $max_wait ]; then
            echo "Timeout waiting for apt lock"
            return 1
        fi
        sleep 2
        waited=$((waited + 2))
    done
    return 0
}

# Function to install packages based on detected manager
install_role_packages() {
    GENERIC_PACKAGES="%s"
    PACKAGES="$GENERIC_PACKAGES"

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

        # Wait for any existing apt locks (fallback if Timeout option isn't supported/enough)
        wait_for_apt_lock || true
        
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
# PROMPT='%F{cyan}%n%f@%F{blue}%m%f %F{yellow}%~%f %(?:%F{green}➜:%F{red}➜) %f'

# Aliases
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias gs='git status'
ZSHRC
fi

echo "Role setup complete!"

# Special handling for Vibe Coder role - install AI CLI tools
if [ "%s" = "Vibe Coder" ]; then
    echo "Installing AI CLI tools..."

    # Ensure pip is available
    if command -v pip3 >/dev/null 2>&1; then
        PIP="pip3"
    elif command -v pip >/dev/null 2>&1; then
        PIP="pip"
    else
        echo "pip not found, skipping Python-based AI tools"
        PIP=""
    fi

    if [ -n "$PIP" ]; then
        # Install popular AI coding assistants
        echo "Installing aider (AI pair programming)..."
        $PIP install --quiet --break-system-packages aider-chat 2>/dev/null || $PIP install --quiet aider-chat 2>/dev/null || true

        echo "Installing opencode (AI coding assistant)..."
        $PIP install --quiet --break-system-packages opencode-ai 2>/dev/null || $PIP install --quiet opencode-ai 2>/dev/null || true

        echo "Installing claude-cli (Anthropic Claude)..."
        $PIP install --quiet --break-system-packages anthropic 2>/dev/null || $PIP install --quiet anthropic 2>/dev/null || true

        echo "Installing llm (CLI for LLMs)..."
        $PIP install --quiet --break-system-packages llm 2>/dev/null || $PIP install --quiet llm 2>/dev/null || true

        echo "Installing gpt-engineer..."
        $PIP install --quiet --break-system-packages gpt-engineer 2>/dev/null || $PIP install --quiet gpt-engineer 2>/dev/null || true

        echo "Installing interpreter (Open Interpreter)..."
        $PIP install --quiet --break-system-packages open-interpreter 2>/dev/null || $PIP install --quiet open-interpreter 2>/dev/null || true

        echo "Installing fabric (AI augmentation)..."
        $PIP install --quiet --break-system-packages fabric-ai 2>/dev/null || $PIP install --quiet fabric-ai 2>/dev/null || true
    fi

    # Install Node.js based AI tools
    if command -v npm >/dev/null 2>&1; then
        echo "Installing AI CLI tools via npm..."

        # Claude Code (Anthropic's official CLI)
        npm install -g @anthropic-ai/claude-code 2>/dev/null || true

        # GitHub Copilot CLI
        npm install -g @githubnext/github-copilot-cli 2>/dev/null || true
    fi

    # Install Go-based tools
    if command -v go >/dev/null 2>&1; then
        echo "Installing Go-based AI tools..."
        go install github.com/charmbracelet/mods@latest 2>/dev/null || true
    fi

    echo ""
    echo "AI CLI Tools Installed:"
    echo "  • aider      - AI pair programming (aider --help)"
    echo "  • opencode   - AI coding assistant (opencode --help)"
    echo "  • llm        - CLI for various LLMs (llm --help)"
    echo "  • interpreter- Open Interpreter (interpreter --help)"
    echo ""
    echo "Note: Some tools require API keys. Set them in your environment:"
    echo "  export ANTHROPIC_API_KEY=your-key"
    echo "  export OPENAI_API_KEY=your-key"
    echo ""
fi
`, role.Name, packages, role.Name)

	return script, nil
}
