package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// GuestAgentServer runs the guest agent server inside the VM
type GuestAgentServer struct {
	listenAddr string
	vsockPort  int
}

func main() {
	var (
		vsockPort  = flag.Int("port", 1234, "Vsock port to listen on")
		tcpPort    = flag.Int("tcp-port", 9000, "TCP port for fallback (when vsock not available)")
		listenAddr = flag.String("listen", "", "Address to listen on (vsock or tcp)")
	)
	flag.Parse()
	_ = tcpPort // reserved for TCP fallback when vsock not available

	server := &GuestAgentServer{
		vsockPort: *vsockPort,
	}

	// Determine listen address
	if *listenAddr != "" {
		server.listenAddr = *listenAddr
	} else {
		// Try vsock first, fallback to TCP
		server.listenAddr = fmt.Sprintf("vsock://3:%d", *vsockPort)
	}

	log.Printf("[GuestAgent] Starting guest agent on %s", server.listenAddr)
	
	if err := server.Run(); err != nil {
		log.Fatalf("[GuestAgent] Failed to start: %v", err)
	}
}

// Run starts the guest agent server
func (s *GuestAgentServer) Run() error {
	// Listen on vsock or TCP
	// For now, we'll use TCP as a fallback since vsock requires special handling
	listener, err := s.listen()
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	log.Printf("[GuestAgent] Listening on %s", s.listenAddr)

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[GuestAgent] Shutting down...")
		listener.Close()
	}()

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[GuestAgent] Accept error: %v", err)
			return err
		}

		go s.handleConnection(conn)
	}
}

// listen creates a listener (vsock or TCP)
func (s *GuestAgentServer) listen() (net.Listener, error) {
	// For now, use TCP as fallback
	// In production, this would use vsock
	tcpAddr := fmt.Sprintf(":%d", 9000+s.vsockPort)
	return net.Listen("tcp", tcpAddr)
}

// handleConnection handles a client connection
func (s *GuestAgentServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf("[GuestAgent] New connection from %s", conn.RemoteAddr())

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req GuestAgentRequest
		if err := decoder.Decode(&req); err != nil {
			if err != io.EOF {
				log.Printf("[GuestAgent] Decode error: %v", err)
			}
			return
		}

		// Handle request
		resp := s.handleRequest(&req)

		// Send response
		if err := encoder.Encode(resp); err != nil {
			log.Printf("[GuestAgent] Encode error: %v", err)
			return
		}
	}
}

// GuestAgentRequest matches the client's request structure
type GuestAgentRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// GuestAgentResponse matches the client's response structure
type GuestAgentResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *GuestAgentError `json:"error,omitempty"`
}

// GuestAgentError represents an error response
type GuestAgentError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// handleRequest processes a JSON-RPC request
func (s *GuestAgentServer) handleRequest(req *GuestAgentRequest) *GuestAgentResponse {
	resp := &GuestAgentResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "exec":
		result := s.handleExec(req.Params)
		resultJSON, _ := json.Marshal(result)
		resp.Result = resultJSON

	case "copy_to":
		result := s.handleCopyTo(req.Params)
		resultJSON, _ := json.Marshal(result)
		resp.Result = resultJSON

	case "copy_from":
		result := s.handleCopyFrom(req.Params)
		resultJSON, _ := json.Marshal(result)
		resp.Result = resultJSON

	case "metrics":
		result := s.handleMetrics()
		resultJSON, _ := json.Marshal(result)
		resp.Result = resultJSON

	case "shell":
		// Shell requires persistent connection, handled separately
		resp.Error = &GuestAgentError{
			Code:    -32601,
			Message: "shell method requires persistent connection",
		}

	default:
		resp.Error = &GuestAgentError{
			Code:    -32601,
			Message: fmt.Sprintf("unknown method: %s", req.Method),
		}
	}

	return resp
}

// ExecParams matches client structure
type ExecParams struct {
	Command []string          `json:"command"`
	Timeout int               `json:"timeout,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// ExecResult matches client structure
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Duration int64  `json:"duration_ms"`
}

// handleExec executes a command
func (s *GuestAgentServer) handleExec(params json.RawMessage) *ExecResult {
	var p ExecParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &ExecResult{
			ExitCode: -1,
			Stderr:   fmt.Sprintf("invalid params: %v", err),
		}
	}

	if len(p.Command) == 0 {
		return &ExecResult{
			ExitCode: -1,
			Stderr:   "command is required",
		}
	}

	start := time.Now()

	// Create command
	cmd := exec.Command(p.Command[0], p.Command[1:]...)

	// Set environment variables
	if len(p.Env) > 0 {
		env := os.Environ()
		for k, v := range p.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Capture output
	stdout, err := cmd.Output()
	stderr := ""
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
	}

	duration := time.Since(start).Milliseconds()

	return &ExecResult{
		ExitCode: cmd.ProcessState.ExitCode(),
		Stdout:   string(stdout),
		Stderr:   stderr,
		Duration: duration,
	}
}

// CopyParams matches client structure
type CopyParams struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Recursive   bool   `json:"recursive,omitempty"`
}

// CopyResult matches client structure
type CopyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Bytes   int64  `json:"bytes,omitempty"`
}

// handleCopyTo copies file from host to guest (placeholder)
func (s *GuestAgentServer) handleCopyTo(params json.RawMessage) *CopyResult {
	var p CopyParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &CopyResult{
			Success: false,
			Message: fmt.Sprintf("invalid params: %v", err),
		}
	}

	// TODO: Implement file copy
	return &CopyResult{
		Success: false,
		Message: "copy_to not yet implemented",
	}
}

// handleCopyFrom copies file from guest to host (placeholder)
func (s *GuestAgentServer) handleCopyFrom(params json.RawMessage) *CopyResult {
	var p CopyParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &CopyResult{
			Success: false,
			Message: fmt.Sprintf("invalid params: %v", err),
		}
	}

	// TODO: Implement file copy
	return &CopyResult{
		Success: false,
		Message: "copy_from not yet implemented",
	}
}

// MetricsResult matches client structure
type MetricsResult struct {
	CPU    CPUMetrics    `json:"cpu"`
	Memory MemoryMetrics `json:"memory"`
	Disk   DiskMetrics   `json:"disk"`
	Network NetworkMetrics `json:"network"`
}

// CPUMetrics, MemoryMetrics, etc. match client structures
type CPUMetrics struct {
	Percent float64 `json:"percent"`
	Count   int     `json:"count"`
}

type MemoryMetrics struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
}

type DiskMetrics struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
}

type NetworkMetrics struct {
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
}

// handleMetrics returns VM metrics
func (s *GuestAgentServer) handleMetrics() *MetricsResult {
	// Read /proc/stat for CPU
	// Read /proc/meminfo for memory
	// Read /proc/diskstats for disk
	// Read /proc/net/dev for network

	// Simplified implementation - in production would read actual system stats
	return &MetricsResult{
		CPU: CPUMetrics{
			Percent: 0.0,
			Count:   1,
		},
		Memory: MemoryMetrics{
			Used:    0,
			Total:   0,
			Percent: 0.0,
		},
		Disk: DiskMetrics{
			Used:    0,
			Total:   0,
			Percent: 0.0,
		},
		Network: NetworkMetrics{
			RxBytes: 0,
			TxBytes: 0,
		},
	}
}
