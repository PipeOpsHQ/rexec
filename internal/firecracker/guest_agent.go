package firecracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// GuestAgentClient handles communication with the guest agent via vsock
type GuestAgentClient struct {
	conn   net.Conn
	reqID  int64
	mu     sync.Mutex
}

// GuestAgentRequest represents a JSON-RPC request
type GuestAgentRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// GuestAgentResponse represents a JSON-RPC response
type GuestAgentResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *GuestAgentError `json:"error,omitempty"`
}

// GuestAgentError represents a JSON-RPC error
type GuestAgentError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ExecParams holds parameters for exec command
type ExecParams struct {
	Command []string `json:"command"`
	Timeout int      `json:"timeout,omitempty"` // seconds
	Env     map[string]string `json:"env,omitempty"`
}

// ExecResult holds the result of exec command
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Duration int64  `json:"duration_ms"`
}

// CopyParams holds parameters for copy operations
type CopyParams struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Recursive   bool   `json:"recursive,omitempty"`
}

// CopyResult holds the result of copy operation
type CopyResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Bytes   int64  `json:"bytes,omitempty"`
}

// MetricsResult holds VM metrics
type MetricsResult struct {
	CPU    CPUMetrics    `json:"cpu"`
	Memory MemoryMetrics `json:"memory"`
	Disk   DiskMetrics   `json:"disk"`
	Network NetworkMetrics `json:"network"`
}

// CPUMetrics holds CPU usage metrics
type CPUMetrics struct {
	Percent float64 `json:"percent"`
	Count   int     `json:"count"`
}

// MemoryMetrics holds memory usage metrics
type MemoryMetrics struct {
	Used  int64 `json:"used"`  // bytes
	Total int64 `json:"total"` // bytes
	Percent float64 `json:"percent"`
}

// DiskMetrics holds disk usage metrics
type DiskMetrics struct {
	Used  int64 `json:"used"`  // bytes
	Total int64 `json:"total"` // bytes
	Percent float64 `json:"percent"`
}

// NetworkMetrics holds network usage metrics
type NetworkMetrics struct {
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
}

// NewGuestAgentClient creates a new guest agent client
// vsockPath is the vsock address (e.g., "vsock://3:1234" or just "3:1234")
func NewGuestAgentClient(ctx context.Context, vsockPath string) (*GuestAgentClient, error) {
	// Parse vsock address
	// Format: "vsock://CID:PORT" or "CID:PORT"
	// For Firecracker, CID is typically 3 (guest), PORT is configurable
	cid, port, err := parseVsockAddress(vsockPath)
	if err != nil {
		return nil, fmt.Errorf("invalid vsock address: %w", err)
	}

	// Connect via vsock
	// Note: Go doesn't have native vsock support, so we'll use a workaround
	// For now, we'll use a TCP connection to a proxy or implement vsock via syscalls
	// This is a placeholder - actual vsock requires Linux-specific code or a library
	conn, err := dialVsock(ctx, cid, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guest agent: %w", err)
	}

	return &GuestAgentClient{
		conn:  conn,
		reqID: 1,
	}, nil
}

// parseVsockAddress parses a vsock address string
func parseVsockAddress(addr string) (cid, port uint32, err error) {
	// Remove vsock:// prefix if present
	if len(addr) > 8 && addr[:8] == "vsock://" {
		addr = addr[8:]
	}

	// Parse CID:PORT format
	var cidVal, portVal uint32
	_, err = fmt.Sscanf(addr, "%d:%d", &cidVal, &portVal)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid vsock address format: %w", err)
	}

	return cidVal, portVal, nil
}

// dialVsock creates a vsock connection
// This is a placeholder - actual implementation requires vsock support
// Options:
// 1. Use github.com/mdlayher/vsock package
// 2. Use a TCP proxy in the guest
// 3. Use Firecracker's MMDS or other communication channel
func dialVsock(ctx context.Context, cid, port uint32) (net.Conn, error) {
	// TODO: Implement actual vsock dialing
	// For now, return an error indicating vsock is not yet implemented
	// In production, this would use the vsock package or a TCP fallback
	
	// Placeholder: Try TCP fallback if vsock is not available
	// Guest agent could listen on both vsock and a TCP port forwarded via Firecracker
	tcpAddr := fmt.Sprintf("127.0.0.1:%d", 9000+port) // Use port mapping
	conn, err := net.DialTimeout("tcp", tcpAddr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("vsock not implemented, TCP fallback failed: %w", err)
	}
	
	return conn, nil
}

// Call sends a JSON-RPC request and waits for response
func (c *GuestAgentClient) Call(ctx context.Context, method string, params interface{}) (*GuestAgentResponse, error) {
	c.mu.Lock()
	reqID := c.reqID
	c.reqID++
	c.mu.Unlock()

	req := GuestAgentRequest{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  params,
	}

	// Encode request
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send request (with newline delimiter)
	reqData = append(reqData, '\n')
	if _, err := c.conn.Write(reqData); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response (line-delimited JSON)
	decoder := json.NewDecoder(c.conn)
	var resp GuestAgentResponse
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for JSON-RPC error
	if resp.Error != nil {
		return nil, fmt.Errorf("guest agent error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
	}

	// Verify response ID matches request
	if resp.ID != reqID {
		return nil, fmt.Errorf("response ID mismatch: expected %d, got %d", reqID, resp.ID)
	}

	return &resp, nil
}

// Exec executes a command in the guest
func (c *GuestAgentClient) Exec(ctx context.Context, cmd []string, timeout int) (*ExecResult, error) {
	params := ExecParams{
		Command: cmd,
		Timeout: timeout,
	}

	resp, err := c.Call(ctx, "exec", params)
	if err != nil {
		return nil, err
	}

	var result ExecResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exec result: %w", err)
	}

	return &result, nil
}

// CopyTo copies a file from host to guest
func (c *GuestAgentClient) CopyTo(ctx context.Context, source, destination string, recursive bool) (*CopyResult, error) {
	params := CopyParams{
		Source:      source,
		Destination: destination,
		Recursive:   recursive,
	}

	resp, err := c.Call(ctx, "copy_to", params)
	if err != nil {
		return nil, err
	}

	var result CopyResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal copy result: %w", err)
	}

	return &result, nil
}

// CopyFrom copies a file from guest to host
func (c *GuestAgentClient) CopyFrom(ctx context.Context, source, destination string, recursive bool) (*CopyResult, error) {
	params := CopyParams{
		Source:      source,
		Destination: destination,
		Recursive:   recursive,
	}

	resp, err := c.Call(ctx, "copy_from", params)
	if err != nil {
		return nil, err
	}

	var result CopyResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal copy result: %w", err)
	}

	return &result, nil
}

// GetMetrics retrieves VM metrics from guest
func (c *GuestAgentClient) GetMetrics(ctx context.Context) (*MetricsResult, error) {
	resp, err := c.Call(ctx, "metrics", nil)
	if err != nil {
		return nil, err
	}

	var result MetricsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics result: %w", err)
	}

	return &result, nil
}

// Shell opens an interactive shell session
// Returns a connection for bidirectional communication
func (c *GuestAgentClient) Shell(ctx context.Context, shell string, cols, rows uint16) (io.ReadWriteCloser, error) {
	params := map[string]interface{}{
		"shell": shell,
		"cols":   cols,
		"rows":   rows,
	}

	_, err := c.Call(ctx, "shell", params)
	if err != nil {
		return nil, err
	}

	// For shell, we need a persistent connection
	// This is a simplified version - actual implementation would need
	// a separate connection or stream handling
	return c.conn, nil
}

// Close closes the guest agent connection
func (c *GuestAgentClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
