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
			Description: "Just vibing. I need a terminal that matches my aesthetic.",
			Icon:        "💼",
			Packages:    []string{"zsh", "git", "tmux", "screen", "python3", "cron", "htop", "vim", "zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions"},
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
    PACKAGES="%s"

    if command -v apt-get >/dev/null 2>&1; then
        export DEBIAN_FRONTEND=noninteractive
        # Wait for any existing apt locks
        wait_for_apt_lock || true
        # Use flock to prevent concurrent apt-get
        flock -w 120 /var/lib/dpkg/lock-frontend apt-get update -qq 2>/dev/null || apt-get update -qq
        flock -w 120 /var/lib/dpkg/lock-frontend apt-get install -y -qq $PACKAGES >/dev/null 2>&1 || apt-get install -y -qq $PACKAGES >/dev/null 2>&1
    elif command -v apk >/dev/null 2>&1; then
        # Alpine mapping
        apk add --no-cache $PACKAGES >/dev/null 2>&1
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y -q $PACKAGES >/dev/null 2>&1
    elif command -v yum >/dev/null 2>&1; then
        yum install -y -q $PACKAGES >/dev/null 2>&1
    elif command -v pacman >/dev/null 2>&1; then
        pacman -Sy --noconfirm $PACKAGES >/dev/null 2>&1
    else
        echo "Unsupported package manager"
        exit 1
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
`, role.Name, packages)

	return script, nil
}
