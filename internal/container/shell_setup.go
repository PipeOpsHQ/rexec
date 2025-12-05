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
		Theme:           "rexec",
		Autosuggestions: false,
		SyntaxHighlight: false,
		HistorySearch:   true,
		GitAliases:      false,
		SystemStats:     true,
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

	// System stats section (conditional) - includes both the function and the call
	systemStats := ""
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
        apt-get update -qq
        apt-get install -y -qq --reinstall zsh git libpcre2-8-0 curl wget locales >/dev/null 2>&1
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
        pacman-key --init 2>/dev/null || true
        pacman-key --populate archlinux 2>/dev/null || true
        pacman -Sy --noconfirm --needed zsh git pcre2 curl wget >/dev/null 2>&1
    elif command -v zypper >/dev/null 2>&1; then
        zypper --non-interactive refresh >/dev/null 2>&1 || true
        zypper --non-interactive install -y zsh git libpcre2-8-0 curl wget >/dev/null 2>&1
    else
        echo "Unsupported package manager"
        exit 1
    fi
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

source $ZSH/oh-my-zsh.sh

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
ZSHRC
}

create_theme() {
    export HOME="${HOME:-/root}"
    mkdir -p "$HOME/.oh-my-zsh/custom/themes"
    cat > "$HOME/.oh-my-zsh/custom/themes/rexec.zsh-theme" << 'THEME'
ZSH_THEME_GIT_PROMPT_PREFIX="%%F{magenta}git:(%%F{green}"
ZSH_THEME_GIT_PROMPT_SUFFIX="%%f "
ZSH_THEME_GIT_PROMPT_DIRTY="%%F{magenta}) %%F{red}*"
ZSH_THEME_GIT_PROMPT_CLEAN="%%F{magenta}) %%F{green}ok"

PROMPT='%%F{cyan}%%n%%f@%%F{blue}%%m%%f %%F{yellow}%%~%%f$(git_prompt_info) %%F{green}$%%f '
RPROMPT=''
THEME

    # Minimal theme (simpler, faster)
    cat > "$HOME/.oh-my-zsh/custom/themes/minimal.zsh-theme" << 'THEME'
PROMPT='%%F{cyan}%%n%%f:%%F{yellow}%%~%%f %%F{green}$%%f '
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
    echo "  [1/6] Installing packages..."
    install_packages
    echo "  [2/6] Installing oh-my-zsh..."
    install_ohmyzsh
    echo "  [3/6] Installing plugins..."
    install_plugins
    echo "  [4/6] Creating configuration..."
    create_zshrc
    echo "  [5/6] Creating theme..."
    create_theme
    echo "  [6/6] Setting default shell..."
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
