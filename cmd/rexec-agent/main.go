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
}

type Agent struct {
	config     *AgentConfig
	conn       *websocket.Conn
	pty        *os.File
	ptmx       *os.File
	cmd        *exec.Cmd
	mu         sync.Mutex
	running    bool
	reconnects int
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	switch os.Args[1] {
	case "help", "-h", "--help":
		showHelp()
	case "version", "-v", "--version":
		fmt.Printf("%srexec-agent%s v%s\n", Bold, Reset, Version)
	case "register":
		handleRegister(os.Args[2:])
	case "start":
		handleStart(os.Args[2:])
	case "stop":
		handleStop()
	case "status":
		handleStatus()
	case "unregister":
		handleUnregister()
	case "install":
		handleInstall()
	default:
		fmt.Printf("%sUnknown command: %s%s\n", Red, os.Args[1], Reset)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf(`
%s%sRexec Agent%s - Connect your server to Rexec

%sUSAGE:%s
  rexec-agent <command> [options]

%sCOMMANDS:%s
  register     Register this machine as a Rexec terminal
  start        Start the agent (connect to Rexec)
  stop         Stop the running agent
  status       Show agent status
  unregister   Remove this machine from Rexec
  install      Install as a system service

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
  rexec-agent status

%sENVIRONMENT:%s
  REXEC_TOKEN      Auth token
  REXEC_HOST       API host

`, Bold, Cyan, Reset,
		Yellow, Reset,
		Yellow, Reset,
		Yellow, Reset,
		DefaultHost,
		Yellow, Reset,
		Yellow, Reset)
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir, AgentFile)
}

func loadAgentConfig() (*AgentConfig, error) {
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		return nil, err
	}

	var cfg AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Override with env vars
	if host := os.Getenv("REXEC_HOST"); host != "" {
		cfg.Host = host
	}
	if token := os.Getenv("REXEC_TOKEN"); token != "" {
		cfg.Token = token
	}

	return &cfg, nil
}

func saveAgentConfig(cfg *AgentConfig) error {
	configPath := getConfigPath()
	dir := filepath.Dir(configPath)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
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
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	cfg.ID = result.ID
	cfg.Registered = true

	if err := saveAgentConfig(cfg); err != nil {
		fmt.Printf("%sError saving config: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}

	fmt.Printf("\n%s✓ Agent registered successfully!%s\n", Green, Reset)
	fmt.Printf("  ID:   %s\n", cfg.ID)
	fmt.Printf("  Name: %s\n", cfg.Name)
	fmt.Printf("\nStart the agent with: %srexec-agent start%s\n\n", Cyan, Reset)
}

func handleStart(args []string) {
	cfg, err := loadAgentConfig()
	if err != nil {
		fmt.Printf("%sAgent not registered. Run 'rexec-agent register' first.%s\n", Red, Reset)
		os.Exit(1)
	}

	if !cfg.Registered {
		fmt.Printf("%sAgent not registered. Run 'rexec-agent register' first.%s\n", Red, Reset)
		os.Exit(1)
	}

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

	fmt.Printf("\n%s%sRexec Agent%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("  Name:   %s\n", cfg.Name)
	fmt.Printf("  ID:     %s\n", cfg.ID)
	fmt.Printf("  Host:   %s\n", cfg.Host)
	fmt.Printf("  Shell:  %s\n", cfg.Shell)
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
			backoff := time.Duration(min(a.reconnects*2, 30)) * time.Second

			log.Printf("%sConnection lost: %v. Reconnecting in %v...%s", Yellow, err, backoff, Reset)
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

	conn, _, err := dialer.Dial(wsURL, http.Header{
		"X-Agent-Name":  []string{a.config.Name},
		"X-Agent-OS":    []string{runtime.GOOS},
		"X-Agent-Arch":  []string{runtime.GOARCH},
		"X-Agent-Shell": []string{a.config.Shell},
	})
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	log.Printf("%s✓ Connected to Rexec%s", Green, Reset)

	return nil
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
			go a.startShell()

		case "shell_input":
			var input struct {
				Data []byte `json:"data"`
			}
			if err := json.Unmarshal(msg.Data, &input); err == nil {
				if a.ptmx != nil {
					a.ptmx.Write(input.Data)
				}
			}

		case "shell_resize":
			var size struct {
				Cols int `json:"cols"`
				Rows int `json:"rows"`
			}
			if err := json.Unmarshal(msg.Data, &size); err == nil {
				if a.ptmx != nil {
					pty.Setsize(a.ptmx, &pty.Winsize{
						Cols: uint16(size.Cols),
						Rows: uint16(size.Rows),
					})
				}
			}

		case "shell_stop":
			a.stopShell()

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

func (a *Agent) startShell() {
	a.mu.Lock()
	if a.cmd != nil {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	// Start shell with PTY
	cmd := exec.Command(a.config.Shell, "-l")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"REXEC_AGENT=1",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		a.sendMessage("shell_error", map[string]string{"error": err.Error()})
		return
	}

	a.mu.Lock()
	a.cmd = cmd
	a.ptmx = ptmx
	a.mu.Unlock()

	a.sendMessage("shell_started", nil)

	// Read PTY output and send to WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}

			a.sendMessage("shell_output", map[string]interface{}{
				"data": buf[:n],
			})
		}

		a.sendMessage("shell_stopped", nil)
		a.stopShell()
	}()

	// Wait for shell to exit
	cmd.Wait()
}

func (a *Agent) stopShell() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ptmx != nil {
		a.ptmx.Close()
		a.ptmx = nil
	}

	if a.cmd != nil && a.cmd.Process != nil {
		a.cmd.Process.Kill()
		a.cmd = nil
	}
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
