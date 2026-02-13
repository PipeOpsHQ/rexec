package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

const (
	Version     = "1.0.0"
	DefaultHost = "https://rexec.pipeops.io"
	ConfigDir   = ".rexec"
	AgentFile   = "agent.json"
)

// ANSI colors
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

type Config struct {
	Host     string `json:"host"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

type AgentConfig struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Host        string   `json:"host"`
	Token       string   `json:"token"`
	Shell       string   `json:"shell"`
	Tags        []string `json:"tags,omitempty"`
	Registered  bool     `json:"registered"`
	AutoStart   bool     `json:"auto_start"`
	AutoUpdate  bool     `json:"auto_update"`
}

// ShellSession represents a single shell/PTY session
type ShellSession struct {
	ID     string
	Cmd    *exec.Cmd
	Ptmx   *os.File
	IsMain bool // Main session shares tmux, split sessions get new windows
}

type Agent struct {
	config     *AgentConfig
	conn       *websocket.Conn
	sessions   map[string]*ShellSession // Multiple shell sessions for split panes
	mainPtmx   *os.File                 // Main PTY for backwards compatibility
	mainCmd    *exec.Cmd                // Main command for backwards compatibility
	mu         sync.Mutex
	running    bool
	reconnects int
}

var configPath string

func main() {
	// Parse global flags first
	args := os.Args[1:]
	var cmdArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--config" || args[i] == "-c" {
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		} else {
			cmdArgs = append(cmdArgs, args[i])
		}
	}

	if len(cmdArgs) < 1 {
		// If --config is provided without command, default to start
		if configPath != "" {
			handleStart([]string{})
			return
		}
		showHelp()
		return
	}

	switch cmdArgs[0] {
	case "help", "-h", "--help":
		showHelp()
	case "version", "-v", "--version":
		fmt.Printf("%srexec-agent%s v%s\n", Bold, Reset, Version)
	case "register":
		handleRegister(cmdArgs[1:])
	case "start":
		handleStart(cmdArgs[1:])
	case "stop":
		handleStop()
	case "status":
		handleStatus()
	case "unregister":
		handleUnregister()
	case "install":
		handleInstall()
	case "refresh-token":
		handleRefreshToken(cmdArgs[1:])
	case "prep-shell":
		handlePrepShell()
	default:
		fmt.Printf("%sUnknown command: %s%s\n", Red, cmdArgs[0], Reset)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf(`
%s%sRexec Agent%s - Connect your server to Rexec

%sUSAGE:%s
  rexec-agent [--config path] <command> [options]

%sGLOBAL OPTIONS:%s
  --config, -c     Path to config file (default: /etc/rexec/agent.yaml or ~/.rexec/agent.json)

%sCOMMANDS:%s
  register       Register this machine as a Rexec terminal
  start          Start the agent (connect to Rexec)
  stop           Stop the running agent
  status         Show agent status
  unregister     Remove this machine from Rexec
  install        Install as a system service
  refresh-token  Update token for existing agent (fixes 401 errors)
  prep-shell     Pre-install enhanced shell (zsh + oh-my-zsh) for faster first connect

%sREGISTER OPTIONS:%s
  --name, -n       Name for this terminal (default: hostname)
  --description    Description of this server
  --shell          Shell to use (default: $SHELL or /bin/bash)
  --tags           Comma-separated tags
  --token          Auth token (or set REXEC_TOKEN)
  --host           API host (default: %s)

%sEXAMPLES:%s
  rexec-agent register --name "prod-server-1" --tags "production,aws"
  rexec-agent start
  rexec-agent --config /etc/rexec/agent.yaml start
  rexec-agent status

	%sENVIRONMENT:%s
	  REXEC_TOKEN      Auth token
	  REXEC_HOST       API host
	  REXEC_API        API host (alternative)
	  REXEC_CONFIG     Config file path
	  REXEC_AUTO_UPDATE  Enable self-updates (true/false)

	`, Bold, Cyan, Reset,
		Yellow, Reset,
		Yellow, Reset,
		Yellow, Reset,
		Yellow, Reset,
		DefaultHost,
		Yellow, Reset,
		Yellow, Reset)
}

func getConfigPath() string {
	// If --config flag was provided, use that
	if configPath != "" {
		return configPath
	}

	// Check for REXEC_CONFIG env var
	if envConfig := os.Getenv("REXEC_CONFIG"); envConfig != "" {
		return envConfig
	}

	// Check /etc/rexec/agent.yaml first (system-wide)
	if _, err := os.Stat("/etc/rexec/agent.yaml"); err == nil {
		return "/etc/rexec/agent.yaml"
	}

	// Fall back to ~/.rexec/agent.json (user-specific)
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir, AgentFile)
}

func loadAgentConfig() (*AgentConfig, error) {
	cfgPath := getConfigPath()
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	var cfg AgentConfig

	// Check if it's YAML (from install script) or JSON (from register command)
	if strings.HasSuffix(cfgPath, ".yaml") || strings.HasSuffix(cfgPath, ".yml") {
		// Parse YAML config
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "api_url":
				cfg.Host = value
			case "token":
				cfg.Token = value
			case "agent_id":
				cfg.ID = value
			case "name":
				cfg.Name = value
			case "shell":
				cfg.Shell = value
			case "auto_update":
				cfg.AutoUpdate = parseBool(value)
			}
		}
		cfg.Registered = cfg.Token != "" && (cfg.ID != "" || cfg.Host != "")
	} else {
		// Parse JSON config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}

	// Override with env vars
	if host := os.Getenv("REXEC_HOST"); host != "" {
		cfg.Host = host
	}
	if host := os.Getenv("REXEC_API"); host != "" {
		cfg.Host = host
	}
	if token := os.Getenv("REXEC_TOKEN"); token != "" {
		cfg.Token = token
	}

	// Set defaults
	if cfg.Host == "" {
		cfg.Host = DefaultHost
	}
	if cfg.Shell == "" {
		cfg.Shell = os.Getenv("SHELL")
		if cfg.Shell == "" {
			cfg.Shell = "/bin/bash"
		}
	}
	if cfg.Name == "" {
		cfg.Name, _ = os.Hostname()
	}

	// Prefer a fully-featured interactive shell (zsh/bash) over plain sh when available.
	cfg.Shell = resolveInteractiveShell(cfg.Shell)

	return &cfg, nil
}

func parseBool(value string) bool {
	v := strings.TrimSpace(strings.ToLower(strings.Trim(value, `"'`)))
	return v == "true" || v == "1" || v == "yes" || v == "y" || v == "on"
}

func normalizeShellValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return strings.TrimSpace(value)
}

func isExecutable(path string) bool {
	if path == "" {
		return false
	}

	// If the user provided a bare command (e.g. "bash"), resolve from PATH.
	if !strings.Contains(path, "/") {
		_, err := exec.LookPath(path)
		return err == nil
	}

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func isShShell(path string) bool {
	return filepath.Base(path) == "sh"
}

// resolveInteractiveShell picks a stable interactive shell for agent sessions.
// If config points to /bin/sh, we prefer zsh/bash when installed to avoid broken
// arrow keys/history (sh lacks readline) and reduce startup script incompatibilities.
func resolveInteractiveShell(preferred string) string {
	preferred = normalizeShellValue(preferred)
	envShell := normalizeShellValue(os.Getenv("SHELL"))

	// Build ordered candidates with de-duplication.
	var candidates []string

	// If preferred is not plain "sh", honor it first.
	if preferred != "" && !isShShell(preferred) {
		candidates = append(candidates, preferred)
	}

	// Next, try the user's login shell if it's not plain sh.
	if envShell != "" && envShell != preferred && !isShShell(envShell) {
		candidates = append(candidates, envShell)
	}

	// Common interactive shells.
	candidates = append(candidates,
		"/bin/zsh",
		"/usr/bin/zsh",
		"/bin/bash",
		"/usr/bin/bash",
		"/usr/local/bin/bash", // Homebrew/macOS
	)

	// If preferred was sh (or empty), try it after better shells.
	if preferred != "" {
		candidates = append(candidates, preferred)
	}

	// Final fallback.
	candidates = append(candidates, "/bin/sh")

	seen := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		c = normalizeShellValue(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}

		// Resolve bare command names via PATH.
		if !strings.Contains(c, "/") {
			if resolved, err := exec.LookPath(c); err == nil {
				return resolved
			}
			continue
		}

		if isExecutable(c) {
			return c
		}
	}

	return "/bin/sh"
}

// maybeAutoUpdate checks the server for a newer agent binary and swaps it in-place.
// This is opt-in via config `auto_update: true` or env `REXEC_AUTO_UPDATE=true`.
func maybeAutoUpdate(cfg *AgentConfig) {
	enabled := cfg.AutoUpdate
	if env := os.Getenv("REXEC_AUTO_UPDATE"); env != "" {
		enabled = parseBool(env)
	}
	if !enabled {
		return
	}

	latest, err := fetchLatestVersion(cfg.Host)
	if err != nil || latest == "" {
		log.Printf("[AutoUpdate] Could not check latest version: %v", err)
		return
	}

	current := Version
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if !isNewerVersion(latest, current) {
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[AutoUpdate] Could not locate executable: %v", err)
		return
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	suffix, err := platformSuffix()
	if err != nil {
		log.Printf("[AutoUpdate] Unsupported platform: %v", err)
		return
	}

	downloadURL := strings.TrimRight(cfg.Host, "/") + "/downloads/rexec-agent-" + suffix
	dir := filepath.Dir(exePath)

	log.Printf("[AutoUpdate] Updating rexec-agent from %s to %s...", current, latest)
	tmpPath, err := downloadToTemp(downloadURL, dir)
	if err != nil {
		log.Printf("[AutoUpdate] Download failed: %v", err)
		return
	}
	defer os.Remove(tmpPath)

	if !verifyDownloadedBinary(tmpPath, latest) {
		log.Printf("[AutoUpdate] Verification failed; keeping current binary.")
		return
	}

	// Preserve executable mode.
	if fi, err := os.Stat(exePath); err == nil {
		_ = os.Chmod(tmpPath, fi.Mode())
	} else {
		_ = os.Chmod(tmpPath, 0755)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		log.Printf("[AutoUpdate] Replace failed: %v", err)
		return
	}

	log.Printf("[AutoUpdate] Updated to %s; restarting agent.", latest)
	args := append([]string{exePath}, os.Args[1:]...)
	if err := syscall.Exec(exePath, args, os.Environ()); err != nil {
		log.Printf("[AutoUpdate] Restart failed: %v (new binary will be used on next start)", err)
	}
}

func fetchLatestVersion(host string) (string, error) {
	url := strings.TrimRight(host, "/") + "/api/version"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}
	return strings.TrimSpace(v.Version), nil
}

func isNewerVersion(latest, current string) bool {
	lMaj, lMin, lPat, okL := parseSemver(latest)
	cMaj, cMin, cPat, okC := parseSemver(current)
	if !okL || !okC {
		return false
	}
	if lMaj != cMaj {
		return lMaj > cMaj
	}
	if lMin != cMin {
		return lMin > cMin
	}
	return lPat > cPat
}

func parseSemver(v string) (int, int, int, bool) {
	s := strings.TrimSpace(strings.TrimPrefix(v, "v"))
	s = strings.SplitN(s, "-", 2)[0]
	parts := strings.Split(s, ".")
	if len(parts) == 0 {
		return 0, 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	minor, patch := 0, 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}
	return major, minor, patch, true
}

func platformSuffix() (string, error) {
	osPart := runtime.GOOS
	switch osPart {
	case "linux", "darwin":
	default:
		return "", fmt.Errorf("unsupported os %q", osPart)
	}

	archPart := runtime.GOARCH
	switch archPart {
	case "amd64", "arm64":
	case "arm":
		archPart = "armv7"
	default:
		return "", fmt.Errorf("unsupported arch %q", archPart)
	}

	return osPart + "-" + archPart, nil
}

func downloadToTemp(downloadURL, dir string) (string, error) {
	tmpFile, err := os.CreateTemp(dir, "rexec-agent-update-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer tmpFile.Close()

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		os.Remove(tmpPath)
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	// Basic sanity check: non-empty binary.
	if fi, err := os.Stat(tmpPath); err != nil || fi.Size() < 1024*1024 {
		os.Remove(tmpPath)
		return "", fmt.Errorf("downloaded file too small")
	}

	// Make it executable so we can verify its version before swapping.
	_ = os.Chmod(tmpPath, 0755)

	return tmpPath, nil
}

func verifyDownloadedBinary(path, expectedVersion string) bool {
	out, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		return false
	}
	expectedNoV := strings.TrimPrefix(expectedVersion, "v")
	return strings.Contains(string(out), expectedNoV)
}

func saveAgentConfig(cfg *AgentConfig) error {
	configPath := getConfigPath()
	dir := filepath.Dir(configPath)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Preserve YAML configs created by install script.
	if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
		existing, _ := os.ReadFile(configPath)
		lines := strings.Split(string(existing), "\n")

		updates := map[string]string{
			"api_url":  cfg.Host,
			"token":    cfg.Token,
			"agent_id": cfg.ID,
			"name":     cfg.Name,
			"shell":    cfg.Shell,
			"auto_update": func() string {
				if cfg.AutoUpdate {
					return "true"
				}
				return "false"
			}(),
		}
		seen := make(map[string]bool)
		out := make([]string, 0, len(lines))

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				out = append(out, line)
				continue
			}

			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				if val, ok := updates[key]; ok {
					prefix := ""
					if idx := strings.Index(line, key); idx > 0 {
						prefix = line[:idx]
					}
					out = append(out, fmt.Sprintf("%s%s: %s", prefix, key, val))
					seen[key] = true
					continue
				}
			}
			out = append(out, line)
		}

		for key, val := range updates {
			if !seen[key] {
				out = append(out, fmt.Sprintf("%s: %s", key, val))
			}
		}

		return os.WriteFile(configPath, []byte(strings.Join(out, "\n")), 0600)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func handleRegister(args []string) {
	cfg := &AgentConfig{
		Host:  DefaultHost,
		Shell: os.Getenv("SHELL"),
	}

	if cfg.Shell == "" {
		cfg.Shell = "/bin/bash"
	}

	// Get hostname as default name
	hostname, _ := os.Hostname()
	cfg.Name = hostname

	// Parse args
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "-n":
			if i+1 < len(args) {
				cfg.Name = args[i+1]
				i++
			}
		case "--description", "-d":
			if i+1 < len(args) {
				cfg.Description = args[i+1]
				i++
			}
		case "--shell", "-s":
			if i+1 < len(args) {
				cfg.Shell = args[i+1]
				i++
			}
		case "--tags", "-t":
			if i+1 < len(args) {
				cfg.Tags = strings.Split(args[i+1], ",")
				i++
			}
		case "--token":
			if i+1 < len(args) {
				cfg.Token = args[i+1]
				i++
			}
		case "--host":
			if i+1 < len(args) {
				cfg.Host = args[i+1]
				i++
			}
		}
	}

	// Check for token in env
	if cfg.Token == "" {
		cfg.Token = os.Getenv("REXEC_TOKEN")
	}

	// Interactive token input if needed
	if cfg.Token == "" {
		fmt.Printf("\n%s%sRexec Agent Registration%s\n\n", Bold, Cyan, Reset)
		fmt.Printf("To register this machine, you need a Rexec API token.\n")
		fmt.Printf("Get one from: %s%s%s\n\n", Cyan, cfg.Host, Reset)
		fmt.Print("Enter token: ")

		reader := bufio.NewReader(os.Stdin)
		token, _ := reader.ReadString('\n')
		cfg.Token = strings.TrimSpace(token)

		if cfg.Token == "" {
			fmt.Printf("%sCancelled%s\n", Yellow, Reset)
			return
		}
	}

	// Verify token and register with API
	fmt.Printf("%sRegistering agent...%s\n", Dim, Reset)

	regData := map[string]interface{}{
		"name":        cfg.Name,
		"description": cfg.Description,
		"type":        "agent",
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"shell":       cfg.Shell,
		"tags":        cfg.Tags,
	}

	resp, err := apiRequest(cfg.Host, cfg.Token, "POST", "/api/agents/register", regData)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		fmt.Printf("%sInvalid token%s\n", Red, Reset)
		os.Exit(1)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("%sRegistration failed: %s%s\n", Red, string(body), Reset)
		os.Exit(1)
	}

	var result struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	cfg.ID = result.ID
	cfg.Registered = true

	// Use the new API token from registration (long-lived, doesn't expire)
	if result.Token != "" {
		cfg.Token = result.Token
	}

	if err := saveAgentConfig(cfg); err != nil {
		fmt.Printf("%sError saving config: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}

	fmt.Printf("\n%s✓ Agent registered successfully!%s\n", Green, Reset)
	fmt.Printf("  ID:   %s\n", cfg.ID)
	fmt.Printf("  Name: %s\n", cfg.Name)
	if result.Token != "" {
		fmt.Printf("  Token: %s (saved to config)%s\n", Green, Reset)
	}
	fmt.Printf("\nStart the agent with: %srexec-agent start%s\n\n", Cyan, Reset)
}

func handleStart(args []string) {
	cfg, err := loadAgentConfig()
	if err != nil {
		fmt.Printf("%sError loading config: %v%s\n", Red, err, Reset)
		fmt.Printf("Agent not configured. Run 'rexec-agent register' or provide --config flag.\n")
		os.Exit(1)
	}

	if cfg.Token == "" {
		fmt.Printf("%sNo token found in config. Set token in config or REXEC_TOKEN env var.%s\n", Red, Reset)
		os.Exit(1)
	}

	if cfg.ID == "" {
		fmt.Printf("%sNo agent_id found in config. Re-run 'rexec-agent register' or set agent_id in your config file.%s\n", Red, Reset)
		os.Exit(1)
	}

	// Optional self-update on startup (opt-in).
	maybeAutoUpdate(cfg)

	// Check for --daemon flag
	daemon := false
	for _, arg := range args {
		if arg == "--daemon" || arg == "-d" {
			daemon = true
		}
	}

	if daemon {
		startDaemon()
		return
	}

	agent := &Agent{
		config:  cfg,
		running: true,
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("\n%sShutting down agent...%s\n", Yellow, Reset)
		agent.Stop()
		os.Exit(0)
	}()

	// Check token type and warn if using JWT
	tokenType := "API token"
	if !strings.HasPrefix(cfg.Token, "rexec_") {
		tokenType = "JWT token (may expire)"
		fmt.Printf("\n%sWarning: Using JWT token which expires after 24 hours.%s\n", Yellow, Reset)
		fmt.Printf("%sRun 'rexec-agent refresh-token' to switch to a permanent API token.%s\n\n", Yellow, Reset)
	}

	fmt.Printf("\n%s%sRexec Agent%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("  Name:   %s\n", cfg.Name)
	fmt.Printf("  ID:     %s\n", cfg.ID)
	fmt.Printf("  Host:   %s\n", cfg.Host)
	fmt.Printf("  Shell:  %s\n", cfg.Shell)
	fmt.Printf("  Token:  %s\n", tokenType)
	fmt.Printf("\n%sConnecting to Rexec...%s\n", Dim, Reset)

	// Start the agent
	if err := agent.Start(); err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
}

func (a *Agent) Start() error {
	for a.running {
		err := a.connect()
		if err != nil {
			if !a.running {
				return nil
			}

			a.reconnects++
			// Exponential backoff: 5s, 10s, 20s, 30s (max)
			backoff := time.Duration(min(5*(1<<min(a.reconnects, 3)), 30)) * time.Second

			log.Printf("Connection lost: %v. Reconnecting in %v... (attempt %d)", err, backoff, a.reconnects)
			time.Sleep(backoff)
			continue
		}

		a.reconnects = 0
		a.handleConnection()
	}

	return nil
}

func (a *Agent) connect() error {
	wsHost := strings.Replace(a.config.Host, "https://", "wss://", 1)
	wsHost = strings.Replace(wsHost, "http://", "ws://", 1)

	wsURL := fmt.Sprintf("%s/ws/agent/%s?token=%s",
		wsHost,
		a.config.ID,
		url.QueryEscape(a.config.Token),
	)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second

	conn, resp, err := dialer.Dial(wsURL, http.Header{
		"X-Agent-Name":   []string{a.config.Name},
		"X-Agent-OS":     []string{runtime.GOOS},
		"X-Agent-Arch":   []string{runtime.GOARCH},
		"X-Agent-Shell":  []string{a.config.Shell},
		"X-Agent-Distro": []string{detectDistro()},
	})
	if err != nil {
		if resp != nil {
			// If we get a 4xx error (Unauthorized, Forbidden, Not Found), stop the agent
			// This happens when the agent is deleted from the server or token is invalid/expired
			if resp.StatusCode == http.StatusUnauthorized ||
				resp.StatusCode == http.StatusForbidden ||
				resp.StatusCode == http.StatusNotFound {
				log.Printf("Fatal: Server rejected connection (Status %d).", resp.StatusCode)
				if resp.StatusCode == http.StatusUnauthorized {
					log.Printf("Token may be expired or invalid. Use an API token (rexec_...) for persistent connections.")
					log.Printf("Generate one at: %s/account/api", a.config.Host)
				} else if resp.StatusCode == http.StatusNotFound {
					log.Printf("Agent not found. It may have been deleted. Run 'rexec-agent register' to re-register.")
				}
				a.running = false
				return nil // Return nil to stop the retry loop naturally
			}
		}
		return err
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	log.Printf("Connected to Rexec successfully")

	// Send system info on connect
	a.sendSystemInfo()

	// Send initial stats immediately so user sees metrics right away
	stats := a.collectStats()
	a.sendMessage("stats", stats)

	// Start periodic stats reporting
	go a.reportStats()

	return nil
}

// detectDistro reads /etc/os-release (or /etc/lsb-release as fallback) to determine the Linux distribution.
// Returns empty string for non-Linux or if detection fails.
func detectDistro() string {
	if runtime.GOOS != "linux" {
		return ""
	}

	// Try /etc/os-release first (standard on most modern distros)
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		return parseOSRelease(string(data))
	}

	// Fallback to /etc/lsb-release (older Ubuntu/Debian)
	if data, err := os.ReadFile("/etc/lsb-release"); err == nil {
		return parseLSBRelease(string(data))
	}

	// Check for specific distro files as last resort
	if _, err := os.Stat("/etc/alpine-release"); err == nil {
		return "alpine"
	}
	if _, err := os.Stat("/etc/arch-release"); err == nil {
		return "arch"
	}
	if _, err := os.Stat("/etc/gentoo-release"); err == nil {
		return "gentoo"
	}
	if _, err := os.Stat("/etc/fedora-release"); err == nil {
		return "fedora"
	}
	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return "rhel"
	}
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return "debian"
	}

	return ""
}

// parseOSRelease parses /etc/os-release format to extract the distribution ID
func parseOSRelease(data string) string {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, `"'`)
			return strings.ToLower(id)
		}
	}
	return ""
}

// parseLSBRelease parses /etc/lsb-release format
func parseLSBRelease(data string) string {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DISTRIB_ID=") {
			id := strings.TrimPrefix(line, "DISTRIB_ID=")
			id = strings.Trim(id, `"'`)
			return strings.ToLower(id)
		}
	}
	return ""
}

// isRoot returns true if running as root user
func isRoot() bool {
	return os.Geteuid() == 0
}

// sudoAvailable checks if sudo is installed and the current user can use it without password
func sudoAvailable() bool {
	// If already root, no need for sudo
	if isRoot() {
		return false
	}

	// Check if sudo command exists
	if _, err := exec.LookPath("sudo"); err != nil {
		return false
	}

	// Check if user can actually use sudo non-interactively
	// -n means non-interactive (don't prompt for password)
	cmd := exec.Command("sudo", "-n", "true")
	return cmd.Run() == nil
}

// detectVirtualization detects the virtualization/container environment
func detectVirtualization() string {
	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}

	// Check for Podman/other containers
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return "container"
	}

	// Check cgroup for container hints (LXC, Docker, etc.)
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "lxc") {
			return "lxc"
		}
		if strings.Contains(content, "docker") {
			return "docker"
		}
		if strings.Contains(content, "kubepods") {
			return "kubernetes"
		}
	}

	// Check for WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			return "wsl"
		}
	}

	// Check for Proxmox VE host
	if _, err := os.Stat("/etc/pve"); err == nil {
		return "proxmox-host"
	}

	// Check systemd-detect-virt if available
	if cmd, err := exec.LookPath("systemd-detect-virt"); err == nil {
		if output, err := exec.Command(cmd).Output(); err == nil {
			virt := strings.TrimSpace(string(output))
			if virt != "none" && virt != "" {
				return virt
			}
		}
	}

	return ""
}

// getSystemInfo collects machine information
func (a *Agent) getSystemInfo() map[string]interface{} {
	info := map[string]interface{}{
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"num_cpu":  runtime.NumCPU(),
		"hostname": "",
		"memory":   map[string]uint64{},
		"disk":     map[string]uint64{},
	}

	// Add distro for Linux systems
	if distro := detectDistro(); distro != "" {
		info["distro"] = distro
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info["hostname"] = hostname
	}

	// Detect cloud region (best-effort)
	if region := detectRegion(); region != "" {
		info["region"] = region
	}

	// Get memory info (Linux)
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			memInfo := map[string]uint64{}
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					var value uint64
					fmt.Sscanf(fields[1], "%d", &value)
					value *= 1024 // Convert from KB to bytes
					switch fields[0] {
					case "MemTotal:":
						memInfo["total"] = value
					case "MemAvailable:":
						memInfo["available"] = value
					case "MemFree:":
						memInfo["free"] = value
					}
				}
			}
			info["memory"] = memInfo
		}
	}

	// Get disk info
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		info["disk"] = map[string]uint64{
			"total":     stat.Blocks * uint64(stat.Bsize),
			"free":      stat.Bfree * uint64(stat.Bsize),
			"available": stat.Bavail * uint64(stat.Bsize),
		}
	}

	// Detect virtualization environment (LXC, Docker, PVE, WSL, etc.)
	if virt := detectVirtualization(); virt != "" {
		info["virtualization"] = virt
	}

	// Add privilege information
	info["is_root"] = isRoot()
	info["has_sudo"] = sudoAvailable()

	return info
}

func detectRegion() string {
	// Manual override for on-prem/local or custom installs
	if region := strings.TrimSpace(os.Getenv("REXEC_REGION")); region != "" {
		return region
	}

	// Common cloud/runtime env vars
	for _, key := range []string{
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
		"GOOGLE_CLOUD_REGION",
		"AZURE_REGION",
	} {
		if region := strings.TrimSpace(os.Getenv(key)); region != "" {
			return region
		}
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return ""
	}

	// Disable proxies to avoid leaking metadata calls via HTTP_PROXY
	metadataTransport := transport.Clone()
	metadataTransport.Proxy = nil

	client := &http.Client{
		Timeout:   400 * time.Millisecond,
		Transport: metadataTransport,
	}

	if region := detectAWSRegion(client); region != "" {
		return region
	}
	if region := detectGCPRegion(client); region != "" {
		return region
	}
	if region := detectAzureRegion(client); region != "" {
		return region
	}
	if region := detectDigitalOceanRegion(client); region != "" {
		return region
	}

	// Best-effort fallback for on-prem/local machines: infer approximate region via IP geolocation.
	// This is optional and can be disabled with REXEC_DISABLE_GEOIP=true.
	if region := detectRegionFromIPGeo(); region != "" {
		return region
	}

	return ""
}

func detectRegionFromIPGeo() string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REXEC_DISABLE_GEOIP")), "true") || os.Getenv("REXEC_DISABLE_GEOIP") == "1" {
		return ""
	}

	geoURL := strings.TrimSpace(os.Getenv("REXEC_GEOIP_URL"))
	if geoURL == "" {
		geoURL = "https://ipinfo.io/json"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, geoURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "rexec-agent/"+Version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var payload struct {
		City    string `json:"city"`
		Region  string `json:"region"`
		Country string `json:"country"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&payload); err != nil {
		return ""
	}

	primary := strings.TrimSpace(payload.Region)
	if primary == "" {
		primary = strings.TrimSpace(payload.City)
	}
	country := strings.TrimSpace(payload.Country)

	if primary != "" && country != "" {
		return primary + ", " + country
	}
	if primary != "" {
		return primary
	}
	return country
}

func detectAWSRegion(client *http.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	if region, status := awsGetRegionFromIdentityDoc(ctx, client, ""); status == http.StatusOK {
		return region
	} else if status != http.StatusUnauthorized && status != http.StatusForbidden {
		return ""
	}

	token := awsIMDSToken(ctx, client)
	if token == "" {
		return ""
	}
	region, status := awsGetRegionFromIdentityDoc(ctx, client, token)
	if status == http.StatusOK {
		return region
	}
	return ""
}

func awsGetRegionFromIdentityDoc(ctx context.Context, client *http.Client, token string) (string, int) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://169.254.169.254/latest/dynamic/instance-identity/document", nil)
	if err != nil {
		return "", 0
	}
	if token != "" {
		req.Header.Set("X-aws-ec2-metadata-token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<10))
	if err != nil {
		return "", 0
	}

	var doc struct {
		Region           string `json:"region"`
		AvailabilityZone string `json:"availabilityZone"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", http.StatusOK
	}
	if doc.Region != "" {
		return doc.Region, http.StatusOK
	}
	if doc.AvailabilityZone != "" {
		// us-east-1a -> us-east-1
		if idx := strings.LastIndex(doc.AvailabilityZone, "-"); idx > 0 {
			return doc.AvailabilityZone[:idx], http.StatusOK
		}
	}
	return "", http.StatusOK
}

func awsIMDSToken(ctx context.Context, client *http.Client) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "60")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func detectGCPRegion(client *http.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if err != nil {
		return ""
	}

	zonePath := strings.TrimSpace(string(body))
	if zonePath == "" {
		return ""
	}
	parts := strings.Split(zonePath, "/")
	zone := parts[len(parts)-1] // e.g. "us-central1-a"
	if zone == "" {
		return ""
	}
	if idx := strings.LastIndex(zone, "-"); idx > 0 {
		return zone[:idx]
	}
	return ""
}

func detectAzureRegion(client *http.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://169.254.169.254/metadata/instance/compute/location?api-version=2021-02-01&format=text", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata", "true")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func detectDigitalOceanRegion(client *http.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://169.254.169.254/metadata/v1.json", nil)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<10))
	if err != nil {
		return ""
	}

	var meta struct {
		Region string `json:"region"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Region)
}

// sendSystemInfo sends machine info to the server
func (a *Agent) sendSystemInfo() {
	info := a.getSystemInfo()
	a.sendMessage("system_info", info)
}

// reportStats periodically sends system stats
func (a *Agent) reportStats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for a.running {
		select {
		case <-ticker.C:
			a.mu.Lock()
			connected := a.conn != nil
			a.mu.Unlock()

			if !connected {
				return
			}

			stats := a.collectStats()
			a.sendMessage("stats", stats)
		}
	}
}

// collectStats gathers current CPU, memory, disk usage
func (a *Agent) collectStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cpu_percent":  0.0,
		"memory":       uint64(0),
		"memory_limit": uint64(0),
		"disk_usage":   uint64(0),
		"disk_limit":   uint64(0),
	}

	// Memory stats (Linux)
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			var total, available uint64
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					var value uint64
					fmt.Sscanf(fields[1], "%d", &value)
					value *= 1024
					switch fields[0] {
					case "MemTotal:":
						total = value
					case "MemAvailable:":
						available = value
					}
				}
			}
			if total > 0 {
				stats["memory"] = total - available
				stats["memory_limit"] = total
			}
		}
	}

	// Disk stats
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		stats["disk_usage"] = total - free
		stats["disk_limit"] = total
	}

	// CPU usage (simplified - load average on Linux)
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/loadavg"); err == nil {
			fields := strings.Fields(string(data))
			if len(fields) > 0 {
				var load float64
				fmt.Sscanf(fields[0], "%f", &load)
				// Convert load average to approximate CPU percent
				cpuPercent := (load / float64(runtime.NumCPU())) * 100
				if cpuPercent > 100 {
					cpuPercent = 100
				}
				stats["cpu_percent"] = cpuPercent
			}
		}
	}

	return stats
}

func (a *Agent) handleConnection() {
	defer func() {
		a.mu.Lock()
		if a.conn != nil {
			a.conn.Close()
			a.conn = nil
		}
		a.mu.Unlock()
	}()

	for a.running {
		_, message, err := a.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "shell_start":
			var startData struct {
				SessionID  string `json:"session_id"`
				NewSession bool   `json:"new_session"`
			}
			if err := json.Unmarshal(msg.Data, &startData); err == nil {
				go a.startShellSession(startData.SessionID, startData.NewSession)
			} else {
				// Backwards compatibility - no data means main session
				go a.startShellSession("main", false)
			}

		case "shell_input":
			var input struct {
				SessionID string `json:"session_id"`
				Data      []byte `json:"data"`
			}
			if err := json.Unmarshal(msg.Data, &input); err == nil {
				a.mu.Lock()
				// Route input to the right session, or main PTY for backwards compat
				session := a.sessions[input.SessionID]
				if session != nil && session.Ptmx != nil && len(input.Data) > 0 {
					session.Ptmx.Write(input.Data)
				} else if a.mainPtmx != nil && len(input.Data) > 0 {
					// Fallback to main PTY
					a.mainPtmx.Write(input.Data)
				}
				a.mu.Unlock()
			}

		case "shell_resize":
			var size struct {
				SessionID string `json:"session_id"`
				Cols      int    `json:"cols"`
				Rows      int    `json:"rows"`
			}
			if err := json.Unmarshal(msg.Data, &size); err == nil {
				a.mu.Lock()
				session := a.sessions[size.SessionID]
				if session != nil && session.Ptmx != nil {
					pty.Setsize(session.Ptmx, &pty.Winsize{
						Cols: uint16(size.Cols),
						Rows: uint16(size.Rows),
					})
				} else if a.mainPtmx != nil {
					// Fallback to main PTY
					pty.Setsize(a.mainPtmx, &pty.Winsize{
						Cols: uint16(size.Cols),
						Rows: uint16(size.Rows),
					})
				}
				a.mu.Unlock()
			}

		case "shell_stop":
			a.stopShell()

		case "shell_stop_session":
			var stopData struct {
				SessionID string `json:"session_id"`
			}
			if err := json.Unmarshal(msg.Data, &stopData); err == nil && stopData.SessionID != "" {
				a.cleanupSession(stopData.SessionID)
			}

		case "ping":
			a.sendMessage("pong", nil)

		case "exec":
			var execCmd struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal(msg.Data, &execCmd); err == nil {
				go a.execCommand(execCmd.Command)
			}
		}
	}
}

// startShellSession starts a shell session with the given ID
// Each session gets its own independent shell/PTY
func (a *Agent) startShellSession(sessionID string, newSession bool) {
	// Send immediate ACK so client knows we received the request
	a.sendMessage("shell_starting", map[string]string{"session_id": sessionID})

	a.mu.Lock()
	if a.sessions == nil {
		a.sessions = make(map[string]*ShellSession)
	}
	// The main session is always keyed as "main" on the agent, regardless of the
	// client connection ID. This keeps routing stable across reconnects/tabs.
	if !newSession {
		sessionID = "main"
	}

	// Check if session already exists
	if _, exists := a.sessions[sessionID]; exists {
		a.mu.Unlock()
		// Session exists, just notify it's ready
		a.sendMessage("shell_started", map[string]string{"session_id": sessionID})
		return
	}

	// For main session, check if mainCmd is already running
	if !newSession && a.mainCmd != nil {
		a.mu.Unlock()
		// Already running, notify ready
		a.sendMessage("shell_started", map[string]string{"session_id": sessionID})
		return
	}
	a.mu.Unlock()

	// Use cached shell path from config (resolved once at startup)
	shellPath := a.config.Shell
	if shellPath == "" {
		shellPath = "/bin/bash"
	}
	args := []string{"-l", "-i"}
	// Avoid login mode for plain sh: it commonly sources /etc/profile where
	// bashisms or CRLF can produce noisy "not found" errors.
	if isShShell(shellPath) {
		args = []string{"-i"}
	}
	cmd := exec.Command(shellPath, args...)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cmd.Dir = home
	}
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"REXEC_AGENT=1",
		"SHELL="+shellPath,
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		a.sendMessage("shell_error", map[string]string{"session_id": sessionID, "error": err.Error()})
		return
	}

	// Set a reasonable default PTY size to avoid garbled output
	// This will be updated when client sends resize
	pty.Setsize(ptmx, &pty.Winsize{
		Cols: 120,
		Rows: 30,
	})

	session := &ShellSession{
		ID:     sessionID,
		Cmd:    cmd,
		Ptmx:   ptmx,
		IsMain: !newSession,
	}

	a.mu.Lock()
	a.sessions[sessionID] = session
	if !newSession {
		a.mainCmd = cmd
		a.mainPtmx = ptmx
	}
	a.mu.Unlock()

	a.sendMessage("shell_started", map[string]string{"session_id": sessionID})

	// Read PTY output and send to WebSocket
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}

			a.sendMessage("shell_output", map[string]interface{}{
				"session_id": sessionID,
				"data":       buf[:n],
			})
		}

		a.sendMessage("shell_stopped", map[string]string{"session_id": sessionID})
		a.cleanupSession(sessionID)
	}()

	// Wait for shell to exit
	cmd.Wait()
}

// cleanupSession removes a session from the map
func (a *Agent) cleanupSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if session, exists := a.sessions[sessionID]; exists {
		// Gracefully terminate the shell so it can flush history/state.
		if session.Cmd != nil && session.Cmd.Process != nil {
			_ = session.Cmd.Process.Signal(syscall.SIGTERM)
		}
		if session.Ptmx != nil {
			session.Ptmx.Close()
		}
		delete(a.sessions, sessionID)

		if session.IsMain {
			a.mainCmd = nil
			a.mainPtmx = nil
		}
	}
}

// Legacy startShell for backwards compatibility
func (a *Agent) startShell() {
	a.startShellSession("main", false)
}

func (a *Agent) stopShell() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Stop all sessions
	for sessionID, session := range a.sessions {
		if session.Cmd != nil && session.Cmd.Process != nil {
			_ = session.Cmd.Process.Signal(syscall.SIGTERM)
		}
		if session.Ptmx != nil {
			session.Ptmx.Close()
		}
		delete(a.sessions, sessionID)
	}

	a.mainPtmx = nil
	a.mainCmd = nil
}

func (a *Agent) execCommand(command string) {
	cmd := exec.Command(a.config.Shell, "-c", command)
	output, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"command": command,
		"output":  string(output),
		"success": err == nil,
	}

	if err != nil {
		result["error"] = err.Error()
	}

	a.sendMessage("exec_result", result)
}

func (a *Agent) sendMessage(msgType string, data interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn == nil {
		return fmt.Errorf("not connected")
	}

	msg := map[string]interface{}{
		"type": msgType,
		"data": data,
	}

	return a.conn.WriteJSON(msg)
}

func (a *Agent) Stop() {
	a.running = false
	a.stopShell()

	a.mu.Lock()
	if a.conn != nil {
		a.conn.Close()
	}
	a.mu.Unlock()
}

func handleStop() {
	// Find and kill running agent process
	pidFile := filepath.Join(os.TempDir(), "rexec-agent.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Printf("%sAgent not running%s\n", Yellow, Reset)
		return
	}

	var pid int
	fmt.Sscanf(string(data), "%d", &pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("%sAgent not running%s\n", Yellow, Reset)
		os.Remove(pidFile)
		return
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("%sError stopping agent: %v%s\n", Red, err, Reset)
		return
	}

	os.Remove(pidFile)
	fmt.Printf("%s✓ Agent stopped%s\n", Green, Reset)
}

func handleStatus() {
	cfg, err := loadAgentConfig()
	if err != nil {
		fmt.Printf("%sAgent not registered%s\n", Dim, Reset)
		return
	}

	fmt.Printf("\n%s%sAgent Status%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("  Registered: %s✓%s\n", Green, Reset)
	fmt.Printf("  ID:         %s\n", cfg.ID)
	fmt.Printf("  Name:       %s\n", cfg.Name)
	fmt.Printf("  Host:       %s\n", cfg.Host)
	fmt.Printf("  Shell:      %s\n", cfg.Shell)

	// Check if running
	pidFile := filepath.Join(os.TempDir(), "rexec-agent.pid")
	if _, err := os.Stat(pidFile); err == nil {
		fmt.Printf("  Running:    %s✓ Yes%s\n", Green, Reset)
	} else {
		fmt.Printf("  Running:    %sNo%s\n", Yellow, Reset)
	}

	fmt.Println()
}

func handleUnregister() {
	cfg, err := loadAgentConfig()
	if err != nil {
		fmt.Printf("%sAgent not registered%s\n", Dim, Reset)
		return
	}

	fmt.Printf("%sAre you sure you want to unregister this agent? (y/N): %s", Yellow, Reset)
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "y" && confirm != "yes" {
		fmt.Printf("%sCancelled%s\n", Dim, Reset)
		return
	}

	// Call API to unregister
	resp, err := apiRequest(cfg.Host, cfg.Token, "DELETE", "/api/agents/"+cfg.ID, nil)
	if err != nil {
		fmt.Printf("%sWarning: Could not unregister from server: %v%s\n", Yellow, err, Reset)
	} else {
		resp.Body.Close()
	}

	// Remove local config
	os.Remove(getConfigPath())

	fmt.Printf("%s✓ Agent unregistered%s\n", Green, Reset)
}

func handleRefreshToken(args []string) {
	cfg, err := loadAgentConfig()
	if err != nil {
		fmt.Printf("%sAgent not registered. Run 'rexec-agent register' first.%s\n", Red, Reset)
		os.Exit(1)
	}

	if cfg.ID == "" {
		fmt.Printf("%sAgent ID not found in config. Re-run 'rexec-agent register' or set agent_id in your config file.%s\n", Red, Reset)
		os.Exit(1)
	}

	fmt.Printf("%sRefreshing agent token...%s\n\n", Dim, Reset)

	// Check for new token in args or prompt
	var newToken string
	for i := 0; i < len(args); i++ {
		if args[i] == "--token" && i+1 < len(args) {
			newToken = args[i+1]
			break
		}
	}

	if newToken == "" {
		newToken = os.Getenv("REXEC_TOKEN")
	}

	if newToken == "" {
		fmt.Printf("To fix 401 errors, you need an API token.\n")
		fmt.Printf("Generate one at: %s%s/account/api%s\n\n", Cyan, cfg.Host, Reset)
		fmt.Print("Enter new API token (rexec_...): ")

		reader := bufio.NewReader(os.Stdin)
		token, _ := reader.ReadString('\n')
		newToken = strings.TrimSpace(token)

		if newToken == "" {
			fmt.Printf("%sCancelled%s\n", Yellow, Reset)
			return
		}
	}

	// Validate it's an API token
	if !strings.HasPrefix(newToken, "rexec_") {
		fmt.Printf("%sWarning: Token doesn't start with 'rexec_'. API tokens are recommended for agents.%s\n", Yellow, Reset)
	}

	// Test the new token
	fmt.Printf("Testing new token...")
	// 1) Validate token works at all.
	resp, err := apiRequest(cfg.Host, newToken, "GET", "/api/tokens/validate", nil)
	if err != nil {
		fmt.Printf(" %sFailed: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		fmt.Printf(" %sInvalid token%s\n", Red, Reset)
		os.Exit(1)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf(" %sFailed: %s%s\n", Red, string(body), Reset)
		os.Exit(1)
	}

	// 2) Ensure token belongs to the same user as this agent by listing agents.
	listResp, err := apiRequest(cfg.Host, newToken, "GET", "/api/agents", nil)
	if err != nil {
		fmt.Printf(" %sFailed listing agents: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != 200 {
		body, _ := io.ReadAll(listResp.Body)
		fmt.Printf(" %sFailed listing agents: %s%s\n", Red, string(body), Reset)
		os.Exit(1)
	}

	var agents []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&agents)
	found := false
	for _, a := range agents {
		if a.ID == cfg.ID {
			found = true
			break
		}
	}
	if !found {
		fmt.Printf(" %sToken does not have access to agent %s%s\n", Red, cfg.ID, Reset)
		os.Exit(1)
	}

	fmt.Printf(" %s✓%s\n", Green, Reset)

	// Save the new token
	cfg.Token = newToken
	if err := saveAgentConfig(cfg); err != nil {
		fmt.Printf("%sError saving config: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}

	fmt.Printf("\n%s✓ Token updated successfully!%s\n", Green, Reset)
	fmt.Printf("Restart the agent: %ssudo systemctl restart rexec-agent%s\n", Cyan, Reset)
}

func handlePrepShell() {
	fmt.Printf("\n%s%sPreparing Enhanced Shell%s\n\n", Bold, Cyan, Reset)
	fmt.Printf("This will install zsh + oh-my-zsh for a better terminal experience.\n\n")

	// Check if already installed
	homeDir, _ := os.UserHomeDir()
	ohmyzshPath := filepath.Join(homeDir, ".oh-my-zsh")
	if _, err := os.Stat(ohmyzshPath); err == nil {
		fmt.Printf("%s✓ Oh-my-zsh already installed at %s%s\n", Green, ohmyzshPath, Reset)
		return
	}

	// Check for zsh
	zshPath, err := exec.LookPath("zsh")
	if err != nil {
		fmt.Printf("%s[1/3] Installing zsh...%s\n", Dim, Reset)
		if err := installZsh(); err != nil {
			fmt.Printf("%sError installing zsh: %v%s\n", Red, err, Reset)
			fmt.Printf("Please install zsh manually and re-run this command.\n")
			os.Exit(1)
		}
		zshPath, _ = exec.LookPath("zsh")
	} else {
		fmt.Printf("%s[1/3] zsh already installed at %s%s\n", Green, zshPath, Reset)
	}

	// Install oh-my-zsh
	fmt.Printf("%s[2/3] Installing oh-my-zsh...%s\n", Dim, Reset)
	ohmyzshCmd := exec.Command("sh", "-c",
		`git clone --depth=1 https://github.com/ohmyzsh/ohmyzsh.git "$HOME/.oh-my-zsh" 2>/dev/null`)
	ohmyzshCmd.Env = os.Environ()
	if err := ohmyzshCmd.Run(); err != nil {
		fmt.Printf("%sError installing oh-my-zsh: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}

	// Install plugins
	fmt.Printf("%s[3/3] Installing plugins...%s\n", Dim, Reset)
	plugins := []struct {
		name string
		repo string
	}{
		{"zsh-autosuggestions", "https://github.com/zsh-users/zsh-autosuggestions"},
		{"zsh-syntax-highlighting", "https://github.com/zsh-users/zsh-syntax-highlighting"},
		{"zsh-completions", "https://github.com/zsh-users/zsh-completions"},
	}

	pluginsDir := filepath.Join(homeDir, ".oh-my-zsh", "custom", "plugins")
	for _, plugin := range plugins {
		pluginPath := filepath.Join(pluginsDir, plugin.name)
		if _, err := os.Stat(pluginPath); err == nil {
			continue // Already installed
		}
		cmd := exec.Command("git", "clone", "--depth=1", plugin.repo, pluginPath)
		_ = cmd.Run() // Best effort
	}

	// Create .zshrc if it doesn't exist
	zshrcPath := filepath.Join(homeDir, ".zshrc")
	if _, err := os.Stat(zshrcPath); os.IsNotExist(err) {
		zshrc := `export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="robbyrussell"
plugins=(git zsh-autosuggestions zsh-syntax-highlighting zsh-completions)
source $ZSH/oh-my-zsh.sh
`
		if err := os.WriteFile(zshrcPath, []byte(zshrc), 0644); err != nil {
			fmt.Printf("%sWarning: Could not create .zshrc: %v%s\n", Yellow, err, Reset)
		}
	}

	fmt.Printf("\n%s✓ Enhanced shell installed successfully!%s\n", Green, Reset)
	fmt.Printf("Your terminal sessions will now use zsh with oh-my-zsh.\n")
	if zshPath != "" {
		fmt.Printf("To use it now: %sexec %s%s\n", Cyan, zshPath, Reset)
	}
}

func installZsh() error {
	// Determine if we need sudo prefix
	// If running as root or sudo is not available, don't use sudo
	sudoPrefix := ""
	if !isRoot() && sudoAvailable() {
		sudoPrefix = "sudo "
	} else if !isRoot() && !sudoAvailable() {
		// Not root and no sudo - check if we can install as root (LXC/container scenario)
		// In LXC containers, we're often already root
		fmt.Printf("%sNote: Running without sudo (not available or not configured)%s\n", Yellow, Reset)
	}

	// Detect package manager and install zsh
	var cmd *exec.Cmd

	switch {
	case commandExists("apt-get"):
		cmd = exec.Command("sh", "-c", sudoPrefix+"apt-get update && "+sudoPrefix+"apt-get install -y zsh git")
	case commandExists("dnf"):
		cmd = exec.Command("sh", "-c", sudoPrefix+"dnf install -y zsh git")
	case commandExists("yum"):
		cmd = exec.Command("sh", "-c", sudoPrefix+"yum install -y zsh git")
	case commandExists("pacman"):
		cmd = exec.Command("sh", "-c", sudoPrefix+"pacman -Sy --noconfirm zsh git")
	case commandExists("apk"):
		cmd = exec.Command("sh", "-c", sudoPrefix+"apk add --no-cache zsh git")
	case commandExists("zypper"):
		cmd = exec.Command("sh", "-c", sudoPrefix+"zypper install -y zsh git")
	case commandExists("brew"):
		// brew should never use sudo
		cmd = exec.Command("sh", "-c", "brew install zsh git")
	default:
		return fmt.Errorf("unsupported package manager")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func handleInstall() {
	if runtime.GOOS != "linux" {
		fmt.Printf("%sService installation is only supported on Linux%s\n", Red, Reset)
		return
	}

	cfg, err := loadAgentConfig()
	if err != nil {
		fmt.Printf("%sAgent not registered. Run 'rexec-agent register' first.%s\n", Red, Reset)
		return
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("%sError getting executable path: %v%s\n", Red, err, Reset)
		return
	}

	// Create systemd service file
	serviceContent := fmt.Sprintf(`[Unit]
Description=Rexec Agent
After=network.target

[Service]
Type=simple
ExecStart=%s start
Restart=always
RestartSec=10
Environment="REXEC_TOKEN=%s"
Environment="REXEC_HOST=%s"

[Install]
WantedBy=multi-user.target
`, execPath, cfg.Token, cfg.Host)

	servicePath := "/etc/systemd/system/rexec-agent.service"

	// Check if we have permissions
	if os.Geteuid() != 0 {
		fmt.Printf("%sRun with sudo to install as a service%s\n", Red, Reset)
		fmt.Printf("\nService file content:\n%s\n", serviceContent)
		return
	}

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		fmt.Printf("%sError writing service file: %v%s\n", Red, err, Reset)
		return
	}

	// Enable and start service
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "rexec-agent").Run()
	exec.Command("systemctl", "start", "rexec-agent").Run()

	fmt.Printf("%s✓ Service installed and started%s\n", Green, Reset)
	fmt.Printf("\nManage with:\n")
	fmt.Printf("  sudo systemctl status rexec-agent\n")
	fmt.Printf("  sudo systemctl stop rexec-agent\n")
	fmt.Printf("  sudo systemctl restart rexec-agent\n")
}

func startDaemon() {
	// Fork and run in background
	cmd := exec.Command(os.Args[0], "start")
	cmd.Env = os.Environ()
	cmd.Start()

	// Write PID file
	pidFile := filepath.Join(os.TempDir(), "rexec-agent.pid")
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)

	fmt.Printf("%s✓ Agent started in background (PID: %d)%s\n", Green, cmd.Process.Pid, Reset)
}

func apiRequest(host, token, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	url := host + endpoint
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Unused import placeholders
var _ = context.Background
