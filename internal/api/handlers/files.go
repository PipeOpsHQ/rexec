package handlers

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	mgr "github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/storage"
)

// FileHandler handles file upload/download operations for containers
type FileHandler struct {
	containerManager *mgr.Manager
	store            *storage.PostgresStore
}

// NewFileHandler creates a new file handler
func NewFileHandler(cm *mgr.Manager, store *storage.PostgresStore) *FileHandler {
	return &FileHandler{
		containerManager: cm,
		store:            store,
	}
}

// FileInfo represents information about a file
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
}

// Upload handles file upload to a container
// POST /api/containers/:id/files?path=/home/user/
func (h *FileHandler) Upload(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")
	destPath := c.Query("path")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if destPath == "" {
		destPath = "/home/user/"
	}

	// Verify ownership
	containerInfo, ok := h.containerManager.GetContainer(dockerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container is not running"})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded: " + err.Error()})
		return
	}
	defer file.Close()

	// Validate file size (max 100MB)
	if header.Size > 100*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 100MB)"})
		return
	}

	// Sanitize filename
	filename := filepath.Base(header.Filename)
	if filename == "." || filename == ".." || strings.Contains(filename, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Create tar archive with the file
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name:    filename,
		Mode:    0644,
		Size:    int64(len(content)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create archive"})
		return
	}

	if _, err := tw.Write(content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write to archive"})
		return
	}

	if err := tw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close archive"})
		return
	}

	// Copy to container
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := h.containerManager.GetClient()
	err = client.CopyToContainer(ctx, dockerID, destPath, &buf, container.CopyToContainerOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to copy to container: " + err.Error()})
		return
	}

	// Touch container to update last used
	h.containerManager.TouchContainer(dockerID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "file uploaded successfully",
		"filename": filename,
		"path":     filepath.Join(destPath, filename),
		"size":     header.Size,
	})
}

// Download handles file download from a container
// GET /api/containers/:id/files?path=/home/user/file.txt
func (h *FileHandler) Download(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")
	filePath := c.Query("path")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter required"})
		return
	}

	// Verify ownership
	containerInfo, ok := h.containerManager.GetContainer(dockerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container is not running"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := h.containerManager.GetClient()

	// Copy from container
	reader, stat, err := client.CopyFromContainer(ctx, dockerID, filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found: " + err.Error()})
		return
	}
	defer reader.Close()

	// Read tar archive
	tr := tar.NewReader(reader)
	header, err := tr.Next()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read archive"})
		return
	}

	// Check if it's a directory
	if header.Typeflag == tar.TypeDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is a directory, use /files/list to list contents"})
		return
	}

	// Read file content
	content, err := io.ReadAll(tr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file content"})
		return
	}

	// Touch container
	h.containerManager.TouchContainer(dockerID)

	// Set headers for download
	filename := filepath.Base(filePath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", len(content)))
	c.Header("X-File-Size", fmt.Sprintf("%d", stat.Size))
	c.Header("X-File-Mode", stat.Mode.String())

	c.Data(http.StatusOK, "application/octet-stream", content)
}

// List lists files in a directory within a container
// GET /api/containers/:id/files/list?path=/home/user/
func (h *FileHandler) List(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")
	dirPath := c.Query("path")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if dirPath == "" {
		dirPath = "/home/user"
	}

	// Verify ownership
	containerInfo, ok := h.containerManager.GetContainer(dockerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container is not running"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := h.containerManager.GetClient()

	// Use exec to run ls command (basic flags for busybox compatibility)
	execConfig := container.ExecOptions{
		Cmd:          []string{"ls", "-la", dirPath},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, dockerID, execConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create exec: " + err.Error()})
		return
	}

	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach exec: " + err.Error()})
		return
	}
	defer attachResp.Close()

	// Read output
	output, err := io.ReadAll(attachResp.Reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read output"})
		return
	}

	// Parse ls output
	files := parseListOutput(string(output), dirPath)

	// Touch container
	h.containerManager.TouchContainer(dockerID)

	c.JSON(http.StatusOK, gin.H{
		"path":  dirPath,
		"files": files,
		"count": len(files),
	})
}

// Delete deletes a file from a container
// DELETE /api/containers/:id/files?path=/home/user/file.txt
func (h *FileHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")
	filePath := c.Query("path")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter required"})
		return
	}

	// Safety check - don't allow deleting system paths
	dangerousPaths := []string{"/", "/bin", "/sbin", "/usr", "/etc", "/var", "/lib", "/root"}
	for _, dangerous := range dangerousPaths {
		if filePath == dangerous || strings.HasPrefix(filePath, dangerous+"/") && !strings.HasPrefix(filePath, "/home/") {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system files"})
			return
		}
	}

	// Verify ownership
	containerInfo, ok := h.containerManager.GetContainer(dockerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container is not running"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := h.containerManager.GetClient()

	// Use exec to run rm command
	execConfig := container.ExecOptions{
		Cmd:          []string{"rm", "-rf", filePath},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, dockerID, execConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create exec: " + err.Error()})
		return
	}

	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach exec: " + err.Error()})
		return
	}
	defer attachResp.Close()

	// Wait for completion
	io.Copy(io.Discard, attachResp.Reader)

	// Check exit code
	inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check result"})
		return
	}

	if inspect.ExitCode != 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}

	// Touch container
	h.containerManager.TouchContainer(dockerID)

	c.JSON(http.StatusOK, gin.H{
		"message": "file deleted successfully",
		"path":    filePath,
	})
}

// Mkdir creates a directory in a container
// POST /api/containers/:id/files/mkdir?path=/home/user/newdir
func (h *FileHandler) Mkdir(c *gin.Context) {
	userID := c.GetString("userID")
	dockerID := c.Param("id")
	dirPath := c.Query("path")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if dirPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter required"})
		return
	}

	// Verify ownership
	containerInfo, ok := h.containerManager.GetContainer(dockerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if containerInfo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if containerInfo.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container is not running"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := h.containerManager.GetClient()

	// Use exec to run mkdir command
	execConfig := container.ExecOptions{
		Cmd:          []string{"mkdir", "-p", dirPath},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, dockerID, execConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create exec: " + err.Error()})
		return
	}

	attachResp, err := client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach exec: " + err.Error()})
		return
	}
	defer attachResp.Close()

	// Wait for completion
	io.Copy(io.Discard, attachResp.Reader)

	// Check exit code
	inspect, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check result"})
		return
	}

	if inspect.ExitCode != 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory"})
		return
	}

	// Touch container
	h.containerManager.TouchContainer(dockerID)

	c.JSON(http.StatusOK, gin.H{
		"message": "directory created successfully",
		"path":    dirPath,
	})
}

// parseListOutput parses ls -la output into FileInfo structs
// Handles both GNU ls and busybox ls formats
func parseListOutput(output, basePath string) []FileInfo {
	var files []FileInfo

	// Clean up docker exec output (may have control characters)
	// Remove first 8 bytes if they look like docker stream header
	cleanOutput := output
	if len(output) > 8 && (output[0] == 1 || output[0] == 2) {
		// Docker multiplexed stream - find first newline and start from there
		// or skip the header bytes
		for i := 0; i < len(output) && i < 100; i++ {
			if output[i] == '\n' {
				cleanOutput = output[i+1:]
				break
			}
		}
	}

	lines := strings.Split(cleanOutput, "\n")

	for _, line := range lines {
		// Skip empty lines and total line
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}

		// Skip lines that are too short to be valid
		if len(line) < 10 {
			continue
		}

		// Parse ls -la output
		// GNU format:    -rw-r--r-- 1 user user 1234 Jan  1 12:00 filename
		// Busybox format: -rw-r--r-- 1 user user 1234 Jan  1 12:00 filename
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		// Get mode - must start with valid file type
		mode := fields[0]
		if len(mode) < 10 {
			continue
		}

		// Validate mode string starts with valid type
		validTypes := "dlcbsp-"
		if !strings.ContainsRune(validTypes, rune(mode[0])) {
			continue
		}

		// Skip . and ..
		// Name is everything after the date/time fields (last field(s))
		// For ls -la, typically: mode links user group size month day time name
		// Name could have spaces, so we rejoin from field 8 onwards
		name := strings.Join(fields[8:], " ")

		// Handle case where time might be year instead (for old files)
		// format: Jan  1  2024 filename
		if len(fields) >= 9 && len(fields[7]) == 4 {
			// Year format - name starts at field 8
			name = strings.Join(fields[8:], " ")
		}

		if name == "" {
			name = fields[len(fields)-1]
		}

		if name == "." || name == ".." {
			continue
		}

		// Parse size (field 4)
		var size int64
		fmt.Sscanf(fields[4], "%d", &size)

		// Parse time - combine month day time/year
		var modTime time.Time
		if len(fields) >= 8 {
			timeStr := fmt.Sprintf("%s %s %s", fields[5], fields[6], fields[7])
			// Try various formats
			formats := []string{
				"Jan 2 15:04",
				"Jan 2 2006",
				"Jan _2 15:04",
				"Jan _2 2006",
			}
			for _, format := range formats {
				if t, err := time.Parse(format, timeStr); err == nil {
					// If no year, use current year
					if t.Year() == 0 {
						t = t.AddDate(time.Now().Year(), 0, 0)
					}
					modTime = t
					break
				}
			}
		}

		isDir := mode[0] == 'd'

		files = append(files, FileInfo{
			Name:    name,
			Path:    filepath.Join(basePath, name),
			Size:    size,
			Mode:    mode,
			ModTime: modTime,
			IsDir:   isDir,
		})
	}

	return files
}
