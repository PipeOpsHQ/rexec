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
			Packages:    []string{"zsh", "git", "docker", "kubectl", "ansible", "terraform", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"},
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
		"gum":                            true,
		"zsh-autosuggestions":            true,
		"zsh-syntax-highlighting":        true,
		"zsh-history-substring-search":   true,
		"aider":                          true,
		"opencode":                       true,
		"llm":                            true,
		"sgpt":                           true,
	}

	packages := ""
	for _, p := range role.Packages {
		if !excludedPackages[p] {
			packages += p + " "
		}
	}

	script := fmt.Sprintf(`#!/bin/sh
# Explicitly disable exit-on-error to allow partial failures
set +e

echo "Installing tools for role: %s..."

# Fix any corrupted dpkg state first (common issue in containers)
fix_dpkg() {
    if [ -d /var/lib/dpkg/updates ] && [ "$(ls -A /var/lib/dpkg/updates 2>/dev/null)" ]; then
        echo "Fixing dpkg state..."
        rm -f /var/lib/dpkg/updates/* 2>/dev/null || true
        dpkg --configure -a 2>/dev/null || true
    fi
    # Clear stale lock files
    rm -f /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock /var/cache/apt/archives/lock 2>/dev/null || true
}

# Wait for any existing package manager locks (max 60 seconds)
wait_for_locks() {
    max_wait=60
    waited=0
    
    # List of known lock files
    locks="/var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock /lib/apk/db/lock /var/run/dnf.pid /var/run/yum.pid /var/lib/pacman/db.lck"
    
    while [ $waited -lt $max_wait ]; do
        locked=0
        for lock in $locks; do
            if [ -f "$lock" ]; then
                # Check if process is holding the lock
                if command -v fuser >/dev/null 2>&1; then
                    if fuser "$lock" >/dev/null 2>&1; then
                        locked=1
                        break
                    fi
                else
                    # No fuser, check if lock file is recent (less than 5 min old)
                    if [ "$(find "$lock" -mmin -5 2>/dev/null)" ]; then
                        locked=1
                        break
                    fi
                fi
            fi
        done
        
        if [ $locked -eq 0 ]; then
            return 0
        fi
        
        echo "Waiting for package manager lock release..."
        sleep 2
        waited=$((waited + 2))
    done
    
    echo "Timeout waiting for lock, attempting to clear stale locks..."
    for lock in $locks; do
        rm -f "$lock" 2>/dev/null || true
    done
    return 0
}

# Function to install packages based on detected manager
install_role_packages() {
    touch /tmp/.rexec_installing_system
    GENERIC_PACKAGES="%s"
    PACKAGES="$GENERIC_PACKAGES"
    
    # Fix dpkg and wait for locks before starting
    fix_dpkg
    wait_for_locks || true

    if command -v apt-get >/dev/null 2>&1; then
        export DEBIAN_FRONTEND=noninteractive
        
        # Apt options for robustness
        APT_OPTS="-o DPkg::Lock::Timeout=60 -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold"
        
        # Enable universe repository for Ubuntu (needed for neovim, ripgrep, etc.)
        if grep -q "Ubuntu" /etc/issue 2>/dev/null || grep -q "Ubuntu" /etc/os-release 2>/dev/null; then
            echo "Enabling universe repository..."
            apt-get $APT_OPTS update -qq 2>&1 || true
            apt-get $APT_OPTS install -y -qq software-properties-common 2>&1 || true
            add-apt-repository -y universe 2>&1 || true
        fi

        # Map generic package names to apt names
        APT_PACKAGES=""
        for pkg in $PACKAGES; do
            case "$pkg" in
                docker) APT_PACKAGES="$APT_PACKAGES docker.io" ;;
                *) APT_PACKAGES="$APT_PACKAGES $pkg" ;;
            esac
        done
        PACKAGES="$APT_PACKAGES"
        
        # Update package lists - handle flock not being available
        if command -v flock >/dev/null 2>&1; then
            flock -w 120 /var/lib/dpkg/lock-frontend apt-get $APT_OPTS update -qq 2>&1 || apt-get $APT_OPTS update -qq 2>&1 || true
        else
            apt-get $APT_OPTS update -qq 2>&1 || true
        fi

        # Try bulk install first
        echo "Installing packages: $PACKAGES"
        if command -v flock >/dev/null 2>&1; then
            if ! flock -w 120 /var/lib/dpkg/lock-frontend apt-get $APT_OPTS install -y -qq $PACKAGES 2>&1; then
                echo "Bulk install failed, trying individual packages..."
                for pkg in $PACKAGES; do
                    echo "Installing $pkg..."
                    apt-get $APT_OPTS install -y -qq "$pkg" 2>&1 || echo "Warning: Failed to install $pkg"
                done
            fi
        else
            if ! apt-get $APT_OPTS install -y -qq $PACKAGES 2>&1; then
                echo "Bulk install failed, trying individual packages..."
                for pkg in $PACKAGES; do
                    echo "Installing $pkg..."
                    apt-get $APT_OPTS install -y -qq "$pkg" 2>&1 || echo "Warning: Failed to install $pkg"
                done
            fi
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
    rm -f /tmp/.rexec_installing_system
}

# Configure Zsh if installed
configure_zsh() {
    if command -v zsh >/dev/null 2>&1; then
        echo "Configuring zsh..."

        # Install Oh My Zsh
        export HOME="${HOME:-/root}"
        export ZSH="$HOME/.oh-my-zsh"
        if [ ! -d "$ZSH" ]; then
            echo "Installing Oh My Zsh..."
            git clone --depth=1 https://github.com/ohmyzsh/ohmyzsh.git "$ZSH" 2>/dev/null || true
        fi

        # Install Plugins
        ZSH_CUSTOM="$ZSH/custom"
        mkdir -p "$ZSH_CUSTOM/plugins"
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
        if [ ! -d "$ZSH_CUSTOM/plugins/zsh-completions" ]; then
            git clone --depth=1 https://github.com/zsh-users/zsh-completions "$ZSH_CUSTOM/plugins/zsh-completions" 2>/dev/null || true
        fi

        # Change default shell
        if [ -f /etc/passwd ]; then
            ZSH_PATH=$(which zsh)
            sed -i "s|root:.*:/bin/.*|root:x:0:0:root:/root:$ZSH_PATH|" /etc/passwd 2>/dev/null || true
        fi

        # Define ZSHRC content once
        ZSHRC_CONTENT=$(cat << 'ZSHRC_TEMPLATE'
export TERM=xterm-256color
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
export PATH="$HOME/.local/bin:$PATH"
export ZSH="$HOME/.oh-my-zsh"

ZSH_THEME="robbyrussell"

plugins=(git zsh-completions command-not-found colored-man-pages extract sudo)

HISTSIZE=10000
SAVEHIST=10000
HISTFILE=~/.zsh_history
setopt HIST_IGNORE_ALL_DUPS HIST_FIND_NO_DUPS HIST_SAVE_NO_DUPS
setopt SHARE_HISTORY APPEND_HISTORY INC_APPEND_HISTORY PROMPT_SUBST
unsetopt PROMPT_SP # Prevent partial line indicator (%%)

autoload -Uz compinit && compinit

unset PS1 # Ensure themes can set their own PS1
source $ZSH/oh-my-zsh.sh

alias ll='ls -alF --color=auto'
alias ls='ls --color=auto'
alias gs='git status'

if [ -z "$REXEC_WELCOMED" ]; then
    export REXEC_WELCOMED=1
    echo ""
    printf "\033[1;36m Welcome to Rexec Terminal \033[0m\n"
    echo ""
    printf " \033[1;33mQuick Commands:\033[0m\n"
    echo "   rexec tools    - See installed tools"
    echo "   rexec info     - Container info"
    echo "   ai-help        - AI tools guide"
    echo "   tgpt \"question\" - Free AI (no API key)"
    echo ""
fi
ZSHRC_TEMPLATE
)

        # Write to /root/.zshrc
        if [ ! -f /root/.zshrc ]; then
            echo "$ZSHRC_CONTENT" > /root/.zshrc
        else
            echo "  .zshrc already exists, skipping overwrite..."
        fi

        # Write to /home/user/.zshrc and correct HOME path
        if [ ! -f /home/user/.zshrc ]; then
            echo "$ZSHRC_CONTENT" | \
                sed "s|export HOME=\"\\${HOME:-/root}\"|export HOME=\"/home/user\"|" | \
                sed "s|export PATH=\"/root/.local/bin:\\$PATH\"|export PATH=\"/home/user/.local/bin:\\$PATH\"|" \
                > /home/user/.zshrc
        else
            echo "  user .zshrc already exists, skipping overwrite..."
        fi
        
        # Setup user environment
        if id "user" >/dev/null 2>&1; then
            mkdir -p /home/user
            if [ -d /root/.oh-my-zsh ]; then
                cp -r /root/.oh-my-zsh /home/user/.oh-my-zsh
            fi
            chown -R user:user /home/user
            chmod 644 /home/user/.zshrc 2>/dev/null || true
        fi
    fi
}

# Create rexec CLI command with subcommands
create_rexec_cli() {
    mkdir -p /root/.local/bin /home/user/.local/bin 2>/dev/null || true
    
    cat > /root/.local/bin/rexec << 'REXECCLI'
#!/bin/sh

# Rexec CLI - Terminal helper commands
VERSION="2.1.0"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[1;36m'
NC='\033[0m' # No Color

# Check for gum (Interactive UI)
HAS_GUM=0
if command -v gum >/dev/null 2>&1; then
    HAS_GUM=1
fi

# Detect package manager
detect_pkg_manager() {
    if command -v apt-get >/dev/null 2>&1; then echo "apt"
    elif command -v apk >/dev/null 2>&1; then echo "apk"
    elif command -v dnf >/dev/null 2>&1; then echo "dnf"
    elif command -v yum >/dev/null 2>&1; then echo "yum"
    elif command -v pacman >/dev/null 2>&1; then echo "pacman"
    elif command -v zypper >/dev/null 2>&1; then echo "zypper"
    else echo "unknown"
    fi
}

# Package name mappings
get_pkg_name() {
    PKG="$1"
    PM="$2"
    case "$PKG" in
        nodejs) echo "nodejs" ;;
        python) [ "$PM" = "pacman" ] && echo "python" || echo "python3" ;;
        pip) [ "$PM" = "apk" ] && echo "py3-pip" || ([ "$PM" = "pacman" ] && echo "python-pip" || echo "python3-pip") ;;
        docker) [ "$PM" = "apt" ] && echo "docker.io" || echo "docker" ;;
        neovim|nvim) echo "neovim" ;;
        ripgrep|rg) echo "ripgrep" ;;
        fd) [ "$PM" = "apt" ] || [ "$PM" = "dnf" ] && echo "fd-find" || echo "fd" ;;
        bat) echo "bat" ;;
        *) echo "$PKG" ;;
    esac
}

# Install a package
do_install() {
    PKG="$1"
    # Interactive selection if no package provided
    if [ -z "$PKG" ] && [ "$HAS_GUM" -eq 1 ]; then
        echo "Select a tool to install or type custom name:"
        POPULAR="neovim\nripgrep\nfzf\njq\nbat\ndocker\nnodejs\npython\ngolang\nrust\ntgpt\naichat\nmods"
        PKG=$(echo "$POPULAR" | gum filter --placeholder "Select or type package name")
    fi
    
    if [ -z "$PKG" ]; then
        printf "${RED}Error: No package specified${NC}\n"
        echo "Usage: rexec install <package>"
        return 1
    fi
    
    PM=$(detect_pkg_manager)
    if [ "$PM" = "unknown" ]; then
        echo "${RED}Error: No supported package manager found${NC}"
        return 1
    fi
    
    ACTUAL_PKG=$(get_pkg_name "$PKG" "$PM")
    
    if [ "$HAS_GUM" -eq 1 ]; then
        gum style --foreground 212 "Installing $PKG ($ACTUAL_PKG)..."
    else
        printf "${CYAN}Installing $PKG ($ACTUAL_PKG)...${NC}\n"
    fi
    
    # Run install command
    case "$PM" in
        apt)
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -qq >/dev/null 2>&1
            apt-get install -y "$ACTUAL_PKG"
            ;;
        apk) apk add --no-cache "$ACTUAL_PKG" ;;
        dnf) dnf install -y "$ACTUAL_PKG" ;;
        yum) yum install -y "$ACTUAL_PKG" ;;
        pacman) pacman -Sy --noconfirm "$ACTUAL_PKG" ;;
        zypper) zypper --non-interactive install "$ACTUAL_PKG" ;;
    esac
    
    if [ $? -eq 0 ]; then
        if [ "$HAS_GUM" -eq 1 ]; then
            gum style --foreground 82 --bold "✓ $PKG installed successfully"
        else
            printf "${GREEN}✓ $PKG installed successfully${NC}\n"
        fi
    else
        printf "${RED}✗ Failed to install $PKG${NC}\n"
        return 1
    fi
}

# Uninstall
do_uninstall() {
    PKG="$1"
    if [ -z "$PKG" ]; then
        printf "${RED}Error: No package specified${NC}\n"
        return 1
    fi
    
    PM=$(detect_pkg_manager)
    ACTUAL_PKG=$(get_pkg_name "$PKG" "$PM")
    
    printf "${CYAN}Uninstalling $PKG...${NC}\n"
    case "$PM" in
        apt) apt-get remove -y "$ACTUAL_PKG" ;;
        apk) apk del "$ACTUAL_PKG" ;;
        dnf) dnf remove -y "$ACTUAL_PKG" ;;
        yum) yum remove -y "$ACTUAL_PKG" ;;
        pacman) pacman -R --noconfirm "$ACTUAL_PKG" ;;
        zypper) zypper --non-interactive remove "$ACTUAL_PKG" ;;
    esac
}

# Search
do_search() {
    TERM="$1"
    if [ -z "$TERM" ] && [ "$HAS_GUM" -eq 1 ]; then
        TERM=$(gum input --placeholder "Search for packages...")
    fi
    
    if [ -z "$TERM" ]; then
        printf "${RED}Error: No search term specified${NC}\n"
        return 1
    fi

    PM=$(detect_pkg_manager)
    printf "${CYAN}Searching for '$TERM'...${NC}\n"
    case "$PM" in
        apt) apt-cache search "$TERM" | head -20 ;;
        apk) apk search "$TERM" | head -20 ;;
        dnf) dnf search "$TERM" 2>/dev/null | head -20 ;;
        yum) yum search "$TERM" 2>/dev/null | head -20 ;;
        pacman) pacman -Ss "$TERM" | head -20 ;;
        *) echo "Search not supported on this OS" ;;
    esac
}

# Show Tools
show_tools() {
    if [ "$HAS_GUM" -eq 1 ]; then
        gum style --border normal --padding "0 2" --foreground 212 "Installed Tools"
    else
        printf "${CYAN}=== Installed Tools ===${NC}\n"
    fi
    
    # System
    echo ""
    printf "${YELLOW}System:${NC}\n"
    if [ -f /tmp/.rexec_installing_system ]; then
        if [ "$HAS_GUM" -eq 1 ]; then
            gum style --foreground 243 "  (Installing system packages...)"
        else
            printf "\033[38;5;243m  (Installing system packages...)\033[0m\n"
        fi
    fi
    for cmd in zsh git curl wget vim nano htop jq tmux fzf ripgrep neofetch; do
        if command -v $cmd >/dev/null 2>&1; then printf "  ${GREEN}✓${NC} $cmd\n"; fi
    done
    
    # AI
    echo ""
    printf "${YELLOW}AI & Dev:${NC}\n"
    if [ -f /tmp/.rexec_installing_ai ]; then
        if [ "$HAS_GUM" -eq 1 ]; then
            gum style --foreground 243 "  (Installing AI tools...)"
        else
            printf "\033[38;5;243m  (Installing AI tools...)\033[0m\n"
        fi
    fi
    for cmd in python3 node go rustc docker kubectl tgpt aichat mods gum aider opencode llm; do
        if command -v $cmd >/dev/null 2>&1; then printf "  ${GREEN}✓${NC} $cmd\n"; fi
    done
    echo ""
}

# Interactive Menu
show_menu() {
    if [ "$HAS_GUM" -eq 1 ]; then
        gum style \
            --border double --border-foreground 212 --padding "1 2" --margin "1 0" \
            --align center "Rexec CLI" "v$VERSION"

        CHOICE=$(gum choose \
            "🛠️  List Tools" \
            "📦  Install Package" \
            "🔍  Search Packages" \
            "ℹ️  System Info" \
            "🤖  AI Help" \
            "🚪  Exit")
        
        case "$CHOICE" in
            "🛠️  List Tools") show_tools ;;
            "📦  Install Package") do_install ;;
            "🔍  Search Packages") do_search ;;
            "ℹ️  System Info") 
                if command -v neofetch >/dev/null 2>&1; then neofetch; else 
                    echo "OS: $(grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d'"' -f2)"
                    echo "Kernel: $(uname -r)"
                fi 
                ;;
            "🤖  AI Help") cat "$HOME/.local/bin/ai-help" 2>/dev/null || echo "AI help not found" ;;
            "🚪  Exit") exit 0 ;;
        esac
    else
        show_help
    fi
}

show_help() {
    printf "${CYAN}Rexec CLI v${VERSION}${NC}\n"
    echo "Usage: rexec [command]"
    echo ""
    echo "Commands:"
    echo "  install <pkg>   Install package"
    echo "  search <term>   Search packages"
    echo "  tools           List installed tools"
    echo "  info            System info"
    echo "  help            Show this help"
    echo ""
    echo "Run 'rexec' without arguments for interactive menu."
}

# Main
CMD="$1"
if [ $# -gt 0 ]; then
    shift
fi

case "$CMD" in
    install|i) do_install "$@" ;;
    uninstall|rm) do_uninstall "$@" ;;
    search|s) do_search "$@" ;;
    tools|ls) show_tools ;;
    info) 
        if command -v neofetch >/dev/null 2>&1; then neofetch
        else echo "Host: $(hostname)"; fi 
        ;;
    help|--help|-h) show_help ;;
    "") show_menu ;;
    *) echo "Unknown command: $CMD"; show_help ;;
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
    touch /tmp/.rexec_installing_ai
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
    
    # gum - Glamorous shell scripts
    # https://github.com/charmbracelet/gum
    echo "  Installing gum (interactive shell UX)..."
    if [ -n "$MODS_ARCH" ]; then
        GUM_VERSION=$(curl -s https://api.github.com/repos/charmbracelet/gum/releases/latest 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
        [ -z "$GUM_VERSION" ] && GUM_VERSION="0.14.3"
        GUM_URL="https://github.com/charmbracelet/gum/releases/download/v${GUM_VERSION}/gum_${GUM_VERSION}_${MODS_ARCH}.tar.gz"
        curl -fsSL "$GUM_URL" 2>/dev/null | tar -xzf - -C "$HOME/.local/bin" gum 2>/dev/null && \
            chmod +x "$HOME/.local/bin/gum" && echo "    ✓ gum installed" || echo "    ! gum install failed"
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
    rm -f /tmp/.rexec_installing_ai
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
