package firecracker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/rexec/rexec/internal/providers"
)

const (
	ProviderName = "firecracker"
	DefaultKernelPath = "/opt/firecracker/vmlinux.bin"
	DefaultRootfsPath = "/var/lib/rexec/firecracker/rootfs"
	DefaultSocketPath = "/tmp/firecracker-%s.socket"
	DefaultBridgeName = "rexec-bridge"
)

// Manager handles Firecracker microVM lifecycle
type Manager struct {
	kernelPath    string
	rootfsPath    string
	socketBaseDir string
	bridgeName    string
	vms           map[string]*VMInfo
	clients       map[string]*FirecrackerClient // vmID -> client
	mu            sync.RWMutex
	networkMgr    *NetworkManager
	storageMgr    *StorageManager
}

// VMInfo holds information about a microVM
type VMInfo struct {
	ID            string
	UserID        string
	VMName        string
	ImageType     string
	Status        string // "creating", "running", "stopped", "error"
	IPAddress     string
	CreatedAt     time.Time
	LastUsedAt    time.Time
	Labels        map[string]string
	FirecrackerID string // Firecracker's internal VM ID
	SocketPath    string
	TapDevice     string
}

// NewManager creates a new Firecracker manager
func NewManager() (*Manager, error) {
	kernelPath := os.Getenv("FIRECRACKER_KERNEL_PATH")
	if kernelPath == "" {
		kernelPath = DefaultKernelPath
	}

	// Check if kernel exists
	if _, err := os.Stat(kernelPath); err != nil {
		log.Printf("[Firecracker] Warning: Kernel not found at %s: %v", kernelPath, err)
		log.Printf("[Firecracker] Set FIRECRACKER_KERNEL_PATH environment variable to specify kernel path")
	}

	rootfsPath := os.Getenv("FIRECRACKER_ROOTFS_PATH")
	if rootfsPath == "" {
		rootfsPath = DefaultRootfsPath
	}

	socketBaseDir := os.Getenv("FIRECRACKER_SOCKET_DIR")
	if socketBaseDir == "" {
		socketBaseDir = "/tmp/firecracker"
	}

	bridgeName := os.Getenv("FIRECRACKER_BRIDGE_NAME")
	if bridgeName == "" {
		bridgeName = DefaultBridgeName
	}

	// Ensure directories exist
	if err := os.MkdirAll(socketBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	// Initialize network manager
	networkMgr, err := NewNetworkManager(bridgeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create network manager: %w", err)
	}

	// Initialize storage manager
	storageMgr, err := NewStorageManager(rootfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage manager: %w", err)
	}

	mgr := &Manager{
		kernelPath:    kernelPath,
		rootfsPath:    rootfsPath,
		socketBaseDir: socketBaseDir,
		bridgeName:    bridgeName,
		vms:           make(map[string]*VMInfo),
		clients:       make(map[string]*FirecrackerClient),
		networkMgr:    networkMgr,
		storageMgr:    storageMgr,
	}

	// Ensure bridge exists
	if err := mgr.networkMgr.EnsureBridge(context.Background()); err != nil {
		log.Printf("[Firecracker] Warning: failed to ensure bridge exists: %v", err)
	}

	return mgr, nil
}

// Name returns the provider name
func (m *Manager) Name() string {
	return ProviderName
}

// IsAvailable checks if Firecracker is available
func (m *Manager) IsAvailable(ctx context.Context) bool {
	// Check if Firecracker binary exists
	firecrackerPath := os.Getenv("FIRECRACKER_BINARY_PATH")
	if firecrackerPath == "" {
		firecrackerPath = "firecracker"
	}

	cmd := exec.CommandContext(ctx, "which", firecrackerPath)
	if err := cmd.Run(); err != nil {
		return false
	}

	// Check if kernel exists
	if _, err := os.Stat(m.kernelPath); err != nil {
		return false
	}

	// Check if KVM is available
	if _, err := os.Stat("/dev/kvm"); err != nil {
		return false
	}

	return true
}

// Create creates a new microVM
func (m *Manager) Create(ctx context.Context, cfg providers.CreateConfig) (*providers.TerminalInfo, error) {
	vmID := fmt.Sprintf("vm-%s-%s", cfg.UserID, cfg.Name)
	socketPath := filepath.Join(m.socketBaseDir, fmt.Sprintf("%s.socket", vmID))

	// Check if VM already exists
	m.mu.RLock()
	if _, exists := m.vms[vmID]; exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("VM %s already exists", vmID)
	}
	m.mu.RUnlock()

	// Get rootfs path for this image type
	rootfsImage, err := m.storageMgr.GetRootfsPath(ctx, cfg.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to get rootfs path: %w", err)
	}

	// Create tap device
	tapDevice, err := m.networkMgr.CreateTapDevice(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to create tap device: %w", err)
	}

	// Create VM info
	now := time.Now()
	vmInfo := &VMInfo{
		ID:            vmID,
		UserID:        cfg.UserID,
		VMName:        cfg.Name,
		ImageType:     cfg.Image,
		Status:        "creating",
		CreatedAt:     now,
		LastUsedAt:    now,
		Labels:        cfg.Labels,
		FirecrackerID: vmID,
		SocketPath:    socketPath,
		TapDevice:     tapDevice,
	}

	// Generate MAC address for the VM (use deterministic generation based on VM ID)
	macAddress := generateMACAddress(vmID)

	// Create VM configuration
	vmConfig := &VMConfig{
		VMName:     vmID,
		VCPUs:      cfg.CPUShares / 1000, // Convert millicores to vCPUs
		MemoryMB:   cfg.MemoryMB,
		KernelArgs: "console=ttyS0 reboot=k panic=1 pci=off",
		RootfsPath: rootfsImage,
		KernelPath: m.kernelPath,
		Network: NetworkConfig{
			IfaceID:     "eth0",
			GuestMAC:   macAddress,
			HostDevName: tapDevice,
			AllowMMDS:  false,
		},
	}

	// Ensure at least 1 vCPU
	if vmConfig.VCPUs < 1 {
		vmConfig.VCPUs = 1
	}

	// Start Firecracker process and configure VM
	client, err := StartFirecrackerProcess(ctx, socketPath, m.kernelPath, rootfsImage, vmConfig)
	if err != nil {
		// Cleanup on failure
		m.networkMgr.DeleteTapDevice(ctx, tapDevice)
		return nil, fmt.Errorf("failed to start firecracker: %w", err)
	}

	// Start the VM
	if err := client.StartVM(ctx); err != nil {
		client.Close()
		m.networkMgr.DeleteTapDevice(ctx, tapDevice)
		return nil, fmt.Errorf("failed to start VM: %w", err)
	}

	// Store client and update status
	m.mu.Lock()
	m.vms[vmID] = vmInfo
	m.clients[vmID] = client
	vmInfo.Status = "running"
	m.mu.Unlock()

	// Get IP address (will be assigned by DHCP, may take a moment)
	// Try a few times with delay
	for i := 0; i < 5; i++ {
		time.Sleep(500 * time.Millisecond)
		ipAddress, err := m.networkMgr.GetVMIP(ctx, tapDevice)
		if err == nil && ipAddress != "" {
			vmInfo.IPAddress = ipAddress
			break
		}
	}

	return m.toTerminalInfo(vmInfo), nil
}

// Start starts a stopped VM
func (m *Manager) Start(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vm, ok := m.vms[id]
	if !ok {
		return fmt.Errorf("VM %s not found", id)
	}

	if vm.Status == "running" {
		return nil // Already running
	}

	// Get or create client
	client, ok := m.clients[id]
	if !ok {
		// VM was stopped, need to recreate Firecracker process
		// Get rootfs path
		rootfsImage, err := m.storageMgr.GetRootfsPath(ctx, vm.ImageType)
		if err != nil {
			return fmt.Errorf("failed to get rootfs path: %w", err)
		}

		// Recreate VM config
		macAddress := generateMACAddress(id)
		vmConfig := &VMConfig{
			VMName:     id,
			VCPUs:      1, // Will be updated from labels if available
			MemoryMB:   512, // Will be updated from labels if available
			KernelArgs: "console=ttyS0 reboot=k panic=1 pci=off",
			RootfsPath: rootfsImage,
			KernelPath: m.kernelPath,
			Network: NetworkConfig{
				IfaceID:     "eth0",
				GuestMAC:   macAddress,
				HostDevName: vm.TapDevice,
				AllowMMDS:  false,
			},
		}

		// Parse resource limits from labels if available
		if vm.Labels != nil {
			if memStr, ok := vm.Labels["rexec.memory_mb"]; ok {
				if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
					vmConfig.MemoryMB = mem
				}
			}
			if cpuStr, ok := vm.Labels["rexec.cpu_shares"]; ok {
				if cpu, err := strconv.ParseInt(cpuStr, 10, 64); err == nil {
					vmConfig.VCPUs = cpu / 1000
					if vmConfig.VCPUs < 1 {
						vmConfig.VCPUs = 1
					}
				}
		}
	}

		client, err = StartFirecrackerProcess(ctx, vm.SocketPath, m.kernelPath, rootfsImage, vmConfig)
		if err != nil {
			return fmt.Errorf("failed to start firecracker: %w", err)
		}

		if err := client.StartVM(ctx); err != nil {
			client.Close()
			return fmt.Errorf("failed to start VM: %w", err)
		}

		m.clients[id] = client
	} else {
		// Client exists, just start the VM
		if err := client.StartVM(ctx); err != nil {
			return fmt.Errorf("failed to start VM: %w", err)
		}
	}

	vm.Status = "running"
	vm.LastUsedAt = time.Now()

	return nil
}

// Stop stops a running VM
func (m *Manager) Stop(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vm, ok := m.vms[id]
	if !ok {
		return fmt.Errorf("VM %s not found", id)
	}

	if vm.Status == "stopped" {
		return nil // Already stopped
	}

	// Stop via Firecracker API
	client, ok := m.clients[id]
	if ok {
		if err := client.StopVM(ctx); err != nil {
			log.Printf("[Firecracker] Failed to stop VM %s: %v", id, err)
		}
		// Close client (stops process)
		client.Close()
		delete(m.clients, id)
	}

	vm.Status = "stopped"

	return nil
}

// Delete removes a VM
func (m *Manager) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	vm, ok := m.vms[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("VM %s not found", id)
	}

	// Get client before deleting from map
	client, hasClient := m.clients[id]
	delete(m.vms, id)
	delete(m.clients, id)
	m.mu.Unlock()

	// Stop VM if running
	if vm.Status == "running" && hasClient {
		if err := client.StopVM(ctx); err != nil {
			log.Printf("[Firecracker] Warning: failed to stop VM %s: %v", id, err)
		}
		client.Close()
	}

	// Clean up tap device
	if vm.TapDevice != "" {
		if err := m.networkMgr.DeleteTapDevice(ctx, vm.TapDevice); err != nil {
			log.Printf("[Firecracker] Warning: failed to delete tap device %s: %v", vm.TapDevice, err)
		}
	}

	// Clean up socket
	if vm.SocketPath != "" {
		if err := os.Remove(vm.SocketPath); err != nil && !os.IsNotExist(err) {
			log.Printf("[Firecracker] Warning: failed to remove socket %s: %v", vm.SocketPath, err)
		}
	}

	return nil
}

// Get retrieves VM information
func (m *Manager) Get(ctx context.Context, id string) (*providers.TerminalInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vm, ok := m.vms[id]
	if !ok {
		return nil, fmt.Errorf("VM %s not found", id)
	}

	return m.toTerminalInfo(vm), nil
}

// List returns all VMs for a user
func (m *Manager) List(ctx context.Context, userID string) ([]*providers.TerminalInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*providers.TerminalInfo
	for _, vm := range m.vms {
		if vm.UserID == userID {
			result = append(result, m.toTerminalInfo(vm))
		}
	}

	return result, nil
}

// ConnectTerminal establishes a terminal connection
func (m *Manager) ConnectTerminal(ctx context.Context, id string, cols, rows uint16) (*providers.TerminalConnection, error) {
	m.mu.RLock()
	vm, ok := m.vms[id]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("VM %s not found", id)
	}

	if vm.Status != "running" {
		return nil, fmt.Errorf("VM %s is not running", id)
	}

	// Connect to guest agent
	vsockAddr := fmt.Sprintf("vsock://3:%d", 1234) // CID 3 = guest, port configurable
	agent, err := NewGuestAgentClient(ctx, vsockAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guest agent: %w", err)
	}

	// Open shell session
	shell := "/bin/bash" // Default shell
	if vm.Labels != nil {
		if s, ok := vm.Labels["rexec.shell"]; ok {
			shell = s
		}
	}

	conn, err := agent.Shell(ctx, shell, cols, rows)
	if err != nil {
		agent.Close()
		return nil, fmt.Errorf("failed to open shell: %w", err)
	}

	return &providers.TerminalConnection{
		ID:       id,
		Provider: ProviderName,
		Reader:   conn,
		Writer:   conn,
		Resize: func(c, r uint16) error {
			// TODO: Send resize command to guest agent
			return nil
		},
		Close: func() error {
			return agent.Close()
		},
	}, nil
}

// Exec executes a command in the VM
func (m *Manager) Exec(ctx context.Context, id string, cmd []string) ([]byte, error) {
	m.mu.RLock()
	_, ok := m.vms[id]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("VM %s not found", id)
	}

	// Connect to guest agent
	// For now, use a placeholder vsock address
	// In production, this would be configured per VM
	vsockAddr := fmt.Sprintf("vsock://3:%d", 1234) // CID 3 = guest, port configurable
	agent, err := NewGuestAgentClient(ctx, vsockAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guest agent: %w", err)
	}
	defer agent.Close()

	// Execute command
	result, err := agent.Exec(ctx, cmd, 30)
	if err != nil {
		return nil, err
	}

	// Return stdout (or stderr if exit code != 0)
	if result.ExitCode != 0 {
		return []byte(result.Stderr), fmt.Errorf("command failed with exit code %d", result.ExitCode)
	}

	return []byte(result.Stdout), nil
}

// GetStats retrieves resource usage statistics
func (m *Manager) GetStats(ctx context.Context, id string) (*providers.ResourceStats, error) {
	m.mu.RLock()
	vm, ok := m.vms[id]
	client, hasClient := m.clients[id]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("VM %s not found", id)
	}

	// Try to get stats from guest agent first (more accurate)
	vsockAddr := fmt.Sprintf("vsock://3:%d", 1234)
	agent, err := NewGuestAgentClient(ctx, vsockAddr)
	if err == nil {
		defer agent.Close()
		metrics, err := agent.GetMetrics(ctx)
		if err == nil {
			return &providers.ResourceStats{
				CPUPercent:  metrics.CPU.Percent,
				Memory:      metrics.Memory.Used,
				MemoryLimit: metrics.Memory.Total,
				DiskUsage:   metrics.Disk.Used,
				DiskLimit:   metrics.Disk.Total,
				NetRx:       metrics.Network.RxBytes,
				NetTx:       metrics.Network.TxBytes,
			}, nil
		}
	}

	// Fallback: Get basic info from Firecracker API
	if hasClient {
		_, err := client.GetInstanceInfo(ctx)
		if err == nil && vm.Labels != nil {
			memMB := int64(0)
			if s, ok := vm.Labels["rexec.memory_mb"]; ok {
				memMB, _ = strconv.ParseInt(s, 10, 64)
			}
			diskMB := int64(0)
			if s, ok := vm.Labels["rexec.disk_mb"]; ok {
				diskMB, _ = strconv.ParseInt(s, 10, 64)
			}
			return &providers.ResourceStats{
				CPUPercent:  0,
				Memory:      0,
				MemoryLimit: memMB * 1024 * 1024,
				DiskUsage:   0,
				DiskLimit:   diskMB * 1024 * 1024,
				NetRx:       0,
				NetTx:       0,
			}, nil
		}
	}

	// Return empty stats if all methods fail
	return &providers.ResourceStats{}, nil
}

// StreamStats streams resource usage statistics
func (m *Manager) StreamStats(ctx context.Context, id string) (<-chan *providers.ResourceStats, error) {
	// TODO: Implement stats streaming
	ch := make(chan *providers.ResourceStats)
	close(ch)
	return ch, fmt.Errorf("stats streaming not yet implemented")
}

// toTerminalInfo converts VMInfo to TerminalInfo
func (m *Manager) toTerminalInfo(vm *VMInfo) *providers.TerminalInfo {
	return &providers.TerminalInfo{
		ID:         vm.ID,
		UserID:     vm.UserID,
		Name:       vm.VMName,
		Provider:   ProviderName,
		Status:     vm.Status,
		IPAddress:  vm.IPAddress,
		CreatedAt:  vm.CreatedAt.Unix(),
		LastUsedAt: vm.LastUsedAt.Unix(),
		Labels:     vm.Labels,
	}
}
