package container

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ShellSetupTimeout is the maximum time allowed for shell setup
const ShellSetupTimeout = 2 * time.Minute

// SetupShellResponse contains the result of shell setup
type SetupShellResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

// ShellSetupConfig defines configurable shell options
type ShellSetupConfig struct {
	Enhanced        bool   // Install oh-my-zsh + plugins
	Theme           string // zsh theme: "rexec", "minimal", "powerlevel10k"
	Autosuggestions bool   // Enable zsh-autosuggestions
	SyntaxHighlight bool   // Enable zsh-syntax-highlighting
	HistorySearch   bool   // Enable history-substring-search
	GitAliases      bool   // Enable git shortcuts
	SystemStats     bool   // Show system stats on login
}

// DefaultShellSetupConfig returns the default (full) shell configuration
func DefaultShellSetupConfig() ShellSetupConfig {
	return ShellSetupConfig{
		Enhanced:        true,
		Theme:           "robbyrussell",
		Autosuggestions: true,
		SyntaxHighlight: true,
		HistorySearch:   true,
		GitAliases:      true,
		SystemStats:     false,
	}
}

// generateShellSetupScript generates a customized shell setup script based on config
func generateShellSetupScript(cfg ShellSetupConfig) string {
	if !cfg.Enhanced {
		// Minimal setup - just ensure basic shell works
		return `#!/bin/sh
echo "Minimal shell mode - no enhanced features installed"
`
	}

	// Build plugins list based on config
	plugins := []string{"git", "zsh-completions", "command-not-found", "colored-man-pages", "extract", "sudo"}
	if cfg.Autosuggestions {
		plugins = append(plugins, "zsh-autosuggestions")
	}
	if cfg.SyntaxHighlight {
		plugins = append(plugins, "zsh-syntax-highlighting")
	}
	if cfg.HistorySearch {
		plugins = append(plugins, "zsh-history-substring-search")
	}
	pluginsStr := strings.Join(plugins, "\n    ")

	// Theme selection
	theme := cfg.Theme
	if theme == "" {
		theme = "rexec"
	}

	// Git aliases section (conditional)
	gitAliases := ""
	if cfg.GitAliases {
		gitAliases = `
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
`
	}

	// Welcome message - always show rexec CLI info on first login
	welcomeMessage := `
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
`

	// System stats section (conditional) - includes both the function and the call
	systemStats := welcomeMessage
	if cfg.SystemStats {
		systemStats = `
# Welcome message with system stats
show_system_stats() {
    local container_id=""
    if command -v hostname >/dev/null 2>&1; then
        container_id=$(hostname 2>/dev/null)
    fi
    [ -z "$container_id" ] && [ -f /etc/hostname ] && container_id=$(cat /etc/hostname 2>/dev/null)
    [ -z "$container_id" ] && container_id="${HOSTNAME:-unknown}"

    local os_name="Linux"
    if [ -f /etc/os-release ]; then
        os_name=$(grep -E "^PRETTY_NAME=" /etc/os-release 2>/dev/null | cut -d'"' -f2 | head -1)
        [ -z "$os_name" ] && os_name=$(grep -E "^NAME=" /etc/os-release 2>/dev/null | cut -d'"' -f2 | head -1)
    fi
    [ -z "$os_name" ] && os_name="Linux"

    local uptime_raw=$(cat /proc/uptime 2>/dev/null | cut -d. -f1)
    [ -z "$uptime_raw" ] && uptime_raw=0
    local uptime_days=$((uptime_raw / 86400))
    local uptime_hours=$(((uptime_raw % 86400) / 3600))
    local uptime_mins=$(((uptime_raw % 3600) / 60))

    local mem_limit_bytes=0 mem_used_bytes=0
    if [ -f /sys/fs/cgroup/memory.max ]; then
        mem_limit_bytes=$(cat /sys/fs/cgroup/memory.max 2>/dev/null)
        mem_used_bytes=$(cat /sys/fs/cgroup/memory.current 2>/dev/null || echo "0")
    elif [ -f /sys/fs/cgroup/memory/memory.limit_in_bytes ]; then
        mem_limit_bytes=$(cat /sys/fs/cgroup/memory/memory.limit_in_bytes 2>/dev/null)
        mem_used_bytes=$(cat /sys/fs/cgroup/memory/memory.usage_in_bytes 2>/dev/null || echo "0")
    fi

    local mem_total_mb=512 mem_used_mb=0
    if [ "$mem_limit_bytes" != "max" ] && [ "$mem_limit_bytes" -gt 0 ] 2>/dev/null && [ "$mem_limit_bytes" -lt 17179869184 ]; then
        mem_total_mb=$((mem_limit_bytes / 1024 / 1024))
    fi
    [ "$mem_used_bytes" -gt 0 ] 2>/dev/null && mem_used_mb=$((mem_used_bytes / 1024 / 1024))

    local cpu_cores="0.5"
    if [ -f /sys/fs/cgroup/cpu.max ]; then
        local cpu_max=$(cat /sys/fs/cgroup/cpu.max 2>/dev/null)
        local cpu_quota=$(echo "$cpu_max" | awk '{print $1}')
        local cpu_period=$(echo "$cpu_max" | awk '{print $2}')
        [ "$cpu_quota" != "max" ] && [ "$cpu_quota" -gt 0 ] 2>/dev/null && cpu_cores=$(awk "BEGIN {printf \"%.1f\", $cpu_quota / $cpu_period}")
    fi

    local disk_quota="${REXEC_DISK_QUOTA:-2G}"
    [ -f /etc/rexec/config ] && disk_quota=$(grep '^DISK=' /etc/rexec/config 2>/dev/null | cut -d= -f2 || echo "$disk_quota")

    echo ""
    echo "\033[38;5;105m  ╭───────────────────────────────────────╮\033[0m"
    echo "\033[38;5;105m  │\033[0m    \033[1;36mWelcome to Rexec Terminal\033[0m          \033[38;5;105m│\033[0m"
    echo "\033[38;5;105m  ╰───────────────────────────────────────╯\033[0m"
    echo ""
    echo "\033[1;33m  Container:\033[0m"
    echo "\033[38;5;243m  ├─ ID:\033[0m    ${container_id:0:12}"
    echo "\033[38;5;243m  ├─ OS:\033[0m    $os_name"
    echo "\033[38;5;243m  └─ Up:\033[0m    ${uptime_days}d ${uptime_hours}h ${uptime_mins}m"
    echo ""
    echo "\033[1;33m  Resources:\033[0m"
    echo "\033[38;5;243m  ├─ CPU:\033[0m   ${cpu_cores} vCPU"
    echo "\033[38;5;243m  ├─ Mem:\033[0m   ${mem_used_mb}MB / ${mem_total_mb}MB"
    echo "\033[38;5;243m  └─ Disk:\033[0m  ${disk_quota}"
    echo ""
    echo "\033[38;5;243m  Type '\033[1;37mhelp\033[0m\033[38;5;243m' for commands\033[0m"
    echo ""
}
show_system_stats
`
	}

	// Generate the full script with conditional sections
	return fmt.Sprintf(`#!/bin/sh
set -e

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

# Detect package manager and install zsh + dependencies
install_packages() {
    if command -v apt-get >/dev/null 2>&1; then
        export DEBIAN_FRONTEND=noninteractive
        wait_for_apt_lock || true
        apt-get update -qq || true
        # Install zsh and locale support. language-pack-en is needed for Ubuntu.
        apt-get install -y -qq zsh git curl wget locales language-pack-en >/dev/null 2>&1 || apt-get install -y -qq zsh git curl wget locales >/dev/null 2>&1
        
        # Ensure locale is generated
        if [ -f /etc/locale.gen ]; then
            sed -i 's/# en_US.UTF-8/en_US.UTF-8/' /etc/locale.gen
        fi
        locale-gen en_US.UTF-8 >/dev/null 2>&1 || update-locale LANG=en_US.UTF-8 >/dev/null 2>&1 || true
    elif command -v apk >/dev/null 2>&1; then
        apk add --no-cache zsh git pcre2 curl wget shadow >/dev/null 2>&1
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y -q zsh git pcre2 curl wget glibc-langpack-en >/dev/null 2>&1 || dnf install -y -q zsh git pcre2 curl wget >/dev/null 2>&1
    elif command -v yum >/dev/null 2>&1; then
        yum install -y -q zsh git pcre2 curl wget glibc-langpack-en >/dev/null 2>&1 || yum install -y -q zsh git pcre2 curl wget >/dev/null 2>&1
    elif command -v urpmi >/dev/null 2>&1; then
        # Mageia
        urpmi --auto --no-recommends zsh git libpcre2-8-0 curl wget locales-en >/dev/null 2>&1
    elif command -v pacman >/dev/null 2>&1; then
        pacman-key --init 2>/dev/null || true
        pacman-key --populate archlinux 2>/dev/null || true
        pacman -Sy --noconfirm --needed zsh git pcre2 curl wget >/dev/null 2>&1
        # Generate locale for Arch
        if [ -f /etc/locale.gen ]; then
            sed -i 's/#en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
            locale-gen >/dev/null 2>&1 || true
        fi
    elif command -v zypper >/dev/null 2>&1; then
        zypper --non-interactive refresh >/dev/null 2>&1 || true
        zypper --non-interactive install -y zsh git libpcre2-8-0 curl wget glibc-locale >/dev/null 2>&1
    else
        echo "Unsupported package manager"
        exit 1
    fi
}

install_opencode() {
    # Install opencode AI coding assistant
    # https://github.com/sst/opencode
    export HOME="${HOME:-/root}"

    # Check if already installed
    if command -v opencode >/dev/null 2>&1; then
        return 0
    fi

    # Detect architecture
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
            echo "Unsupported architecture: $ARCH"
            return 1
            ;;
    esac

    # Get latest version
    OPENCODE_VERSION=$(curl -s https://api.github.com/repos/sst/opencode/releases/latest 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$OPENCODE_VERSION" ]; then
        OPENCODE_VERSION="v1.0.133"  # Fallback version
    fi

    # Download and install
    OPENCODE_URL="https://github.com/sst/opencode/releases/download/${OPENCODE_VERSION}/opencode-${OPENCODE_ARCH}.tar.gz"

    mkdir -p "$HOME/.local/bin"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$OPENCODE_URL" 2>/dev/null | tar -xzf - -C "$HOME/.local/bin" 2>/dev/null || true
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$OPENCODE_URL" 2>/dev/null | tar -xzf - -C "$HOME/.local/bin" 2>/dev/null || true
    fi

    # Make executable
    chmod +x "$HOME/.local/bin/opencode" 2>/dev/null || true
}

install_ohmyzsh() {
    export HOME="${HOME:-/root}"
    export ZSH="$HOME/.oh-my-zsh"
    if [ ! -d "$ZSH" ]; then
        git clone --depth=1 https://github.com/ohmyzsh/ohmyzsh.git "$ZSH" 2>/dev/null
    fi
}

install_plugins() {
    export HOME="${HOME:-/root}"
    export ZSH="$HOME/.oh-my-zsh"
    ZSH_CUSTOM="$ZSH/custom"
    %s
}

create_zshrc() {
    export HOME="${HOME:-/root}"
    cat > "$HOME/.zshrc" << 'ZSHRC'
export ZSH="$HOME/.oh-my-zsh"
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
export TERM=xterm-256color
export PATH="$HOME/.local/bin:$PATH"

ZSH_THEME="%s"

plugins=(
    %s
)

ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="fg=#666666"
ZSH_AUTOSUGGEST_STRATEGY=(history completion)
ZSH_AUTOSUGGEST_BUFFER_MAX_SIZE=20

HISTSIZE=10000
SAVEHIST=10000
HISTFILE=~/.zsh_history
setopt HIST_IGNORE_ALL_DUPS HIST_FIND_NO_DUPS HIST_SAVE_NO_DUPS
setopt SHARE_HISTORY APPEND_HISTORY INC_APPEND_HISTORY PROMPT_SUBST
unsetopt PROMPT_SP # Prevent partial line indicator (%%)

autoload -Uz compinit
zstyle ':completion:*' menu select
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}'
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"

bindkey '^[[A' history-substring-search-up
bindkey '^[[B' history-substring-search-down
bindkey '^[OA' history-substring-search-up
bindkey '^[OB' history-substring-search-down
bindkey '^ ' autosuggest-accept
bindkey '^[[Z' reverse-menu-complete

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
alias myip='curl -s ifconfig.me'
%s
%s

unset PS1 # Ensure themes can set their own PS1
source $ZSH/oh-my-zsh.sh

ZSHRC
}

create_theme() {
    export HOME="${HOME:-/root}"
    mkdir -p "$HOME/.oh-my-zsh/custom/themes"
    # Simple theme that works reliably - avoid complex escape sequences
    cat > "$HOME/.oh-my-zsh/custom/themes/rexec.zsh-theme" << 'THEME'
# Simple prompt: user@host dir $
PROMPT='%%n@%%m %%~ $ '
RPROMPT=''
THEME

    # Minimal theme (simpler, faster)
    cat > "$HOME/.oh-my-zsh/custom/themes/minimal.zsh-theme" << 'THEME'
PROMPT='%%n@%%m %%~ $ '
RPROMPT=''
THEME
}

create_help() {
    export HOME="${HOME:-/root}"
    mkdir -p "$HOME/.local/bin"
    cat > "$HOME/.local/bin/help" << 'HELP'
#!/bin/sh
echo "\033[1;36m=== Rexec Terminal Help ===\033[0m"
echo "↑/↓: History search | Tab: Autocomplete | Ctrl+Space: Accept suggestion"
echo "ll/la/l: List files | ..: Go up | myip: Show IP"
echo "gs/ga/gc/gp: Git shortcuts"
HELP
    chmod +x "$HOME/.local/bin/help"
}

set_default_shell() {
    ZSH_PATH=$(which zsh)
    if [ -n "$ZSH_PATH" ]; then
        grep -q "$ZSH_PATH" /etc/shells 2>/dev/null || echo "$ZSH_PATH" >> /etc/shells
        command -v chsh >/dev/null 2>&1 && chsh -s "$ZSH_PATH" root 2>/dev/null || true
        [ -f /etc/passwd ] && sed -i "s|root:.*:/bin/.*|root:x:0:0:root:/root:$ZSH_PATH|" /etc/passwd 2>/dev/null || true
    fi
}

setup_dirs() {
    export HOME="${HOME:-/root}"
    mkdir -p "$HOME/.local/bin" "$HOME/.cache"
    grep -q '.local/bin' "$HOME/.profile" 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.profile"
}

main() {
    echo "Setting up shell environment..."
    setup_dirs
    echo "  [1/7] Installing packages..."
    install_packages
    echo "  [2/7] Installing oh-my-zsh..."
    install_ohmyzsh
    echo "  [3/7] Installing plugins..."
    install_plugins
    echo "  [4/7] Installing opencode..."
    install_opencode
    echo "  [5/7] Creating configuration..."
    create_zshrc
    echo "  [6/7] Creating theme..."
    create_theme
    echo "  [7/7] Setting default shell..."
    create_help
    set_default_shell
    echo "Shell setup complete!"
}

main
`, generatePluginInstallScript(cfg), theme, pluginsStr, gitAliases, systemStats)
}

// generatePluginInstallScript generates the plugin installation part of the script
func generatePluginInstallScript(cfg ShellSetupConfig) string {
	var parts []string

	// Always install zsh-autosuggestions
	parts = append(parts, `
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-autosuggestions" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-autosuggestions "$ZSH_CUSTOM/plugins/zsh-autosuggestions" 2>/dev/null
    fi`)

	if cfg.SyntaxHighlight {
		parts = append(parts, `
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-syntax-highlighting "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" 2>/dev/null
    fi`)
	}

	if cfg.HistorySearch {
		parts = append(parts, `
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-history-substring-search" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-history-substring-search "$ZSH_CUSTOM/plugins/zsh-history-substring-search" 2>/dev/null
    fi`)
	}

	// Always install zsh-completions as it's lightweight
	parts = append(parts, `
    if [ ! -d "$ZSH_CUSTOM/plugins/zsh-completions" ]; then
        git clone --depth=1 https://github.com/zsh-users/zsh-completions "$ZSH_CUSTOM/plugins/zsh-completions" 2>/dev/null
    fi`)

	return strings.Join(parts, "")
}

// SetupEnhancedShell installs and configures zsh with oh-my-zsh in a container (default config)
func SetupEnhancedShell(ctx context.Context, cli *client.Client, containerID string) (*SetupShellResponse, error) {
	return SetupShellWithConfig(ctx, cli, containerID, DefaultShellSetupConfig())
}

// SetupShellWithConfig installs and configures zsh with oh-my-zsh using custom config
func SetupShellWithConfig(ctx context.Context, cli *client.Client, containerID string, cfg ShellSetupConfig) (*SetupShellResponse, error) {
	// If enhanced is disabled, skip shell setup entirely
	if !cfg.Enhanced {
		return &SetupShellResponse{
			Success: true,
			Message: "Minimal shell mode - no enhanced features installed",
		}, nil
	}

	// Apply timeout to prevent hanging on slow containers
	ctx, cancel := context.WithTimeout(ctx, ShellSetupTimeout)
	defer cancel()

	// Generate customized script
	script := generateShellSetupScript(cfg)

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

	// Use ContainerExecAttach instead of ContainerExecStart for Podman compatibility
	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return false
	}
	attachResp.Close()

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

		// Use ContainerExecAttach instead of ContainerExecStart for Podman compatibility
		attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
		if err != nil {
			continue
		}
		attachResp.Close()

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
