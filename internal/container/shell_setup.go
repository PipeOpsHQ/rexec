package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ShellSetupScript contains the script to install and configure zsh with oh-my-zsh
const ShellSetupScript = `#!/bin/sh
set -e

# Detect package manager and install zsh + dependencies
install_packages() {
    if command -v apt-get >/dev/null 2>&1; then
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -qq
        # Reinstall git with proper dependencies to fix libpcre2 version issues
        apt-get install -y -qq --reinstall zsh git libpcre2-8-0 curl wget locales >/dev/null 2>&1
        # Generate locale
        if [ -f /etc/locale.gen ]; then
            sed -i 's/# en_US.UTF-8/en_US.UTF-8/' /etc/locale.gen
            locale-gen >/dev/null 2>&1 || true
        fi
    elif command -v apk >/dev/null 2>&1; then
        apk add --no-cache zsh git pcre2 curl wget shadow >/dev/null 2>&1
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y -q zsh git pcre2 curl wget >/dev/null 2>&1
    elif command -v yum >/dev/null 2>&1; then
        yum install -y -q zsh git pcre2 curl wget >/dev/null 2>&1
    elif command -v pacman >/dev/null 2>&1; then
        pacman -Sy --noconfirm zsh git pcre2 curl wget >/dev/null 2>&1
    elif command -v zypper >/dev/null 2>&1; then
        zypper install -y -q zsh git libpcre2-8-0 curl wget >/dev/null 2>&1
    else
        echo "Unsupported package manager"
        exit 1
    fi
}

# Install oh-my-zsh
install_ohmyzsh() {
    export HOME="${HOME:-/root}"
    export ZSH="$HOME/.oh-my-zsh"

    if [ ! -d "$ZSH" ]; then
        git clone --depth=1 https://github.com/ohmyzsh/ohmyzsh.git "$ZSH" 2>/dev/null
    fi
}

# Install zsh plugins
install_plugins() {
    export HOME="${HOME:-/root}"
    export ZSH="$HOME/.oh-my-zsh"
    ZSH_CUSTOM="$ZSH/custom"

    # zsh-autosuggestions
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-autosuggestions" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-autosuggestions "$ZSH_CUSTOM/plugins/zsh-autosuggestions" 2>/dev/null
    fi

    # zsh-syntax-highlighting
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-syntax-highlighting "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" 2>/dev/null
    fi

    # zsh-completions
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-completions" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-completions "$ZSH_CUSTOM/plugins/zsh-completions" 2>/dev/null
    fi

    # zsh-history-substring-search
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-history-substring-search" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-history-substring-search "$ZSH_CUSTOM/plugins/zsh-history-substring-search" 2>/dev/null
    fi
}

# Create zshrc configuration
create_zshrc() {
    export HOME="${HOME:-/root}"
    cat > "$HOME/.zshrc" << 'ZSHRC'
# Rexec Terminal Configuration
export ZSH="$HOME/.oh-my-zsh"
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
export TERM=xterm-256color

# Theme - using a simple but nice theme
ZSH_THEME="rexec"

# Plugins
plugins=(
    git
    zsh-autosuggestions
    zsh-syntax-highlighting
    zsh-completions
    zsh-history-substring-search
    command-not-found
    colored-man-pages
    extract
    sudo
)

# Plugin settings
ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="fg=#666666"
ZSH_AUTOSUGGEST_STRATEGY=(history completion)
ZSH_AUTOSUGGEST_BUFFER_MAX_SIZE=20

# History settings
HISTSIZE=10000
SAVEHIST=10000
HISTFILE=~/.zsh_history
setopt HIST_IGNORE_ALL_DUPS
setopt HIST_FIND_NO_DUPS
setopt HIST_SAVE_NO_DUPS
setopt SHARE_HISTORY
setopt APPEND_HISTORY
setopt INC_APPEND_HISTORY
setopt PROMPT_SUBST # Enable command substitution in prompt

# Completion settings
autoload -Uz compinit
zstyle ':completion:*' menu select
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}'
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"
zstyle ':completion:*' group-name ''
zstyle ':completion:*:descriptions' format '%F{yellow}-- %d --%f'
zstyle ':completion:*:warnings' format '%F{red}-- no matches found --%f'

# Key bindings
bindkey '^[[A' history-substring-search-up
bindkey '^[[B' history-substring-search-down
bindkey '^[OA' history-substring-search-up
bindkey '^[OB' history-substring-search-down
bindkey '^ ' autosuggest-accept
bindkey '^[[Z' reverse-menu-complete

# Load oh-my-zsh
source $ZSH/oh-my-zsh.sh

# Aliases
alias ll='ls -alF --color=auto'
alias la='ls -A --color=auto'
alias l='ls -CF --color=auto'
alias ls='ls --color=auto'
alias grep='grep --color=auto'
alias ..='cd ..'
alias ...='cd ../..'
alias cls='clear'
alias h='history'
alias hg='history | grep'
alias ports='netstat -tulanp'
alias df='df -h'
alias du='du -h'
alias free='free -h'
alias myip='curl -s ifconfig.me'

# Git aliases
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gl='git pull'
alias gd='git diff'
alias gco='git checkout'
alias gb='git branch'
alias glog='git log --oneline --graph --decorate'

# Docker aliases (if docker is available)
alias d='docker'
alias dc='docker-compose'
alias dps='docker ps'
alias dpsa='docker ps -a'

# Welcome message with system stats (like DigitalOcean)
# Shows container-specific resources, not host resources
show_system_stats() {
    # Get container info (hide kernel to prevent host info leak)
    local hostname=$(hostname)
    local os_name="Linux"
    # Try to get container OS info instead of kernel
    if [ -f /etc/os-release ]; then
        os_name=$(grep -E "^PRETTY_NAME=" /etc/os-release | cut -d'"' -f2 | head -1)
    fi
    local uptime_raw=$(cat /proc/uptime 2>/dev/null | cut -d. -f1)
    local uptime_days=$((uptime_raw / 86400))
    local uptime_hours=$(((uptime_raw % 86400) / 3600))
    local uptime_mins=$(((uptime_raw % 3600) / 60))
    
    # Container Memory info from cgroups (shows container limits, not host)
    local mem_limit_bytes=0
    local mem_used_bytes=0
    # Try cgroup v2 first
    if [ -f /sys/fs/cgroup/memory.max ]; then
        mem_limit_bytes=$(cat /sys/fs/cgroup/memory.max 2>/dev/null)
        mem_used_bytes=$(cat /sys/fs/cgroup/memory.current 2>/dev/null || echo "0")
    # Fall back to cgroup v1
    elif [ -f /sys/fs/cgroup/memory/memory.limit_in_bytes ]; then
        mem_limit_bytes=$(cat /sys/fs/cgroup/memory/memory.limit_in_bytes 2>/dev/null)
        mem_used_bytes=$(cat /sys/fs/cgroup/memory/memory.usage_in_bytes 2>/dev/null || echo "0")
    fi
    
    # Convert to MB and handle "max" value (unlimited)
    local mem_total_mb=512
    local mem_used_mb=0
    if [ "$mem_limit_bytes" = "max" ] || [ "$mem_limit_bytes" -gt 17179869184 ] 2>/dev/null; then
        # If unlimited or >16GB, use container default
        mem_total_mb=512
    elif [ -n "$mem_limit_bytes" ] && [ "$mem_limit_bytes" -gt 0 ] 2>/dev/null; then
        mem_total_mb=$((mem_limit_bytes / 1024 / 1024))
    fi
    
    if [ -n "$mem_used_bytes" ] && [ "$mem_used_bytes" -gt 0 ] 2>/dev/null; then
        mem_used_mb=$((mem_used_bytes / 1024 / 1024))
    fi
    
    local mem_percent=0
    if [ "$mem_total_mb" -gt 0 ] 2>/dev/null; then
        mem_percent=$((mem_used_mb * 100 / mem_total_mb))
    fi
    
    # Container CPU info from cgroups
    local cpu_quota=0
    local cpu_period=100000
    # Try cgroup v2 first
    if [ -f /sys/fs/cgroup/cpu.max ]; then
        local cpu_max=$(cat /sys/fs/cgroup/cpu.max 2>/dev/null)
        cpu_quota=$(echo "$cpu_max" | awk '{print $1}')
        cpu_period=$(echo "$cpu_max" | awk '{print $2}')
        [ "$cpu_quota" = "max" ] && cpu_quota=0
    # Fall back to cgroup v1
    elif [ -f /sys/fs/cgroup/cpu/cpu.cfs_quota_us ]; then
        cpu_quota=$(cat /sys/fs/cgroup/cpu/cpu.cfs_quota_us 2>/dev/null || echo "-1")
        cpu_period=$(cat /sys/fs/cgroup/cpu/cpu.cfs_period_us 2>/dev/null || echo "100000")
    fi
    
    # Calculate CPU cores allocated to container
    # Prefer environment variable if set, otherwise calculate from cgroup
    local cpu_cores="${REXEC_CPU_LIMIT:-0.5}"
    if [ -z "$REXEC_CPU_LIMIT" ] && [ "$cpu_quota" -gt 0 ] 2>/dev/null && [ "$cpu_period" -gt 0 ] 2>/dev/null; then
        # cpu_cores = quota / period (e.g., 50000/100000 = 0.5 cores)
        cpu_cores=$(awk "BEGIN {printf \"%.1f\", $cpu_quota / $cpu_period}")
    fi
    
    # Container Disk info - use allocated quota from environment
    local disk_quota="${REXEC_DISK_QUOTA:-2G}"
    # Get actual disk usage of root filesystem
    local disk_used=$(df -h / 2>/dev/null | awk 'NR==2 {print $3}' || echo "N/A")
    
    # Memory limit - prefer environment variable, clean up format
    local mem_limit="${REXEC_MEMORY_LIMIT:-${mem_total_mb}M}"
    # Remove decimal from memory limit if present (e.g., 1024.00M -> 1024M)
    mem_limit=$(echo "$mem_limit" | sed 's/\.00//')
    
    # Print banner
    echo ""
    echo "\033[38;5;105m  ╭─────────────────────────────────────────────────────────╮\033[0m"
    echo "\033[38;5;105m  │\033[0m           \033[1;36mWelcome to Rexec Terminal\033[0m                   \033[38;5;105m│\033[0m"
    echo "\033[38;5;105m  ╰─────────────────────────────────────────────────────────╯\033[0m"
    echo ""
    echo "\033[1;33m  Container:\033[0m"
    echo "\033[38;5;243m  ├─ ID:\033[0m          ${hostname:0:12}"
    echo "\033[38;5;243m  ├─ OS:\033[0m          $os_name"
    echo "\033[38;5;243m  └─ Uptime:\033[0m      ${uptime_days}d ${uptime_hours}h ${uptime_mins}m"
    echo ""
    echo "\033[1;33m  Resources (Allocated):\033[0m"
    echo "\033[38;5;243m  ├─ CPU:\033[0m         ${cpu_cores} vCPU"
    echo "\033[38;5;243m  ├─ Memory:\033[0m      ${mem_used_mb}MB / ${mem_limit}"
    echo "\033[38;5;243m  └─ Storage:\033[0m     ${disk_used} / ${disk_quota}"
    echo ""
    echo "\033[38;5;243m  Type '\033[1;37mhelp\033[38;5;243m' for common commands\033[0m"
    echo ""
}

# Run stats on shell start
show_system_stats
ZSHRC
}

# Create custom theme
create_theme() {
    export HOME="${HOME:-/root}"
    mkdir -p "$HOME/.oh-my-zsh/custom/themes"
    cat > "$HOME/.oh-my-zsh/custom/themes/rexec.zsh-theme" << 'THEME'
# Rexec Terminal Theme

# Git prompt settings - must be defined before use
ZSH_THEME_GIT_PROMPT_PREFIX="%F{magenta}git:(%F{green}"
ZSH_THEME_GIT_PROMPT_SUFFIX="%f "
ZSH_THEME_GIT_PROMPT_DIRTY="%F{magenta}) %F{red}✗"
ZSH_THEME_GIT_PROMPT_CLEAN="%F{magenta}) %F{green}✓"

# Main prompt using direct function call
PROMPT='
%F{cyan}%n%f@%F{blue}%m%f %F{yellow}%~%f $(git_prompt_info)
%(?:%F{green}➜:%F{red}➜) %f'

# Right prompt - show time
RPROMPT='%F{240}%*%f'
THEME
}

# Create help command
create_help() {
    export HOME="${HOME:-/root}"
    cat > "$HOME/.local/bin/help" << 'HELP'
#!/bin/sh
echo ""
echo "\033[1;36m=== Rexec Terminal Help ===\033[0m"
echo ""
echo "\033[1;33mNavigation:\033[0m"
echo "  ll, la, l    - List files (different formats)"
echo "  ..           - Go up one directory"
echo "  ...          - Go up two directories"
echo ""
echo "\033[1;33mHistory:\033[0m"
echo "  ↑/↓          - Search history (substring match)"
echo "  Ctrl+R       - Reverse search history"
echo "  h            - Show history"
echo "  hg <term>    - Search history for term"
echo ""
echo "\033[1;33mAutosuggestions:\033[0m"
echo "  →            - Accept suggestion"
echo "  Ctrl+Space   - Accept suggestion"
echo "  Tab          - Autocomplete"
echo ""
echo "\033[1;33mGit Shortcuts:\033[0m"
echo "  gs           - git status"
echo "  ga           - git add"
echo "  gc           - git commit"
echo "  gp           - git push"
echo "  gl           - git pull"
echo "  glog         - git log (graph)"
echo ""
echo "\033[1;33mSystem:\033[0m"
echo "  myip         - Show public IP"
echo "  ports        - Show open ports"
echo "  df           - Disk usage"
echo "  free         - Memory usage"
echo ""
echo "\033[1;33mSSH Access:\033[0m"
echo "  Your SSH keys are synced automatically."
echo "  Connect via: ssh root@<host> -p <port>"
echo ""
HELP
    chmod +x "$HOME/.local/bin/help"
}

# Set zsh as default shell
set_default_shell() {
    ZSH_PATH=$(which zsh)
    if [ -n "$ZSH_PATH" ]; then
        # Add to /etc/shells if not present
        if ! grep -q "$ZSH_PATH" /etc/shells 2>/dev/null; then
            echo "$ZSH_PATH" >> /etc/shells
        fi
        # Change shell for root
        if command -v chsh >/dev/null 2>&1; then
            chsh -s "$ZSH_PATH" root 2>/dev/null || true
        fi
        # Also update /etc/passwd directly as fallback
        if [ -f /etc/passwd ]; then
            sed -i "s|root:.*:/bin/.*|root:x:0:0:root:/root:$ZSH_PATH|" /etc/passwd 2>/dev/null || true
        fi
    fi
}

# Create necessary directories
setup_dirs() {
    export HOME="${HOME:-/root}"
    mkdir -p "$HOME/.local/bin"
    mkdir -p "$HOME/.cache"

    # Add local bin to path in profile
    if ! grep -q '.local/bin' "$HOME/.profile" 2>/dev/null; then
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.profile"
    fi
}

# Main setup
main() {
    echo "Setting up enhanced shell environment..."

    setup_dirs
    echo "  [1/7] Installing packages..."
    install_packages

    echo "  [2/7] Installing oh-my-zsh..."
    install_ohmyzsh

    echo "  [3/7] Installing plugins..."
    install_plugins

    echo "  [4/7] Creating configuration..."
    create_zshrc

    echo "  [5/7] Creating theme..."
    create_theme

    echo "  [6/7] Creating help command..."
    create_help

    echo "  [7/7] Setting default shell..."
    set_default_shell

    echo "Shell setup complete!"
}

main
`

// SetupShellResponse contains the result of shell setup
type SetupShellResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

// SetupEnhancedShell installs and configures zsh with oh-my-zsh in a container
func SetupEnhancedShell(ctx context.Context, cli *client.Client, containerID string) (*SetupShellResponse, error) {
	// Create exec configuration
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", ShellSetupScript},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	// Start exec
	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
		Tty: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	// Read output
	var output strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := attachResp.Reader.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	// Check exit code
	inspect, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	if inspect.ExitCode != 0 {
		return &SetupShellResponse{
			Success: false,
			Message: "Shell setup failed",
			Output:  output.String(),
		}, nil
	}

	return &SetupShellResponse{
		Success: true,
		Message: "Shell setup complete",
		Output:  output.String(),
	}, nil
}

// IsShellSetupComplete checks if the enhanced shell is already set up
func IsShellSetupComplete(ctx context.Context, cli *client.Client, containerID string) bool {
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", "test -d ~/.oh-my-zsh && test -f ~/.zshrc"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return false
	}

	if err := cli.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{}); err != nil {
		return false
	}

	inspect, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return false
	}

	return inspect.ExitCode == 0
}

// GetContainerShell returns the best available shell in the container
func GetContainerShell(ctx context.Context, cli *client.Client, containerID string) string {
	// Check for zsh first
	shells := []string{"/bin/zsh", "/usr/bin/zsh", "/bin/bash", "/usr/bin/bash", "/bin/sh"}

	for _, shell := range shells {
		execConfig := container.ExecOptions{
			Cmd:          []string{"/bin/sh", "-c", fmt.Sprintf("test -x %s", shell)},
			AttachStdout: true,
			AttachStderr: true,
		}

		execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			continue
		}

		if err := cli.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{}); err != nil {
			continue
		}

		inspect, err := cli.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			continue
		}

		if inspect.ExitCode == 0 {
			return shell
		}
	}

	return "/bin/sh"
}

// SetupRole installs tools for a specific role
func SetupRole(ctx context.Context, cli *client.Client, containerID string, roleID string) (*SetupShellResponse, error) {
	script, err := GenerateRoleScript(roleID)
	if err != nil {
		return nil, err
	}

	// Create exec configuration
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", script},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	// Start exec
	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
		Tty: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	// Read output
	var output strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := attachResp.Reader.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	// Check exit code
	inspect, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	if inspect.ExitCode != 0 {
		return &SetupShellResponse{
			Success: false,
			Message: fmt.Sprintf("Role setup failed for %s", roleID),
			Output:  output.String(),
		}, nil
	}

	return &SetupShellResponse{
		Success: true,
		Message: fmt.Sprintf("Role setup complete for %s", roleID),
		Output:  output.String(),
	}, nil
}
