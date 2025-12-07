package handlers

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// SSHHandler handles SSH key management API endpoints
type SSHHandler struct {
	store            *storage.PostgresStore
	containerManager *container.Manager
}

// NewSSHHandler creates a new SSH handler
func NewSSHHandler(store *storage.PostgresStore, containerManager *container.Manager) *SSHHandler {
	return &SSHHandler{
		store:            store,
		containerManager: containerManager,
	}
}

// AddSSHKeyRequest represents the request to add an SSH key
type AddSSHKeyRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=255"`
	PublicKey string `json:"public_key" binding:"required"`
}

// SSHKeyResponse represents an SSH key in API responses
type SSHKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Fingerprint string     `json:"fingerprint"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// AddRemoteHostRequest represents the request to add a remote host
type AddRemoteHostRequest struct {
	Name         string `json:"name" binding:"required,min=1,max=255"`
	Hostname     string `json:"hostname" binding:"required"`
	Port         int    `json:"port"`
	Username     string `json:"username" binding:"required"`
	IdentityFile string `json:"identity_file"`
}

// RemoteHostResponse represents a remote host in API responses
type RemoteHostResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Hostname     string    `json:"hostname"`
	Port         int       `json:"port"`
	Username     string    `json:"username"`
	IdentityFile string    `json:"identity_file,omitempty"`
	SSHCommand   string    `json:"ssh_command"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListSSHKeys returns all SSH keys for the authenticated user
// GET /api/ssh/keys
func (h *SSHHandler) ListSSHKeys(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keys, err := h.store.GetSSHKeysByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch SSH keys"})
		return
	}

	response := make([]SSHKeyResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, SSHKeyResponse{
			ID:          key.ID,
			Name:        key.Name,
			Fingerprint: key.Fingerprint,
			CreatedAt:   key.CreatedAt,
			LastUsedAt:  key.LastUsedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"keys":  response,
		"count": len(response),
	})
}

// AddSSHKey adds a new SSH key for the authenticated user
// POST /api/ssh/keys
func (h *SSHHandler) AddSSHKey(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req AddSSHKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate and parse the public key
	publicKey := strings.TrimSpace(req.PublicKey)
	if err := validateSSHPublicKey(publicKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate fingerprint
	fingerprint, err := calculateSSHFingerprint(publicKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to calculate key fingerprint"})
		return
	}

	ctx := c.Request.Context()

	// Check if key with same fingerprint already exists for this user
	existingKeys, err := h.store.GetSSHKeysByUserID(ctx, userID)
	if err == nil {
		for _, key := range existingKeys {
			if key.Fingerprint == fingerprint {
				c.JSON(http.StatusConflict, gin.H{
					"error":       "SSH key already exists",
					"fingerprint": fingerprint,
					"name":        key.Name,
				})
				return
			}
		}
	}

	// Check key limit (max 10 keys per user)
	if len(existingKeys) >= 10 {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "SSH key limit reached",
			"limit":   10,
			"current": len(existingKeys),
		})
		return
	}

	// Create the SSH key record
	keyRecord := &storage.SSHKeyRecord{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now(),
	}

	if err := h.store.CreateSSHKey(ctx, keyRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save SSH key"})
		return
	}

	// Sync keys to all user's running containers
	go h.syncKeysToUserContainers(userID)

	c.JSON(http.StatusCreated, gin.H{
		"id":          keyRecord.ID,
		"name":        keyRecord.Name,
		"fingerprint": keyRecord.Fingerprint,
		"created_at":  keyRecord.CreatedAt,
		"message":     "SSH key added successfully",
	})
}

// DeleteSSHKey removes an SSH key
// DELETE /api/ssh/keys/:id
func (h *SSHHandler) DeleteSSHKey(c *gin.Context) {
	userID := c.GetString("userID")
	keyID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// Verify the key belongs to the user
	key, err := h.store.GetSSHKeyByID(ctx, keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch SSH key"})
		return
	}

	if key == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSH key not found"})
		return
	}

	if key.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Delete the key
	if err := h.store.DeleteSSHKey(ctx, keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete SSH key"})
		return
	}

	// Sync keys to all user's running containers
	go h.syncKeysToUserContainers(userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "SSH key deleted successfully",
		"id":      keyID,
	})
}

// GetSSHConnectionInfo returns SSH connection info for a container
// GET /api/ssh/connect/:containerId
func (h *SSHHandler) GetSSHConnectionInfo(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("containerId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify container ownership
	containerInfo, ok := h.containerManager.GetContainer(containerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Check if container is running
	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "container is not running",
			"status": containerInfo.Status,
		})
		return
	}

	// Get container's IP address
	client := h.containerManager.GetClient()
	inspect, err := client.ContainerInspect(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to inspect container"})
		return
	}

	ipAddress := inspect.NetworkSettings.IPAddress
	if ipAddress == "" {
		// Try to get IP from the first network
		for _, network := range inspect.NetworkSettings.Networks {
			if network.IPAddress != "" {
				ipAddress = network.IPAddress
				break
			}
		}
	}

	if ipAddress == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "container has no IP address",
			"hint":  "The container may still be starting up",
		})
		return
	}

	// Check if user has any SSH keys
	keys, err := h.store.GetSSHKeysByUserID(c.Request.Context(), userID)
	if err != nil || len(keys) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"container_id":   containerID,
			"container_name": containerInfo.ContainerName,
			"host":           ipAddress,
			"port":           22,
			"username":       "user",
			"has_keys":       false,
			"message":        "No SSH keys configured. Add an SSH key to connect.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"container_id":   containerID,
		"container_name": containerInfo.ContainerName,
		"host":           ipAddress,
		"port":           22,
		"username":       "user",
		"has_keys":       true,
		"key_count":      len(keys),
		"ssh_command":    fmt.Sprintf("ssh user@%s", ipAddress),
	})
}

// SyncSSHKeys manually syncs SSH keys to a container
// POST /api/ssh/sync/:containerId
func (h *SSHHandler) SyncSSHKeys(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("containerId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify container ownership
	containerInfo, ok := h.containerManager.GetContainer(containerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Sync keys
	if err := h.syncKeysToContainer(c.Request.Context(), containerID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync SSH keys: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "SSH keys synced successfully",
		"container_id": containerID,
	})
}

// syncKeysToUserContainers syncs SSH keys to all running containers for a user
func (h *SSHHandler) syncKeysToUserContainers(userID string) {
	containers := h.containerManager.GetUserContainers(userID)
	for _, container := range containers {
		if container.Status == "running" {
			h.syncKeysToContainer(context.Background(), container.ID, userID)
		}
	}
}

// syncKeysToContainer syncs SSH keys to a specific container
func (h *SSHHandler) syncKeysToContainer(ctx context.Context, containerID, userID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Get all public keys for the user
	keys, err := h.store.GetAllUserSSHPublicKeys(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get SSH keys: %w", err)
	}

	// Write authorized_keys to the container
	client := h.containerManager.GetClient()

	// Escape single quotes in keys for shell command
	escapedKeys := strings.ReplaceAll(keys, "'", "'\"'\"'")

	// Create exec to write the authorized_keys file
	execConfig := dockerContainer.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd: []string{
			"sh", "-c",
			fmt.Sprintf("mkdir -p /home/user/.ssh && echo '%s' > /home/user/.ssh/authorized_keys && chmod 600 /home/user/.ssh/authorized_keys && chown user:user /home/user/.ssh/authorized_keys", escapedKeys),
		},
		User: "root",
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	// Use ContainerExecAttach for Podman compatibility (it starts the exec)
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, dockerContainer.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach/start exec: %w", err)
	}
	attachResp.Close()

	return nil
}

// validateSSHPublicKey validates that the string is a valid SSH public key
func validateSSHPublicKey(key string) error {
	key = strings.TrimSpace(key)

	// Check for common key types
	validPrefixes := []string{
		"ssh-rsa",
		"ssh-ed25519",
		"ssh-dss",
		"ecdsa-sha2-nistp256",
		"ecdsa-sha2-nistp384",
		"ecdsa-sha2-nistp521",
		"sk-ssh-ed25519@openssh.com",
		"sk-ecdsa-sha2-nistp256@openssh.com",
	}

	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("invalid SSH public key format: must start with a valid key type (ssh-rsa, ssh-ed25519, etc.)")
	}

	// Basic structure validation: type base64-data [comment]
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return fmt.Errorf("invalid SSH public key format: missing key data")
	}

	// Validate base64 encoding of the key data
	keyData := parts[1]
	if _, err := base64.StdEncoding.DecodeString(keyData); err != nil {
		return fmt.Errorf("invalid SSH public key: key data is not valid base64")
	}

	// Check for obviously invalid keys (too short)
	if len(keyData) < 100 {
		return fmt.Errorf("invalid SSH public key: key data too short")
	}

	// Check for newlines which shouldn't be in a public key
	if strings.Contains(key, "\n") {
		// Remove the key type prefix and check again - it should be a single line
		return fmt.Errorf("invalid SSH public key: key should be on a single line")
	}

	return nil
}

// calculateSSHFingerprint calculates the SHA256 fingerprint of an SSH public key
func calculateSSHFingerprint(key string) (string, error) {
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid key format")
	}

	keyData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode key: %w", err)
	}

	// Calculate SHA256 fingerprint
	hash := sha256.Sum256(keyData)
	fingerprint := base64.StdEncoding.EncodeToString(hash[:])
	// Remove trailing = padding
	fingerprint = strings.TrimRight(fingerprint, "=")

	return "SHA256:" + fingerprint, nil
}

// calculateMD5Fingerprint calculates the MD5 fingerprint (legacy format)
func calculateMD5Fingerprint(key string) (string, error) {
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid key format")
	}

	keyData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode key: %w", err)
	}

	hash := md5.Sum(keyData)
	// Format as colon-separated hex
	fingerprint := fmt.Sprintf("%x", hash)
	// Add colons
	re := regexp.MustCompile("(.{2})")
	fingerprint = re.ReplaceAllString(fingerprint, "$1:")
	fingerprint = strings.TrimSuffix(fingerprint, ":")

	return fingerprint, nil
}

// InstallSSH installs SSH server in a container on-demand
// POST /api/ssh/install/:containerId
func (h *SSHHandler) InstallSSH(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("containerId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify container ownership
	containerInfo, ok := h.containerManager.GetContainer(containerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Check if container is running
	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "container is not running",
			"status": containerInfo.Status,
		})
		return
	}

	ctx := c.Request.Context()
	client := h.containerManager.GetClient()

	// Detect OS and install SSH
	installCmd := h.getSSHInstallCommand(ctx, containerID, containerInfo.ImageType)
	if installCmd == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to determine install command for this image"})
		return
	}

	// Create exec to install SSH
	execConfig := dockerContainer.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"sh", "-c", installCmd},
		User:         "root",
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create exec: " + err.Error()})
		return
	}

	// Start the exec via attach (Podman compatibility - attach implicitly starts)
	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, dockerContainer.ExecAttachOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach exec: " + err.Error()})
		return
	}
	defer attachResp.Close()

	// Read output to wait for completion
	io.Copy(io.Discard, attachResp.Reader)

	// Check exit code
	inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to inspect exec: " + err.Error()})
		return
	}

	if inspect.ExitCode != 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "SSH installation failed",
			"exit_code": inspect.ExitCode,
		})
		return
	}

	// Sync SSH keys after installation
	if err := h.syncKeysToContainer(ctx, containerID, userID); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "SSH installed but failed to sync keys",
			"warning": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "SSH server installed and configured",
		"container_id": containerID,
	})
}

// getSSHInstallCommand returns the appropriate SSH install command for the image type
func (h *SSHHandler) getSSHInstallCommand(ctx context.Context, containerID, imageType string) string {
	// Commands to install SSH server and configure it
	commands := map[string]string{
		"ubuntu": `apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y openssh-server && \
			mkdir -p /var/run/sshd && \
			sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config && \
			sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
			sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
			ssh-keygen -A && \
			mkdir -p /home/user/.ssh && \
			chmod 700 /home/user/.ssh && \
			touch /home/user/.ssh/authorized_keys && \
			chmod 600 /home/user/.ssh/authorized_keys && \
			chown -R user:user /home/user/.ssh && \
			/usr/sbin/sshd`,

		"debian": `apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y openssh-server && \
			mkdir -p /var/run/sshd && \
			sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config && \
			sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
			sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
			ssh-keygen -A && \
			mkdir -p /home/user/.ssh && \
			chmod 700 /home/user/.ssh && \
			touch /home/user/.ssh/authorized_keys && \
			chmod 600 /home/user/.ssh/authorized_keys && \
			chown -R user:user /home/user/.ssh && \
			/usr/sbin/sshd`,

		"alpine": `apk add --no-cache openssh-server && \
			mkdir -p /var/run/sshd && \
			ssh-keygen -A && \
			sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config && \
			sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
			sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
			mkdir -p /home/user/.ssh && \
			chmod 700 /home/user/.ssh && \
			touch /home/user/.ssh/authorized_keys && \
			chmod 600 /home/user/.ssh/authorized_keys && \
			chown -R user:user /home/user/.ssh && \
			/usr/sbin/sshd`,

		"fedora": `dnf install -y openssh-server && \
			mkdir -p /var/run/sshd && \
			ssh-keygen -A && \
			sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config && \
			sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
			sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
			mkdir -p /home/user/.ssh && \
			chmod 700 /home/user/.ssh && \
			touch /home/user/.ssh/authorized_keys && \
			chmod 600 /home/user/.ssh/authorized_keys && \
			chown -R user:user /home/user/.ssh && \
			/usr/sbin/sshd`,

		"arch": `pacman -Sy --noconfirm openssh && \
			mkdir -p /var/run/sshd && \
			ssh-keygen -A && \
			sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config && \
			sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
			sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
			mkdir -p /home/user/.ssh && \
			chmod 700 /home/user/.ssh && \
			touch /home/user/.ssh/authorized_keys && \
			chmod 600 /home/user/.ssh/authorized_keys && \
			chown -R user:user /home/user/.ssh && \
			/usr/sbin/sshd`,

		"opensuse": `zypper install -y openssh && \
			mkdir -p /var/run/sshd && \
			ssh-keygen -A && \
			sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config && \
			sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
			sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
			mkdir -p /home/user/.ssh && \
			chmod 700 /home/user/.ssh && \
			touch /home/user/.ssh/authorized_keys && \
			chmod 600 /home/user/.ssh/authorized_keys && \
			chown -R user:user /home/user/.ssh && \
			/usr/sbin/sshd`,
	}

	if cmd, ok := commands[imageType]; ok {
		return cmd
	}

	// Try to detect the OS if imageType is unknown
	client := h.containerManager.GetClient()

	// Helper function to run a simple command and check exit code
	runCheck := func(cmd []string) bool {
		execConfig := dockerContainer.ExecOptions{
			Cmd:          cmd,
			AttachStdout: true,
		}
		execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			return false
		}
		// Use attach for Podman compatibility
		attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, dockerContainer.ExecAttachOptions{})
		if err != nil {
			return false
		}
		attachResp.Close()
		inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
		return err == nil && inspect.ExitCode == 0
	}

	// Check for apt (Debian/Ubuntu)
	if runCheck([]string{"which", "apt-get"}) {
		return commands["debian"]
	}

	// Check for apk (Alpine)
	if runCheck([]string{"which", "apk"}) {
		return commands["alpine"]
	}

	// Check for dnf (Fedora)
	if runCheck([]string{"which", "dnf"}) {
		return commands["fedora"]
	}

	// Check for pacman (Arch)
	if runCheck([]string{"which", "pacman"}) {
		return commands["arch"]
	}

	// Check for zypper (openSUSE/Tumbleweed)
	if runCheck([]string{"which", "zypper"}) {
		return commands["opensuse"]
	}

	return ""
}

// CheckSSHStatus checks if SSH is installed and running in a container
// GET /api/ssh/status/:containerId
func (h *SSHHandler) CheckSSHStatus(c *gin.Context) {
	userID := c.GetString("userID")
	containerID := c.Param("containerId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify container ownership
	containerInfo, ok := h.containerManager.GetContainer(containerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if containerInfo.Status != "running" {
		c.JSON(http.StatusOK, gin.H{
			"installed":        false,
			"running":          false,
			"container_status": containerInfo.Status,
		})
		return
	}

	ctx := c.Request.Context()
	client := h.containerManager.GetClient()

	// Helper to run a simple check command
	runCheck := func(cmd []string) bool {
		execConfig := dockerContainer.ExecOptions{
			Cmd:          cmd,
			AttachStdout: true,
		}
		execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			return false
		}
		// Use attach for Podman compatibility
		attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, dockerContainer.ExecAttachOptions{})
		if err != nil {
			return false
		}
		attachResp.Close()
		inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
		return err == nil && inspect.ExitCode == 0
	}

	// Check if sshd exists
	installed := runCheck([]string{"which", "sshd"})

	// Check if sshd is running
	running := false
	if installed {
		running = runCheck([]string{"pgrep", "-x", "sshd"})
	}

	c.JSON(http.StatusOK, gin.H{
		"installed":        installed,
		"running":          running,
		"container_id":     containerID,
		"container_status": containerInfo.Status,
	})
}

// ListRemoteHosts returns all remote hosts for the authenticated user
// GET /api/ssh/hosts
func (h *SSHHandler) ListRemoteHosts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	hosts, err := h.store.GetRemoteHostsByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch remote hosts"})
		return
	}

	response := make([]RemoteHostResponse, 0, len(hosts))
	for _, host := range hosts {
		port := host.Port
		if port == 0 {
			port = 22
		}
		sshCmd := fmt.Sprintf("ssh %s@%s", host.Username, host.Hostname)
		if port != 22 {
			sshCmd += fmt.Sprintf(" -p %d", port)
		}
		if host.IdentityFile != "" {
			sshCmd += fmt.Sprintf(" -i %s", host.IdentityFile)
		}

		response = append(response, RemoteHostResponse{
			ID:           host.ID,
			Name:         host.Name,
			Hostname:     host.Hostname,
			Port:         port,
			Username:     host.Username,
			IdentityFile: host.IdentityFile,
			SSHCommand:   sshCmd,
			CreatedAt:    host.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"hosts": response,
		"count": len(response),
	})
}

// AddRemoteHost adds a new remote host
// POST /api/ssh/hosts
func (h *SSHHandler) AddRemoteHost(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req AddRemoteHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	port := req.Port
	if port == 0 {
		port = 22
	}

	host := &models.RemoteHost{
		ID:           uuid.New().String(),
		UserID:       userID,
		Name:         req.Name,
		Hostname:     req.Hostname,
		Port:         port,
		Username:     req.Username,
		IdentityFile: req.IdentityFile,
		CreatedAt:    time.Now(),
	}

	if err := h.store.CreateRemoteHost(c.Request.Context(), host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save remote host"})
		return
	}

	// Construct SSH command for response
	sshCmd := fmt.Sprintf("ssh %s@%s", host.Username, host.Hostname)
	if host.Port != 22 {
		sshCmd += fmt.Sprintf(" -p %d", host.Port)
	}
	if host.IdentityFile != "" {
		sshCmd += fmt.Sprintf(" -i %s", host.IdentityFile)
	}

	c.JSON(http.StatusCreated, RemoteHostResponse{
		ID:           host.ID,
		Name:         host.Name,
		Hostname:     host.Hostname,
		Port:         host.Port,
		Username:     host.Username,
		IdentityFile: host.IdentityFile,
		SSHCommand:   sshCmd,
		CreatedAt:    host.CreatedAt,
	})
}

// DeleteRemoteHost removes a remote host
// DELETE /api/ssh/hosts/:id
func (h *SSHHandler) DeleteRemoteHost(c *gin.Context) {
	userID := c.GetString("userID")
	hostID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// Verify ownership
	host, err := h.store.GetRemoteHostByID(ctx, hostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch host"})
		return
	}
	if host == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}
	if host.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.store.DeleteRemoteHost(ctx, hostID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete host"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Remote host deleted"})
}
