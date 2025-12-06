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
			Description: "I use Arch btw. Just give me a shell + free AI tools.",
			Icon:        "🧘",
			Packages:    []string{"zsh", "git", "curl", "wget", "vim", "nano", "htop", "jq", "neofetch", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"},
		},
		{
			ID:          "node",
			Name:        "10x JS Ninja",
			Description: "Ship fast, break things, npm install everything + free AI.",
			Icon:        "🚀",
			Packages:    []string{"zsh", "git", "nodejs", "npm", "yarn", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"},
		},
		{
			ID:          "python",
			Name:        "Data Wizard",
			Description: "Import antigravity. I speak in list comprehensions + AI.",
			Icon:        "🧙‍♂️",
			Packages:    []string{"zsh", "git", "python3", "python3-pip", "python3-venv", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"},
		},
		{
			ID:          "go",
			Name:        "The Gopher",
			Description: "If err != nil { panic(err) }. Simplicity + AI tools.",
			Icon:        "🐹",
			Packages:    []string{"zsh", "git", "make", "go", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"},
		},
		{
			ID:          "neovim",
			Name:        "Neovim God",
			Description: "My config is longer than your code. Mouse? AI helps.",
			Icon:        "⌨️",
			Packages:    []string{"zsh", "git", "neovim", "ripgrep", "gcc", "make", "curl", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"},
		},
		{
			ID:          "devops",
			Name:        "YAML Herder",
			Description: "I don't write code, I write config. AI assists.",
			Icon:        "☸️",
			Packages:    []string{"zsh", "git", "docker-cli", "kubectl", "ansible", "terraform", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"},
		},
		{
			ID:          "overemployed",
			Name:        "Vibe Coder",
			Description: "AI-powered coding: tgpt, aichat, mods, aider, opencode & more.",
			Icon:        "🤖",
			Packages:    []string{"zsh", "git", "tmux", "python3", "python3-pip", "python3-venv", "nodejs", "npm", "curl", "wget", "htop", "vim", "neovim", "ripgrep", "fzf", "jq", "tgpt", "aichat", "mods", "aider", "opencode", "llm", "sgpt", "zsh-autosuggestions", "zsh-syntax-highlighting"},
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
	// Filter out non-system packages that are handled separately
	excludedPackages := map[string]bool{
		"tgpt":                           true,
		"aichat":                         true,
		"mods":                           true,
		"zsh-autosuggestions":            true,
		"zsh-syntax-highlighting":        true,
		"zsh-history-substring-search":   true,
		"aider":                          true,
		"opencode":                       true,
		"llm":                            true,
		"sgpt":                           true,
		"neofetch":                       true, // Often not in base repos or needs contrib
	}

	packages := ""
	for _, p := range role.Packages {
		if !excludedPackages[p] {
			packages += p + " "
		}
	}

	script := fmt.Sprintf(`#!/bin/sh
# set -e removed to allow partial failures

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

# Configure Zsh if installed
configure_zsh() {
    if command -v zsh >/dev/null 2>&1; then
        echo "Configuring zsh..."

        # Ensure oh-my-zsh custom plugins directory exists
        export HOME="${HOME:-/root}"
        ZSH_CUSTOM="${HOME}/.oh-my-zsh/custom"
        mkdir -p "$ZSH_CUSTOM/plugins"

        # Install zsh plugins
        echo "Installing zsh plugins..."
        if [ ! -d "$ZSH_CUSTOM/plugins/zsh-autosuggestions" ]; then
            git clone --depth=1 https://github.com/zsh-users/zsh-autosuggestions "$ZSH_CUSTOM/plugins/zsh-autosuggestions" 2>/dev/null || true
        fi
        if [ ! -d "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" ]; then
            git clone --depth=1 https://github.com/zsh-users/zsh-syntax-highlighting "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" 2>/dev/null || true
        fi
        if [ ! -d "$ZSH_CUSTOM/plugins/zsh-history-substring-search" ]; then
            git clone --depth=1 https://github.com/zsh-users/zsh-history-substring-search "$ZSH_CUSTOM/plugins/zsh-history-substring-search" 2>/dev/null || true
        fi

        # Change default shell
        if [ -f /etc/passwd ]; then
            ZSH_PATH=$(which zsh)
            sed -i "s|root:.*:/bin/.*|root:x:0:0:root:/root:$ZSH_PATH|" /etc/passwd 2>/dev/null || true
        fi

        # Create minimal .zshrc only if it doesn't exist (preserve oh-my-zsh if installed)
        if [ ! -f /root/.zshrc ]; then
            cat > /root/.zshrc << 'ZSHRC'
export TERM=xterm-256color
export LANG=en_US.UTF-8
export PATH="$HOME/.local/bin:$PATH"

# Basic Config
autoload -Uz compinit && compinit
setopt HIST_IGNORE_ALL_DUPS SHARE_HISTORY
HISTFILE=~/.zsh_history
HISTSIZE=10000
SAVEHIST=10000

# Simple prompt that works reliably across all terminals
# Format: user@hostname directory $
PS1='%%n@%%m %%~ $ '

# Aliases
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias gs='git status'

# Welcome message on first login (only show once per session)
if [ -z "$REXEC_WELCOMED" ]; then
    export REXEC_WELCOMED=1
    echo ""
    echo "\033[1;36m Welcome to Rexec Terminal \033[0m"
    echo ""
    echo " \033[1;33mQuick Commands:\033[0m"
    echo "   rexec tools    - See installed tools"
    echo "   rexec info     - Container info"
    echo "   ai-help        - AI tools guide"
    echo "   tgpt \"question\" - Free AI (no API key)"
    echo ""
fi
ZSHRC
        fi
    fi
}

# Create rexec CLI command with subcommands
create_rexec_cli() {
    mkdir -p /root/.local/bin /home/user/.local/bin 2>/dev/null || true
    
    cat > /root/.local/bin/rexec << 'REXECCLI'
#!/bin/sh

# Rexec CLI - Terminal helper commands
VERSION="2.0.0"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[1;36m'
NC='\033[0m' # No Color

# Detect package manager
detect_pkg_manager() {
    if command -v apt-get >/dev/null 2>&1; then
        echo "apt"
    elif command -v apk >/dev/null 2>&1; then
        echo "apk"
    elif command -v dnf >/dev/null 2>&1; then
        echo "dnf"
    elif command -v yum >/dev/null 2>&1; then
        echo "yum"
    elif command -v pacman >/dev/null 2>&1; then
        echo "pacman"
    elif command -v zypper >/dev/null 2>&1; then
        echo "zypper"
    else
        echo "unknown"
    fi
}

# Package name mappings for different distros
# Format: generic_name -> apt_name:apk_name:dnf_name:pacman_name
get_pkg_name() {
    PKG="$1"
    PM="$2"
    
    # Common package mappings (generic -> apt:apk:dnf:pacman)
    case "$PKG" in
        # Development
        nodejs) 
            case "$PM" in
                apt) echo "nodejs" ;;
                apk) echo "nodejs" ;;
                dnf|yum) echo "nodejs" ;;
                pacman) echo "nodejs" ;;
                *) echo "$PKG" ;;
            esac ;;
        python)
            case "$PM" in
                apt) echo "python3" ;;
                apk) echo "python3" ;;
                dnf|yum) echo "python3" ;;
                pacman) echo "python" ;;
                *) echo "$PKG" ;;
            esac ;;
        pip)
            case "$PM" in
                apt) echo "python3-pip" ;;
                apk) echo "py3-pip" ;;
                dnf|yum) echo "python3-pip" ;;
                pacman) echo "python-pip" ;;
                *) echo "$PKG" ;;
            esac ;;
        # Docker
        docker)
            case "$PM" in
                apt) echo "docker.io" ;;
                apk) echo "docker" ;;
                dnf|yum) echo "docker" ;;
                pacman) echo "docker" ;;
                *) echo "$PKG" ;;
            esac ;;
        docker-compose)
            case "$PM" in
                apt) echo "docker-compose" ;;
                apk) echo "docker-compose" ;;
                dnf|yum) echo "docker-compose" ;;
                pacman) echo "docker-compose" ;;
                *) echo "$PKG" ;;
            esac ;;
        # Editors
        neovim|nvim)
            case "$PM" in
                apt) echo "neovim" ;;
                apk) echo "neovim" ;;
                dnf|yum) echo "neovim" ;;
                pacman) echo "neovim" ;;
                *) echo "neovim" ;;
            esac ;;
        # Tools
        ripgrep|rg)
            case "$PM" in
                apt) echo "ripgrep" ;;
                apk) echo "ripgrep" ;;
                dnf|yum) echo "ripgrep" ;;
                pacman) echo "ripgrep" ;;
                *) echo "ripgrep" ;;
            esac ;;
        fd)
            case "$PM" in
                apt) echo "fd-find" ;;
                apk) echo "fd" ;;
                dnf|yum) echo "fd-find" ;;
                pacman) echo "fd" ;;
                *) echo "$PKG" ;;
            esac ;;
        bat)
            case "$PM" in
                apt) echo "bat" ;;
                apk) echo "bat" ;;
                dnf|yum) echo "bat" ;;
                pacman) echo "bat" ;;
                *) echo "$PKG" ;;
            esac ;;
        # Default: use as-is
        *) echo "$PKG" ;;
    esac
}

# Popular tools database for suggestions
POPULAR_TOOLS="git curl wget vim nano htop tmux zsh neovim ripgrep fzf jq yq bat fd exa tree ncdu docker docker-compose kubectl terraform ansible python nodejs npm yarn go rust cargo gcc make cmake nginx redis postgresql mysql sqlite mongodb"

# Fuzzy match for suggestions
find_similar() {
    QUERY="$1"
    MATCHES=""
    
    for tool in $POPULAR_TOOLS; do
        case "$tool" in
            *"$QUERY"*) MATCHES="$MATCHES $tool" ;;
        esac
    done
    
    # Also check if query is substring
    if [ -z "$MATCHES" ]; then
        for tool in $POPULAR_TOOLS; do
            # Check if first 2+ chars match
            PREFIX=$(echo "$QUERY" | cut -c1-2)
            case "$tool" in
                "$PREFIX"*) MATCHES="$MATCHES $tool" ;;
            esac
        done
    fi
    
    echo "$MATCHES" | xargs
}

# Install a package
do_install() {
    if [ -z "$1" ]; then
        echo "${RED}Error: No package specified${NC}"
        echo "Usage: rexec install <package>"
        echo ""
        echo "Examples:"
        echo "  rexec install docker"
        echo "  rexec install nodejs"
        echo "  rexec install neovim"
        return 1
    fi
    
    PKG="$1"
    PM=$(detect_pkg_manager)
    
    if [ "$PM" = "unknown" ]; then
        echo "${RED}Error: No supported package manager found${NC}"
        return 1
    fi
    
    # Check if already installed
    if command -v "$PKG" >/dev/null 2>&1; then
        echo "${GREEN}✓${NC} $PKG is already installed"
        return 0
    fi
    
    # Get distro-specific package name
    ACTUAL_PKG=$(get_pkg_name "$PKG" "$PM")
    
    echo "${CYAN}Installing $PKG...${NC}"
    echo "  Package manager: $PM"
    echo "  Package name: $ACTUAL_PKG"
    echo ""
    
    case "$PM" in
        apt)
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -qq
            apt-get install -y "$ACTUAL_PKG"
            ;;
        apk)
            apk add --no-cache "$ACTUAL_PKG"
            ;;
        dnf)
            dnf install -y "$ACTUAL_PKG"
            ;;
        yum)
            yum install -y "$ACTUAL_PKG"
            ;;
        pacman)
            pacman -Sy --noconfirm "$ACTUAL_PKG"
            ;;
        zypper)
            zypper --non-interactive install "$ACTUAL_PKG"
            ;;
    esac
    
    RESULT=$?
    if [ $RESULT -eq 0 ]; then
        echo ""
        echo "${GREEN}✓ $PKG installed successfully${NC}"
    else
        echo ""
        echo "${RED}✗ Failed to install $PKG${NC}"
        
        # Suggest similar packages
        SIMILAR=$(find_similar "$PKG")
        if [ -n "$SIMILAR" ]; then
            echo ""
            echo "${YELLOW}Did you mean one of these?${NC}"
            for s in $SIMILAR; do
                echo "  - $s"
            done
        fi
    fi
    
    return $RESULT
}

# Uninstall a package
do_uninstall() {
    if [ -z "$1" ]; then
        echo "${RED}Error: No package specified${NC}"
        echo "Usage: rexec uninstall <package>"
        return 1
    fi
    
    PKG="$1"
    PM=$(detect_pkg_manager)
    
    if [ "$PM" = "unknown" ]; then
        echo "${RED}Error: No supported package manager found${NC}"
        return 1
    fi
    
    ACTUAL_PKG=$(get_pkg_name "$PKG" "$PM")
    
    echo "${CYAN}Uninstalling $PKG...${NC}"
    
    case "$PM" in
        apt)
            apt-get remove -y "$ACTUAL_PKG"
            ;;
        apk)
            apk del "$ACTUAL_PKG"
            ;;
        dnf)
            dnf remove -y "$ACTUAL_PKG"
            ;;
        yum)
            yum remove -y "$ACTUAL_PKG"
            ;;
        pacman)
            pacman -R --noconfirm "$ACTUAL_PKG"
            ;;
        zypper)
            zypper --non-interactive remove "$ACTUAL_PKG"
            ;;
    esac
    
    RESULT=$?
    if [ $RESULT -eq 0 ]; then
        echo "${GREEN}✓ $PKG uninstalled${NC}"
    else
        echo "${RED}✗ Failed to uninstall $PKG${NC}"
    fi
    
    return $RESULT
}

# Search for packages
do_search() {
    if [ -z "$1" ]; then
        echo "${RED}Error: No search term specified${NC}"
        echo "Usage: rexec search <term>"
        return 1
    fi
    
    TERM="$1"
    PM=$(detect_pkg_manager)
    
    echo "${CYAN}Searching for '$TERM'...${NC}"
    echo ""
    
    case "$PM" in
        apt)
            apt-cache search "$TERM" | head -20
            ;;
        apk)
            apk search "$TERM" | head -20
            ;;
        dnf)
            dnf search "$TERM" 2>/dev/null | head -20
            ;;
        yum)
            yum search "$TERM" 2>/dev/null | head -20
            ;;
        pacman)
            pacman -Ss "$TERM" | head -20
            ;;
        zypper)
            zypper search "$TERM" | head -20
            ;;
        *)
            echo "${RED}Package search not supported on this system${NC}"
            return 1
            ;;
    esac
    
    echo ""
    echo "${YELLOW}Tip:${NC} Use 'rexec install <package>' to install"
}

# List available/popular packages
do_list() {
    echo ""
    echo "${CYAN}=== Popular Packages ===${NC}"
    echo ""
    echo "${YELLOW}Development:${NC}"
    echo "  nodejs npm yarn python pip go rust cargo gcc make cmake"
    echo ""
    echo "${YELLOW}Editors:${NC}"
    echo "  vim neovim nano emacs"
    echo ""
    echo "${YELLOW}Tools:${NC}"
    echo "  git curl wget htop tmux zsh fzf ripgrep bat fd jq yq tree ncdu"
    echo ""
    echo "${YELLOW}DevOps:${NC}"
    echo "  docker docker-compose kubectl terraform ansible helm"
    echo ""
    echo "${YELLOW}Databases:${NC}"
    echo "  redis postgresql mysql sqlite mongodb"
    echo ""
    echo "${YELLOW}Web:${NC}"
    echo "  nginx apache2 caddy"
    echo ""
    echo "Use 'rexec install <package>' to install any package"
    echo "Use 'rexec search <term>' to find packages"
    echo ""
}

show_help() {
    echo ""
    echo "${CYAN}Rexec CLI v${VERSION}${NC}"
    echo ""
    echo "Usage: rexec <command> [arguments]"
    echo ""
    echo "${YELLOW}Commands:${NC}"
    echo "  tools              Show installed tools and their status"
    echo "  info               Show container information"
    echo "  install <pkg>      Install a package (auto-detects OS)"
    echo "  uninstall <pkg>    Uninstall a package"
    echo "  search <term>      Search for packages"
    echo "  list               List popular packages"
    echo "  help               Show this help message"
    echo ""
    echo "${YELLOW}Examples:${NC}"
    echo "  rexec install docker"
    echo "  rexec install nodejs"
    echo "  rexec search redis"
    echo "  rexec uninstall vim"
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
    echo "\033[1;33mAI Tools (Free - No API Key):\033[0m"
    FREE_AI_FOUND=0
    for cmd in tgpt aichat mods; do
        if command -v $cmd >/dev/null 2>&1; then
            echo "  \033[32m✓\033[0m $cmd"
            FREE_AI_FOUND=1
        fi
    done
    if [ "$FREE_AI_FOUND" = "0" ]; then
        echo "  \033[90m(none installed)\033[0m"
    fi

    echo ""
    echo "\033[1;33mAI Tools (API Key Required):\033[0m"
    AI_FOUND=0
    for cmd in aider opencode llm sgpt claude; do
        if command -v $cmd >/dev/null 2>&1; then
            echo "  \033[32m✓\033[0m $cmd"
            AI_FOUND=1
        fi
    done
    
    # Check pip-installed tools that might not be in PATH
    if command -v pip3 >/dev/null 2>&1; then
        for pkg in aider-chat llm shell-gpt; do
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
    echo "\033[38;5;243mRun 'ai-help' for AI tools usage guide\033[0m"
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
        UPTIME_HRS=$(((UPTIME_SEC %% 86400) / 3600))
        UPTIME_MIN=$(((UPTIME_SEC %% 3600) / 60))
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
    install)
        shift
        do_install "$@"
        ;;
    uninstall|remove)
        shift
        do_uninstall "$@"
        ;;
    search)
        shift
        do_search "$@"
        ;;
    list)
        do_list
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo "${RED}Unknown command: $1${NC}"
        echo ""
        # Try to suggest if it looks like a package name
        SIMILAR=$(find_similar "$1")
        if [ -n "$SIMILAR" ]; then
            echo "${YELLOW}Did you mean to install one of these?${NC}"
            for s in $SIMILAR; do
                echo "  rexec install $s"
            done
            echo ""
        fi
        show_help
        exit 1
        ;;
esac
REXECCLI

    chmod +x /root/.local/bin/rexec
    cp /root/.local/bin/rexec /home/user/.local/bin/rexec 2>/dev/null || true
    chmod +x /home/user/.local/bin/rexec 2>/dev/null || true
    
    # Symlink to global path to ensure it's always found
    ln -sf /root/.local/bin/rexec /usr/local/bin/rexec 2>/dev/null || true
    ln -sf /root/.local/bin/rexec /usr/bin/rexec 2>/dev/null || true
}

# Setup PATH for all roles - ensures installed tools are found
setup_path() {
    # Ensure config files exist
    touch /root/.zshrc /root/.bashrc /root/.profile 2>/dev/null || true
    mkdir -p /home/user 2>/dev/null || true
    touch /home/user/.zshrc /home/user/.bashrc /home/user/.profile 2>/dev/null || true
    chown -R user:user /home/user 2>/dev/null || true

    for rcfile in /root/.zshrc /root/.bashrc /root/.profile /home/user/.zshrc /home/user/.bashrc /home/user/.profile; do
        if [ -f "$rcfile" ]; then
            if ! grep -q '.local/bin' "$rcfile" 2>/dev/null; then
                echo '' >> "$rcfile"
                echo '# Add rexec tools to PATH' >> "$rcfile"
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

# Install free AI tools for ALL roles (no API key required)
install_free_ai_tools() {
    echo "Installing free AI terminal tools..."
    export HOME="${HOME:-/root}"
    mkdir -p "$HOME/.local/bin"
    
    # tgpt - Free GPT in terminal (no API key, uses free providers)
    # https://github.com/aandrew-me/tgpt
    echo "  Installing tgpt (free terminal GPT)..."
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) TGPT_ARCH="amd64" ;;
        aarch64|arm64) TGPT_ARCH="arm64" ;;
        *) TGPT_ARCH="" ;;
    esac
    if [ -n "$TGPT_ARCH" ]; then
        TGPT_URL="https://github.com/aandrew-me/tgpt/releases/latest/download/tgpt-linux-${TGPT_ARCH}"
        curl -fsSL "$TGPT_URL" -o "$HOME/.local/bin/tgpt" 2>/dev/null && \
            chmod +x "$HOME/.local/bin/tgpt" && echo "    ✓ tgpt installed" || echo "    ! tgpt install failed"
    fi
    
    # aichat - Feature-rich AI CLI chat (supports local models via ollama)
    # https://github.com/sigoden/aichat
    echo "  Installing aichat (AI terminal chat)..."
    case "$ARCH" in
        x86_64|amd64) 
            if ldd /bin/ls 2>/dev/null | grep -q musl; then
                AICHAT_ARCH="x86_64-unknown-linux-musl"
            else
                AICHAT_ARCH="x86_64-unknown-linux-gnu"
            fi
            ;;
        aarch64|arm64)
            if ldd /bin/ls 2>/dev/null | grep -q musl; then
                AICHAT_ARCH="aarch64-unknown-linux-musl"
            else
                AICHAT_ARCH="aarch64-unknown-linux-gnu"
            fi
            ;;
        *) AICHAT_ARCH="" ;;
    esac
    if [ -n "$AICHAT_ARCH" ]; then
        AICHAT_VERSION=$(curl -s https://api.github.com/repos/sigoden/aichat/releases/latest 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
        [ -z "$AICHAT_VERSION" ] && AICHAT_VERSION="0.26.0"
        AICHAT_URL="https://github.com/sigoden/aichat/releases/download/v${AICHAT_VERSION}/aichat-v${AICHAT_VERSION}-${AICHAT_ARCH}.tar.gz"
        curl -fsSL "$AICHAT_URL" 2>/dev/null | tar -xzf - -C "$HOME/.local/bin" 2>/dev/null && \
            chmod +x "$HOME/.local/bin/aichat" && echo "    ✓ aichat installed" || echo "    ! aichat install failed"
    fi
    
    # mods - AI for the command line (works great with ollama)
    # https://github.com/charmbracelet/mods
    echo "  Installing mods (AI for CLI)..."
    case "$ARCH" in
        x86_64|amd64) MODS_ARCH="Linux_x86_64" ;;
        aarch64|arm64) MODS_ARCH="Linux_arm64" ;;
        *) MODS_ARCH="" ;;
    esac
    if [ -n "$MODS_ARCH" ]; then
        MODS_VERSION=$(curl -s https://api.github.com/repos/charmbracelet/mods/releases/latest 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
        [ -z "$MODS_VERSION" ] && MODS_VERSION="1.6.0"
        MODS_URL="https://github.com/charmbracelet/mods/releases/download/v${MODS_VERSION}/mods_${MODS_VERSION}_${MODS_ARCH}.tar.gz"
        curl -fsSL "$MODS_URL" 2>/dev/null | tar -xzf - -C "$HOME/.local/bin" mods 2>/dev/null && \
            chmod +x "$HOME/.local/bin/mods" && echo "    ✓ mods installed" || echo "    ! mods install failed"
    fi
    
    # Install opencode (sst/opencode) - binary release
    echo "  Installing opencode (AI coding assistant)..."
    install_opencode() {
        export HOME="${HOME:-/root}"
        
        if command -v opencode >/dev/null 2>&1; then
            echo "    ✓ opencode already installed"
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
                echo "    ! Unsupported architecture: $ARCH"
                return 1
                ;;
        esac
        
        OPENCODE_VERSION=$(curl -s https://api.github.com/repos/sst/opencode/releases/latest 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
        [ -z "$OPENCODE_VERSION" ] && OPENCODE_VERSION="v1.0.133"
        
        OPENCODE_URL="https://github.com/sst/opencode/releases/download/${OPENCODE_VERSION}/opencode-${OPENCODE_ARCH}.tar.gz"
        
        mkdir -p "$HOME/.local/bin"
        curl -fsSL "$OPENCODE_URL" 2>/dev/null | tar -xzf - -C "$HOME/.local/bin" 2>/dev/null && \
            chmod +x "$HOME/.local/bin/opencode" && echo "    ✓ opencode installed" || echo "    ! opencode install failed"
    }
    install_opencode
    
    # Create helper script to show AI tools usage
    cat > "$HOME/.local/bin/ai-help" << 'AIHELP'
#!/bin/sh
echo ""
echo "\033[1;36m=== Free AI Terminal Tools ===\033[0m"
echo ""
echo "\033[1;33mNo API Key Required:\033[0m"
echo "  tgpt \"your question\"     - Free GPT (uses free providers)"
echo "  tgpt -i                   - Interactive chat mode"
echo "  tgpt -c \"code question\"  - Code generation mode"
echo ""
echo "\033[1;33mWith Ollama (local LLM):\033[0m"
echo "  # First install ollama: curl -fsSL https://ollama.com/install.sh | sh"
echo "  # Then pull a model: ollama pull llama3.2"
echo "  aichat                    - Interactive AI chat"
echo "  echo \"question\" | mods   - Pipe to AI"
echo "  mods -m ollama:llama3.2   - Use specific model"
echo ""
echo "\033[1;33mWith API Keys:\033[0m"
echo "  aider                     - AI pair programming"
echo "  opencode                  - AI coding assistant"
echo "  llm \"question\"           - CLI for various LLMs"
echo ""
echo "\033[38;5;243mRun 'rexec tools' to see all installed tools\033[0m"
echo ""
AIHELP
    chmod +x "$HOME/.local/bin/ai-help"
    cp "$HOME/.local/bin/ai-help" /home/user/.local/bin/ai-help 2>/dev/null || true
}

# --- Execution ---

echo "Setting up Rexec environment..."

# 1. Install CLI and setup paths first (critical)
echo "[[REXEC_STATUS]]Installing Rexec CLI..."
create_rexec_cli
setup_path
save_role_info

# 2. Install packages (might take time or fail, but CLI is ready)
echo "[[REXEC_STATUS]]Installing system packages..."
install_role_packages

# 3. Configure Zsh (if installed)
echo "[[REXEC_STATUS]]Configuring shell..."
configure_zsh

# 4. Install AI tools
echo "[[REXEC_STATUS]]Installing AI tools..."
install_free_ai_tools

# Special handling for Vibe Coder role - install additional AI CLI tools
if [ "%s" = "Vibe Coder" ]; then
    echo "[[REXEC_STATUS]]Installing extra tools..."
    echo "Installing additional AI coding tools..."
    
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
        
        # Install aider - the main AI pair programming tool (needs API key)
        echo "  Installing aider (AI pair programming)..."
        $PIP install --quiet --break-system-packages aider-chat 2>/dev/null || $PIP install --quiet aider-chat 2>/dev/null || echo "    ! aider install failed"

        # Install llm - versatile CLI for LLMs
        echo "  Installing llm (CLI for LLMs)..."
        $PIP install --quiet --break-system-packages llm 2>/dev/null || $PIP install --quiet llm 2>/dev/null || echo "    ! llm install failed"

        # Install shell-gpt - another great CLI tool
        echo "  Installing shell-gpt (sgpt)..."
        $PIP install --quiet --break-system-packages shell-gpt 2>/dev/null || $PIP install --quiet shell-gpt 2>/dev/null || echo "    ! sgpt install failed"
    fi

    echo ""
    echo "\033[1;32m=== Vibe Coder AI Tools ===\033[0m"
    echo ""
    echo "  \033[1;36mFree (No API Key):\033[0m"
    command -v tgpt >/dev/null 2>&1 && echo "    ✓ tgpt      - Free terminal GPT" || true
    command -v aichat >/dev/null 2>&1 && echo "    ✓ aichat    - AI chat (supports ollama)" || true
    command -v mods >/dev/null 2>&1 && echo "    ✓ mods      - AI for CLI" || true
    echo ""
    echo "  \033[1;36mWith API Keys:\033[0m"
    command -v aider >/dev/null 2>&1 && echo "    ✓ aider     - AI pair programming" || echo "    · aider     - pip3 install aider-chat"
    command -v opencode >/dev/null 2>&1 && echo "    ✓ opencode  - AI coding assistant" || true
    command -v llm >/dev/null 2>&1 && echo "    ✓ llm       - CLI for various LLMs" || true
    command -v sgpt >/dev/null 2>&1 && echo "    ✓ sgpt      - Shell GPT" || true
    echo ""
    echo "  \033[1;33mQuick Start:\033[0m"
    echo "    tgpt \"how do I list files?\"   # Free, no setup"
    echo "    ai-help                        # Show all AI tools"
    echo ""
fi

echo "[[REXEC_STATUS]]Setup complete."
`, role.Name, packages, role.Name, role.Name)

	return script, nil
}
