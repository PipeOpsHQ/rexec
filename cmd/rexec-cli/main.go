package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

const (
	Version     = "1.0.0"
	DefaultHost = "https://rexec.pipeops.io"
	ConfigDir   = ".rexec"
	ConfigFile  = "config.json"
)

// ANSI colors
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BgBlue     = "\033[44m"
	BgMagenta  = "\033[45m"
	BgCyan     = "\033[46m"
)

type Config struct {
	Host     string `json:"host"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Tier     string `json:"tier"`
}

type Container struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Status    string `json:"status"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type Snippet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsPublic    bool   `json:"is_public"`
}

type AgentConfig struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		showHelp()
	case "version", "-v", "--version":
		showVersion()
	case "-i", "--interactive", "tui":
		handleInteractive()
	case "login":
		handleLogin(args)
	case "logout":
		handleLogout()
	case "whoami":
		handleWhoami()
	case "ls", "list":
		handleList(args)
	case "create":
		handleCreate(args)
	case "connect", "ssh":
		handleConnect(args)
	case "start":
		handleStart(args)
	case "stop":
		handleStop(args)
	case "rm", "delete":
		handleDelete(args)
	case "snippets":
		handleSnippets(args)
	case "run":
		handleRun(args)
	case "agent":
		handleAgent(args)
	case "dashboard", "ui":
		handleDashboard()
	case "config":
		handleConfig(args)
	default:
		fmt.Printf("%sUnknown command: %s%s\n", Red, cmd, Reset)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf(`
%s%s██████╗ ███████╗██╗  ██╗███████╗ ██████╗%s
%s%s██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝%s
%s%s██████╔╝█████╗   ╚███╔╝ █████╗  ██║     %s
%s%s██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║     %s
%s%s██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗%s
%s%s╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝%s

%sCloud Terminal Environment%s - v%s

%sUSAGE:%s
  rexec <command> [options]
  rexec -i                  Launch interactive TUI

%sCOMMANDS:%s
  %sAuthentication:%s
    login              Login to rexec (interactive or with --token)
    logout             Logout and clear credentials
    whoami             Show current user info

  %sTerminals:%s
    ls, list           List all terminals
    create             Create a new terminal
    connect, ssh       Connect to a terminal (interactive shell)
    start <id>         Start a stopped terminal
    stop <id>          Stop a running terminal
    rm, delete <id>    Delete a terminal

  %sSnippets & Macros:%s
    snippets           List/manage snippets
    run <snippet>      Run a snippet on a terminal

  %sAgent Mode:%s
    agent register     Register this machine as a rexec terminal
    agent start        Start the agent (connect to rexec)
    agent stop         Stop the agent
    agent status       Show agent status

  %sUtility:%s
    -i, tui            Launch interactive TUI dashboard
    dashboard, ui      Open TUI dashboard (alias for -i)
    config             View/edit configuration
    version            Show version info
    help               Show this help message

%sEXAMPLES:%s
  rexec login
  rexec create --name mydev --image ubuntu:22.04 --role devops
  rexec ls
  rexec connect abc123
  rexec run "docker-install" --terminal abc123
  rexec agent register --name "my-server"
  rexec dashboard

%sENVIRONMENT:%s
  REXEC_HOST         API host (default: %s)
  REXEC_TOKEN        Auth token (overrides config)

`, 
	Cyan, Bold, Reset,
	Cyan, Bold, Reset,
	Cyan, Bold, Reset,
	Cyan, Bold, Reset,
	Cyan, Bold, Reset,
	Cyan, Bold, Reset,
	Dim, Reset, Version,
	Yellow, Reset,
	Yellow, Reset,
	Blue, Reset,
	Blue, Reset,
	Blue, Reset,
	Blue, Reset,
	Blue, Reset,
	Yellow, Reset,
	Yellow, Reset,
	DefaultHost)
}

func showVersion() {
	fmt.Printf("%srexec-cli%s v%s\n", Bold, Reset, Version)
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// Config management
func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir, ConfigFile)
}

func loadConfig() *Config {
	cfg := &Config{Host: DefaultHost}
	
	// Check env overrides first
	if host := os.Getenv("REXEC_HOST"); host != "" {
		cfg.Host = host
	}
	if token := os.Getenv("REXEC_TOKEN"); token != "" {
		cfg.Token = token
	}
	
	// Load from file
	data, err := os.ReadFile(getConfigPath())
	if err == nil {
		json.Unmarshal(data, cfg)
	}
	
	// Env overrides file config
	if host := os.Getenv("REXEC_HOST"); host != "" {
		cfg.Host = host
	}
	if token := os.Getenv("REXEC_TOKEN"); token != "" {
		cfg.Token = token
	}
	
	return cfg
}

func saveConfig(cfg *Config) error {
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

// API helpers
func apiRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	cfg := loadConfig()
	
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}
	
	url := cfg.Host + endpoint
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func checkAuth() *Config {
	cfg := loadConfig()
	if cfg.Token == "" {
		fmt.Printf("%sNot logged in. Run 'rexec login' first.%s\n", Red, Reset)
		os.Exit(1)
	}
	return cfg
}

// Command handlers
func handleLogin(args []string) {
	cfg := loadConfig()
	
	// Check for --token flag
	for i, arg := range args {
		if arg == "--token" && i+1 < len(args) {
			cfg.Token = args[i+1]
			
			// Verify token
			resp, err := apiRequest("GET", "/api/profile", nil)
			if err != nil {
				fmt.Printf("%sError: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != 200 {
				fmt.Printf("%sInvalid token%s\n", Red, Reset)
				os.Exit(1)
			}
			
			var profile struct {
				Username string `json:"username"`
				Email    string `json:"email"`
				Tier     string `json:"tier"`
			}
			json.NewDecoder(resp.Body).Decode(&profile)
			
			cfg.Username = profile.Username
			cfg.Email = profile.Email
			cfg.Tier = profile.Tier
			
			if err := saveConfig(cfg); err != nil {
				fmt.Printf("%sError saving config: %v%s\n", Red, err, Reset)
				os.Exit(1)
			}
			
			fmt.Printf("%s✓ Logged in as %s (%s)%s\n", Green, profile.Username, profile.Email, Reset)
			return
		}
	}
	
	// Interactive login
	fmt.Printf("\n%s%sRexec Login%s\n\n", Bold, Cyan, Reset)
	fmt.Printf("1. Open %s%s/api/auth/oauth/url%s in your browser\n", Cyan, cfg.Host, Reset)
	fmt.Printf("2. Login with PipeOps\n")
	fmt.Printf("3. Copy the token from the callback\n\n")
	
	fmt.Print("Enter token: ")
	reader := bufio.NewReader(os.Stdin)
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)
	
	if token == "" {
		fmt.Printf("%sCancelled%s\n", Yellow, Reset)
		return
	}
	
	cfg.Token = token
	
	// Verify token
	resp, err := apiRequest("GET", "/api/profile", nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		fmt.Printf("%sInvalid token%s\n", Red, Reset)
		os.Exit(1)
	}
	
	var profile struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Tier     string `json:"tier"`
	}
	json.NewDecoder(resp.Body).Decode(&profile)
	
	cfg.Username = profile.Username
	cfg.Email = profile.Email
	cfg.Tier = profile.Tier
	
	if err := saveConfig(cfg); err != nil {
		fmt.Printf("%sError saving config: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	
	fmt.Printf("\n%s✓ Logged in as %s (%s)%s\n", Green, profile.Username, profile.Email, Reset)
}

func handleLogout() {
	configPath := getConfigPath()
	os.Remove(configPath)
	fmt.Printf("%s✓ Logged out%s\n", Green, Reset)
}

func handleWhoami() {
	cfg := checkAuth()
	
	resp, err := apiRequest("GET", "/api/profile", nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var profile struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Tier     string `json:"tier"`
		IsAdmin  bool   `json:"is_admin"`
	}
	json.NewDecoder(resp.Body).Decode(&profile)
	
	tierColor := Green
	switch profile.Tier {
	case "pro":
		tierColor = Cyan
	case "enterprise":
		tierColor = Magenta
	}
	
	fmt.Printf("\n%s%sUser Profile%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────\n")
	fmt.Printf("  Username: %s%s%s\n", Bold, profile.Username, Reset)
	fmt.Printf("  Email:    %s\n", profile.Email)
	fmt.Printf("  Tier:     %s%s%s\n", tierColor, profile.Tier, Reset)
	if profile.IsAdmin {
		fmt.Printf("  Role:     %s%sAdmin%s\n", Bold, Red, Reset)
	}
	fmt.Printf("  Host:     %s\n", cfg.Host)
	fmt.Println()
}

func handleList(args []string) {
	checkAuth()
	
	resp, err := apiRequest("GET", "/api/containers", nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var containers []Container
	json.NewDecoder(resp.Body).Decode(&containers)
	
	if len(containers) == 0 {
		fmt.Printf("\n%sNo terminals found.%s\n", Dim, Reset)
		fmt.Printf("Create one with: %srexec create --name mydev%s\n\n", Cyan, Reset)
		return
	}
	
	fmt.Printf("\n%s%sTerminals%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────────────────────────────────────────\n")
	fmt.Printf("%-12s %-20s %-15s %-12s %s\n", "ID", "NAME", "IMAGE", "STATUS", "ROLE")
	fmt.Printf("─────────────────────────────────────────────────────────────────────────────\n")
	
	for _, c := range containers {
		statusColor := Dim
		switch c.Status {
		case "running":
			statusColor = Green
		case "stopped", "exited":
			statusColor = Yellow
		case "error":
			statusColor = Red
		}
		
		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		
		name := c.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		
		image := c.Image
		if len(image) > 15 {
			image = image[:12] + "..."
		}
		
		fmt.Printf("%-12s %-20s %-15s %s%-12s%s %s\n", 
			shortID, name, image, statusColor, c.Status, Reset, c.Role)
	}
	fmt.Println()
}

func handleCreate(args []string) {
	checkAuth()
	
	// Parse flags
	name := ""
	image := "ubuntu:22.04"
	role := "default"
	memory := "512m"
	cpu := "0.5"
	
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "-n":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--image", "-i":
			if i+1 < len(args) {
				image = args[i+1]
				i++
			}
		case "--role", "-r":
			if i+1 < len(args) {
				role = args[i+1]
				i++
			}
		case "--memory", "-m":
			if i+1 < len(args) {
				memory = args[i+1]
				i++
			}
		case "--cpu", "-c":
			if i+1 < len(args) {
				cpu = args[i+1]
				i++
			}
		}
	}
	
	if name == "" {
		name = fmt.Sprintf("terminal-%d", time.Now().Unix())
	}
	
	fmt.Printf("%sCreating terminal...%s\n", Dim, Reset)
	
	body := map[string]interface{}{
		"name":   name,
		"image":  image,
		"role":   role,
		"memory": memory,
		"cpu":    cpu,
	}
	
	resp, err := apiRequest("POST", "/api/containers", body)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("%sError: %s%s\n", Red, errResp.Error, Reset)
		os.Exit(1)
	}
	
	var container Container
	json.NewDecoder(resp.Body).Decode(&container)
	
	fmt.Printf("\n%s✓ Terminal created%s\n", Green, Reset)
	fmt.Printf("  ID:    %s\n", container.ID[:12])
	fmt.Printf("  Name:  %s\n", container.Name)
	fmt.Printf("  Image: %s\n", container.Image)
	fmt.Printf("\nConnect with: %srexec connect %s%s\n\n", Cyan, container.ID[:12], Reset)
}

func handleConnect(args []string) {
	cfg := checkAuth()
	
	if len(args) == 0 {
		fmt.Printf("%sUsage: rexec connect <terminal-id>%s\n", Red, Reset)
		os.Exit(1)
	}
	
	terminalID := args[0]
	
	// Build WebSocket URL
	wsHost := strings.Replace(cfg.Host, "https://", "wss://", 1)
	wsHost = strings.Replace(wsHost, "http://", "ws://", 1)
	wsURL := fmt.Sprintf("%s/ws/terminal/%s?token=%s", wsHost, terminalID, url.QueryEscape(cfg.Token))
	
	fmt.Printf("%sConnecting to terminal %s...%s\n", Dim, terminalID, Reset)
	
	// Connect to WebSocket
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Printf("%sError connecting: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer conn.Close()
	
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("%sError setting raw mode: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	
	// Get terminal size
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	
	// Send initial resize
	resizeMsg := map[string]interface{}{
		"type": "resize",
		"cols": width,
		"rows": height,
	}
	resizeData, _ := json.Marshal(resizeMsg)
	conn.WriteMessage(websocket.TextMessage, resizeData)
	
	// Handle SIGWINCH for terminal resize
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			w, h, _ := term.GetSize(int(os.Stdout.Fd()))
			msg := map[string]interface{}{
				"type": "resize",
				"cols": w,
				"rows": h,
			}
			data, _ := json.Marshal(msg)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	}()
	
	// Handle interrupt
	intCh := make(chan os.Signal, 1)
	signal.Notify(intCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-intCh
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Println("\n\rDisconnected.")
		os.Exit(0)
	}()
	
	// Read from WebSocket and write to stdout
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), oldState)
				fmt.Println("\n\rConnection closed.")
				os.Exit(0)
			}
			os.Stdout.Write(message)
		}
	}()
	
	// Read from stdin and write to WebSocket
	buf := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}
		
		// Check for escape sequence (Ctrl+] to disconnect)
		if n == 1 && buf[0] == 0x1d {
			term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Println("\n\rDisconnected.")
			return
		}
		
		conn.WriteMessage(websocket.BinaryMessage, buf[:n])
	}
}

func handleStart(args []string) {
	checkAuth()
	
	if len(args) == 0 {
		fmt.Printf("%sUsage: rexec start <terminal-id>%s\n", Red, Reset)
		os.Exit(1)
	}
	
	resp, err := apiRequest("POST", "/api/containers/"+args[0]+"/start", nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		fmt.Printf("%s✓ Terminal started%s\n", Green, Reset)
	} else {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("%sError: %s%s\n", Red, errResp.Error, Reset)
	}
}

func handleStop(args []string) {
	checkAuth()
	
	if len(args) == 0 {
		fmt.Printf("%sUsage: rexec stop <terminal-id>%s\n", Red, Reset)
		os.Exit(1)
	}
	
	resp, err := apiRequest("POST", "/api/containers/"+args[0]+"/stop", nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		fmt.Printf("%s✓ Terminal stopped%s\n", Green, Reset)
	} else {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("%sError: %s%s\n", Red, errResp.Error, Reset)
	}
}

func handleDelete(args []string) {
	checkAuth()
	
	if len(args) == 0 {
		fmt.Printf("%sUsage: rexec delete <terminal-id>%s\n", Red, Reset)
		os.Exit(1)
	}
	
	// Confirm deletion
	fmt.Printf("%sAre you sure you want to delete terminal %s? (y/N): %s", Yellow, args[0], Reset)
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	
	if confirm != "y" && confirm != "yes" {
		fmt.Printf("%sCancelled%s\n", Dim, Reset)
		return
	}
	
	resp, err := apiRequest("DELETE", "/api/containers/"+args[0], nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		fmt.Printf("%s✓ Terminal deleted%s\n", Green, Reset)
	} else {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("%sError: %s%s\n", Red, errResp.Error, Reset)
	}
}

func handleSnippets(args []string) {
	checkAuth()
	
	if len(args) > 0 && args[0] == "marketplace" {
		// Show public snippets
		resp, err := apiRequest("GET", "/api/marketplace/snippets", nil)
		if err != nil {
			fmt.Printf("%sError: %v%s\n", Red, err, Reset)
			os.Exit(1)
		}
		defer resp.Body.Close()
		
		var result struct {
			Snippets []Snippet `json:"snippets"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		
		fmt.Printf("\n%s%sMarketplace Snippets%s\n", Bold, Cyan, Reset)
		fmt.Printf("─────────────────────────────────────────────────────────\n")
		
		for _, s := range result.Snippets {
			fmt.Printf("  %s%s%s - %s\n", Bold, s.Name, Reset, s.Description)
			fmt.Printf("    %s$ %s%s\n", Dim, s.Command, Reset)
		}
		fmt.Println()
		return
	}
	
	// Show user snippets
	resp, err := apiRequest("GET", "/api/snippets", nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var snippets []Snippet
	json.NewDecoder(resp.Body).Decode(&snippets)
	
	if len(snippets) == 0 {
		fmt.Printf("\n%sNo snippets found.%s\n", Dim, Reset)
		fmt.Printf("Browse marketplace: %srexec snippets marketplace%s\n\n", Cyan, Reset)
		return
	}
	
	fmt.Printf("\n%s%sYour Snippets%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────────────────────\n")
	
	for _, s := range snippets {
		visibility := Dim + "private" + Reset
		if s.IsPublic {
			visibility = Green + "public" + Reset
		}
		fmt.Printf("  %s%s%s [%s] - %s\n", Bold, s.Name, Reset, visibility, s.Description)
		fmt.Printf("    %s$ %s%s\n", Dim, s.Command, Reset)
	}
	fmt.Println()
}

func handleRun(args []string) {
	checkAuth()
	
	if len(args) == 0 {
		fmt.Printf("%sUsage: rexec run <snippet-name> --terminal <id>%s\n", Red, Reset)
		os.Exit(1)
	}
	
	snippetName := args[0]
	terminalID := ""
	
	for i := 1; i < len(args); i++ {
		if (args[i] == "--terminal" || args[i] == "-t") && i+1 < len(args) {
			terminalID = args[i+1]
			break
		}
	}
	
	if terminalID == "" {
		// List terminals and ask user to select
		resp, err := apiRequest("GET", "/api/containers", nil)
		if err != nil {
			fmt.Printf("%sError: %v%s\n", Red, err, Reset)
			os.Exit(1)
		}
		defer resp.Body.Close()
		
		var containers []Container
		json.NewDecoder(resp.Body).Decode(&containers)
		
		if len(containers) == 0 {
			fmt.Printf("%sNo terminals available. Create one first.%s\n", Red, Reset)
			os.Exit(1)
		}
		
		fmt.Printf("\n%sSelect terminal:%s\n", Bold, Reset)
		for i, c := range containers {
			fmt.Printf("  %d. %s (%s)\n", i+1, c.Name, c.ID[:12])
		}
		fmt.Print("\nEnter number: ")
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		idx, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || idx < 1 || idx > len(containers) {
			fmt.Printf("%sInvalid selection%s\n", Red, Reset)
			os.Exit(1)
		}
		
		terminalID = containers[idx-1].ID
	}
	
	// Find snippet
	resp, err := apiRequest("GET", "/api/snippets", nil)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var snippets []Snippet
	json.NewDecoder(resp.Body).Decode(&snippets)
	
	var snippet *Snippet
	for _, s := range snippets {
		if s.Name == snippetName || s.ID == snippetName {
			snippet = &s
			break
		}
	}
	
	if snippet == nil {
		// Check marketplace
		resp, err := apiRequest("GET", "/api/marketplace/snippets", nil)
		if err == nil {
			var result struct {
				Snippets []Snippet `json:"snippets"`
			}
			json.NewDecoder(resp.Body).Decode(&result)
			for _, s := range result.Snippets {
				if s.Name == snippetName || s.ID == snippetName {
					snippet = &s
					break
				}
			}
			resp.Body.Close()
		}
	}
	
	if snippet == nil {
		fmt.Printf("%sSnippet not found: %s%s\n", Red, snippetName, Reset)
		os.Exit(1)
	}
	
	fmt.Printf("%sRunning snippet '%s' on terminal %s...%s\n", Dim, snippet.Name, terminalID[:12], Reset)
	fmt.Printf("%s$ %s%s\n\n", Dim, snippet.Command, Reset)
	
	// Connect and run command
	cfg := loadConfig()
	wsHost := strings.Replace(cfg.Host, "https://", "wss://", 1)
	wsHost = strings.Replace(wsHost, "http://", "ws://", 1)
	wsURL := fmt.Sprintf("%s/ws/terminal/%s?token=%s", wsHost, terminalID, url.QueryEscape(cfg.Token))
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Printf("%sError connecting: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
	defer conn.Close()
	
	// Send command
	conn.WriteMessage(websocket.BinaryMessage, []byte(snippet.Command+"\n"))
	
	// Read output for a few seconds
	done := make(chan bool)
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			fmt.Print(string(message))
		}
		done <- true
	}()
	
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}
	
	fmt.Printf("\n%s✓ Command sent%s\n", Green, Reset)
}

func handleAgent(args []string) {
	if len(args) == 0 {
		fmt.Printf("%sUsage: rexec agent <register|start|stop|status>%s\n", Red, Reset)
		return
	}
	
	switch args[0] {
	case "register":
		handleAgentRegister(args[1:])
	case "start":
		handleAgentStart()
	case "stop":
		handleAgentStop()
	case "status":
		handleAgentStatus()
	default:
		fmt.Printf("%sUnknown agent command: %s%s\n", Red, args[0], Reset)
	}
}

func handleAgentRegister(args []string) {
	checkAuth()
	
	name := ""
	description := ""
	
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "-n":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--description", "-d":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		}
	}
	
	if name == "" {
		hostname, _ := os.Hostname()
		name = hostname
	}
	
	fmt.Printf("\n%s%sAgent Registration%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("This will register this machine as a rexec terminal.\n\n")
	fmt.Printf("  Name:        %s\n", name)
	fmt.Printf("  Description: %s\n", description)
	fmt.Printf("  OS:          %s/%s\n\n", runtime.GOOS, runtime.GOARCH)
	
	fmt.Print("Continue? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	
	if confirm != "y" && confirm != "yes" {
		fmt.Printf("%sCancelled%s\n", Dim, Reset)
		return
	}
	
	// Save agent config
	agentCfg := AgentConfig{
		Name:        name,
		Description: description,
	}
	
	home, _ := os.UserHomeDir()
	agentPath := filepath.Join(home, ConfigDir, "agent.json")
	data, _ := json.MarshalIndent(agentCfg, "", "  ")
	os.WriteFile(agentPath, data, 0600)
	
	fmt.Printf("\n%s✓ Agent registered%s\n", Green, Reset)
	fmt.Printf("Start the agent with: %srexec agent start%s\n\n", Cyan, Reset)
}

func handleAgentStart() {
	cfg := checkAuth()
	
	home, _ := os.UserHomeDir()
	agentPath := filepath.Join(home, ConfigDir, "agent.json")
	
	data, err := os.ReadFile(agentPath)
	if err != nil {
		fmt.Printf("%sAgent not registered. Run 'rexec agent register' first.%s\n", Red, Reset)
		return
	}
	
	var agentCfg AgentConfig
	json.Unmarshal(data, &agentCfg)
	
	fmt.Printf("\n%s%sStarting Rexec Agent%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("  Name: %s\n", agentCfg.Name)
	fmt.Printf("  Host: %s\n\n", cfg.Host)
	
	// This would normally start a background process that maintains
	// a reverse tunnel to rexec, but for now we'll just show what it would do
	fmt.Printf("%sAgent mode connects your machine to rexec as a terminal.\n", Dim)
	fmt.Printf("This allows remote terminal access through the rexec dashboard.%s\n\n", Reset)
	
	fmt.Printf("%s⚠ Agent daemon not yet implemented%s\n", Yellow, Reset)
	fmt.Printf("For now, use SSH tunneling:\n")
	fmt.Printf("  %sssh -R 0:localhost:22 tunnel@%s%s\n\n", Cyan, strings.TrimPrefix(cfg.Host, "https://"), Reset)
}

func handleAgentStop() {
	fmt.Printf("%s⚠ Agent daemon not running%s\n", Yellow, Reset)
}

func handleAgentStatus() {
	home, _ := os.UserHomeDir()
	agentPath := filepath.Join(home, ConfigDir, "agent.json")
	
	data, err := os.ReadFile(agentPath)
	if err != nil {
		fmt.Printf("%sAgent not registered.%s\n", Dim, Reset)
		return
	}
	
	var agentCfg AgentConfig
	json.Unmarshal(data, &agentCfg)
	
	fmt.Printf("\n%s%sAgent Status%s\n", Bold, Cyan, Reset)
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("  Registered: %s✓%s\n", Green, Reset)
	fmt.Printf("  Name:       %s\n", agentCfg.Name)
	fmt.Printf("  Running:    %sNo%s\n", Yellow, Reset)
	fmt.Println()
}

func handleDashboard() {
	handleInteractive()
}

func handleInteractive() {
	checkAuth()
	
	// Check if we have a TUI-capable terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Printf("%sDashboard requires an interactive terminal%s\n", Red, Reset)
		os.Exit(1)
	}
	
	// Try to run the TUI dashboard
	tuiPath := os.Getenv("REXEC_TUI_PATH")
	if tuiPath == "" {
		// Check common locations
		home, _ := os.UserHomeDir()
		paths := []string{
			filepath.Join(home, ".rexec", "rexec-tui"),
			"/usr/local/bin/rexec-tui",
			"rexec-tui",
		}
		for _, p := range paths {
			if _, err := exec.LookPath(p); err == nil {
				tuiPath = p
				break
			}
		}
	}
	
	if tuiPath != "" {
		cmd := exec.Command(tuiPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		return
	}
	
	// Fallback: simple ASCII dashboard
	showASCIIDashboard()
}

func showASCIIDashboard() {
	cfg := loadConfig()
	
	// Clear screen
	fmt.Print("\033[2J\033[H")
	
	// Header
	fmt.Printf("%s%s", BgCyan, Bold)
	fmt.Printf("                                                                               \n")
	fmt.Printf("  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗  Dashboard                          \n")
	fmt.Printf("  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝                                     \n")
	fmt.Printf("  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║                                          \n")
	fmt.Printf("  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║                                          \n")
	fmt.Printf("  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗                                     \n")
	fmt.Printf("  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝                                     \n")
	fmt.Printf("                                                                               \n")
	fmt.Printf("%s\n", Reset)
	
	// User info
	fmt.Printf("\n  %sLogged in as:%s %s (%s)\n", Dim, Reset, cfg.Username, cfg.Tier)
	
	// Fetch terminals
	resp, err := apiRequest("GET", "/api/containers", nil)
	if err != nil {
		fmt.Printf("\n  %sError loading terminals: %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()
	
	var containers []Container
	json.NewDecoder(resp.Body).Decode(&containers)
	
	// Terminals section
	fmt.Printf("\n  %s%sTerminals (%d)%s\n", Bold, Cyan, len(containers), Reset)
	fmt.Printf("  ─────────────────────────────────────────────────────────────────\n")
	
	if len(containers) == 0 {
		fmt.Printf("  %sNo terminals. Press 'c' to create one.%s\n", Dim, Reset)
	} else {
		for i, c := range containers {
			statusIcon := "○"
			statusColor := Dim
			switch c.Status {
			case "running":
				statusIcon = "●"
				statusColor = Green
			case "stopped", "exited":
				statusIcon = "○"
				statusColor = Yellow
			}
			
			shortID := c.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			
			fmt.Printf("  %s%s%s %d. %-20s %s%-10s%s [%s]\n", 
				statusColor, statusIcon, Reset,
				i+1, c.Name, statusColor, c.Status, Reset, shortID)
		}
	}
	
	// Help
	fmt.Printf("\n  %s%sControls%s\n", Bold, Cyan, Reset)
	fmt.Printf("  ─────────────────────────────────────────────────────────────────\n")
	fmt.Printf("  %s1-9%s  Connect to terminal    %sc%s  Create new    %sr%s  Refresh\n", Bold, Reset, Bold, Reset, Bold, Reset)
	fmt.Printf("  %ss%s    Snippets               %sq%s  Quit\n", Bold, Reset, Bold, Reset)
	fmt.Printf("\n  %sPress a key to continue...%s\n", Dim, Reset)
	
	// Wait for input
	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	
	buf := make([]byte, 1)
	os.Stdin.Read(buf)
	
	term.Restore(int(os.Stdin.Fd()), oldState)
	
	switch buf[0] {
	case 'q', 'Q', 0x03: // q or Ctrl+C
		fmt.Print("\033[2J\033[H")
		return
	case 'c', 'C':
		fmt.Print("\033[2J\033[H")
		handleCreate([]string{})
	case 'r', 'R':
		showASCIIDashboard()
	case 's', 'S':
		fmt.Print("\033[2J\033[H")
		handleSnippets([]string{})
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		idx := int(buf[0] - '1')
		if idx < len(containers) {
			fmt.Print("\033[2J\033[H")
			handleConnect([]string{containers[idx].ID})
		}
	default:
		showASCIIDashboard()
	}
}

func handleConfig(args []string) {
	cfg := loadConfig()
	
	if len(args) == 0 {
		// Show config
		fmt.Printf("\n%s%sConfiguration%s\n", Bold, Cyan, Reset)
		fmt.Printf("─────────────────────────────────────────\n")
		fmt.Printf("  Config file: %s\n", getConfigPath())
		fmt.Printf("  Host:        %s\n", cfg.Host)
		fmt.Printf("  Username:    %s\n", cfg.Username)
		fmt.Printf("  Email:       %s\n", cfg.Email)
		fmt.Printf("  Tier:        %s\n", cfg.Tier)
		if cfg.Token != "" {
			fmt.Printf("  Token:       %s...%s\n", cfg.Token[:20], cfg.Token[len(cfg.Token)-10:])
		}
		fmt.Println()
		return
	}
	
	// Set config
	if args[0] == "set" && len(args) >= 3 {
		key := args[1]
		value := args[2]
		
		switch key {
		case "host":
			cfg.Host = value
		default:
			fmt.Printf("%sUnknown config key: %s%s\n", Red, key, Reset)
			return
		}
		
		if err := saveConfig(cfg); err != nil {
			fmt.Printf("%sError saving config: %v%s\n", Red, err, Reset)
			return
		}
		
		fmt.Printf("%s✓ Config updated%s\n", Green, Reset)
	}
}
