package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rexec/rexec/internal/api/handlers"
	"github.com/rexec/rexec/internal/api/middleware"
	"github.com/rexec/rexec/internal/billing"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/crypto"
	"github.com/rexec/rexec/internal/storage"
)

var (
	apiLimiter       = middleware.APIRateLimiter()
	authLimiter      = middleware.AuthRateLimiter()
	containerLimiter = middleware.ContainerRateLimiter()
	wsLimiter        = middleware.WebSocketRateLimiter()
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "server":
			runServer()
		case "admin":
			handleAdminCommand(os.Args[2:])
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			fmt.Println("Usage: rexec [server|admin]")
			os.Exit(1)
		}
		return
	}

	showMenu()
}

func showMenu() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n🚀 Rexec CLI")
		fmt.Println("-----------------------------")
		fmt.Println("1. Start Server")
		fmt.Println("2. Admin Tools")
		fmt.Println("3. Exit")
		fmt.Println("-----------------------------")
		fmt.Print("Select an option: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			runServer()
			return
		case "2":
			showAdminMenu()
		case "3":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("Invalid option, please try again.")
		}
	}
}

func handleAdminCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: rexec admin [promote <email>]")
		return
	}

	switch args[0] {
	case "promote":
		if len(args) < 2 {
			fmt.Println("Usage: rexec admin promote <email>")
			return
		}
		promoteUser(args[1])
	default:
		fmt.Printf("Unknown admin command: %s\n", args[0])
	}
}

func showAdminMenu() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n🛡️  Admin Tools")
		fmt.Println("-----------------------------")
		fmt.Println("1. Promote User to Admin")
		fmt.Println("2. Back to Main Menu")
		fmt.Println("-----------------------------")
		fmt.Print("Select an option: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			fmt.Print("Enter email to promote: ")
			email, _ := reader.ReadString('\n')
			promoteUser(strings.TrimSpace(email))
		case "2":
			return
		default:
			fmt.Println("Invalid option.")
		}
	}
}

func promoteUser(email string) {
	// Initialize DB connection
	godotenv.Load()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://rexec:rexec@localhost:5432/rexec?sslmode=disable"
	}

	// We don't need the encryptor for this specific operation, but NewPostgresStore requires it
	// Just use a dummy key since we won't be using encrypted fields
	encryptor, _ := crypto.NewEncryptor("dummy-key-for-admin-cli-operation")

	store, err := storage.NewPostgresStore(databaseURL, encryptor)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	user, _, err := store.GetUserByEmail(ctx, email)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		return
	}
	if user == nil {
		fmt.Printf("User with email %s not found.\n", email)
		return
	}

	user.IsAdmin = true
	// UpdateUser updates username, tier, pipeops_id, etc.
	// But our UpdateUser implementation now includes is_admin!
	// Let's double check storage.UpdateUser implementation we updated.
	// Yes: UPDATE users SET username = $2, tier = $3, is_admin = $4 ...
	
	if err := store.UpdateUser(ctx, user); err != nil {
		log.Printf("Failed to promote user: %v", err)
		return
	}

	fmt.Printf("✅ User %s (%s) successfully promoted to Admin.\n", user.Email, user.ID)
}

func runServer() {
	// Load .env file if it exists
	godotenv.Load()

	// Get database URL
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://rexec:rexec@localhost:5432/rexec?sslmode=disable"
	}

	// Initialize encryption
	encryptionKey := os.Getenv("REXEC_ENCRYPTION_KEY")
	if encryptionKey == "" {
		// Use a default key for development ONLY if not set
		if os.Getenv("GIN_MODE") != "release" {
			log.Println("⚠️  REXEC_ENCRYPTION_KEY not set, using default dev key")
			encryptionKey = "rexec-dev-key-do-not-use-in-prod" // 32 bytes
		} else {
			log.Fatal("REXEC_ENCRYPTION_KEY must be set in production")
		}
	}

	encryptor, err := crypto.NewEncryptor(encryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize encryptor: %v", err)
	}

	// Initialize PostgreSQL store
	store, err := storage.NewPostgresStore(databaseURL, encryptor)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()
	log.Println("✅ Connected to PostgreSQL")

	// Initialize container manager
	containerManager, err := container.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize container manager: %v", err)
	}
	defer containerManager.Close()

	// Log Docker connection type
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost != "" {
		log.Printf("✅ Connected to Docker (remote: %s)", dockerHost)
	} else {
		log.Println("✅ Connected to Docker (local socket)")
	}

	// Load existing containers from Docker
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := containerManager.LoadExistingContainers(ctx); err != nil {
		log.Printf("⚠️  Warning: Failed to load existing containers: %v", err)
	} else {
		containers := containerManager.ListContainers()
		log.Printf("✅ Loaded %d existing containers from Docker", len(containers))
	}

	// Start cleanup service for idle containers
	cleanupConfig := container.DefaultCleanupConfig()
	if os.Getenv("GIN_MODE") == "debug" {
		cleanupConfig = container.DevelopmentCleanupConfig()
	}

	if cleanupConfig.Enabled {
		cleanupService := container.NewCleanupService(
			containerManager,
			cleanupConfig.IdleTimeout,
			cleanupConfig.CheckInterval,
		)
		cleanupService.Start()
		defer cleanupService.Stop()
	}

	// Start reconciler service to sync DB state with Docker
	reconcilerService := container.NewReconcilerService(
		containerManager,
		store,
		1*time.Minute, // Check every minute
	)
	reconcilerService.Start()
	defer reconcilerService.Stop()

	// Initialize billing service
	var billingService *billing.Service
	if os.Getenv("STRIPE_SECRET_KEY") != "" {
		billingService = billing.NewService()
		log.Println("✅ Stripe billing enabled")
	} else {
		log.Println("⚠️  Stripe not configured (billing disabled)")
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(store)
	containerHandler := handlers.NewContainerHandler(containerManager, store)
	containerEventsHub := handlers.NewContainerEventsHub(containerManager, store)
	containerHandler.SetEventsHub(containerEventsHub)
	terminalHandler := handlers.NewTerminalHandler(containerManager, store)
	fileHandler := handlers.NewFileHandler(containerManager, store)
	sshHandler := handlers.NewSSHHandler(store, containerManager)
	collabHandler := handlers.NewCollabHandler(store, containerManager, terminalHandler)
	recordingHandler := handlers.NewRecordingHandler(store, os.Getenv("RECORDINGS_PATH"))
	// Connect recording handler to terminal handler to capture events
	terminalHandler.SetRecordingHandler(recordingHandler)
	// Connect collab handler to terminal handler for shared session access
	terminalHandler.SetCollabHandler(collabHandler)
	var billingHandler *handlers.BillingHandler
	if billingService != nil {
		billingHandler = handlers.NewBillingHandler(billingService, store)
	}

	// Initialize port forward handler
	portForwardHandler := handlers.NewPortForwardHandler(store, containerManager)
	
	// Initialize snippet handler
	snippetHandler := handlers.NewSnippetHandler(store)

	// Initialize admin handler
	adminHandler := handlers.NewAdminHandler(store)

	// Setup Gin router
	router := gin.Default()

	// Gzip compression for faster transfers (skip WebSocket)
	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/ws/"})))

	// Cache control middleware for static assets
	router.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Long cache for hashed assets (immutable)
		if strings.HasPrefix(path, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
			// Cache SW and other static files
			c.Header("Cache-Control", "public, max-age=86400")
		} else if path == "/sw.js" {
			// Service worker should be checked frequently
			c.Header("Cache-Control", "public, max-age=0, must-revalidate")
		}
		c.Next()
	})

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		stats := containerManager.GetContainerStats()

		health := gin.H{
			"status":             "ok",
			"database":           "connected",
			"docker":             "connected",
			"containers_total":   stats.Total,
			"containers_running": stats.Running,
		}

		c.JSON(200, health)
	})

	// Auth routes (public) - strict rate limiting
	auth := router.Group("/api/auth")
	auth.Use(authLimiter.Middleware())
	{
		// Guest login (1-hour session limit)
		auth.POST("/guest", authHandler.GuestLogin)

		// PipeOps OAuth routes
		auth.GET("/oauth/url", authHandler.GetOAuthURL)
		auth.GET("/callback", authHandler.OAuthCallback)
		auth.GET("/signin", authHandler.OAuthCallback) // Alternative callback path
		auth.POST("/oauth/exchange", authHandler.OAuthExchange)
	}

	// API routes (protected) - with rate limiting
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	api.Use(apiLimiter.Middleware())
	{
		// Container management - stricter rate limiting for mutations
		api.GET("/containers", containerHandler.List)
		api.POST("/containers", containerLimiter.Middleware(), containerHandler.Create)
		api.POST("/containers/stream", containerLimiter.Middleware(), containerHandler.CreateWithProgress)
		api.GET("/containers/:id", containerHandler.Get)
		api.PATCH("/containers/:id/settings", containerHandler.UpdateSettings)
		api.DELETE("/containers/:id", containerHandler.Delete)
		api.POST("/containers/:id/start", containerHandler.Start)
		api.POST("/containers/:id/stop", containerHandler.Stop)

		// Shell setup
		api.GET("/containers/:id/shell/status", containerHandler.GetShellStatus)
		api.POST("/containers/:id/shell/setup", containerHandler.SetupShell)

		// File operations
		api.POST("/containers/:id/files", fileHandler.Upload)
		api.GET("/containers/:id/files", fileHandler.Download)
		api.GET("/containers/:id/files/list", fileHandler.List)
		api.DELETE("/containers/:id/files", fileHandler.Delete)
		api.POST("/containers/:id/files/mkdir", fileHandler.Mkdir)

		// Available images
		api.GET("/images", containerHandler.ListImages)

		// Available roles/environments
		api.GET("/roles", containerHandler.ListRoles)

		// Stats endpoint
		api.GET("/stats", containerHandler.Stats)

		// User profile
		api.GET("/profile", authHandler.GetProfile)
		api.PUT("/profile", authHandler.UpdateProfile)

		// SSH key management
		ssh := api.Group("/ssh")
		{
			ssh.GET("/keys", sshHandler.ListSSHKeys)
			ssh.POST("/keys", sshHandler.AddSSHKey)
			ssh.DELETE("/keys/:id", sshHandler.DeleteSSHKey)
			ssh.GET("/connect/:containerId", sshHandler.GetSSHConnectionInfo)
			ssh.POST("/sync/:containerId", sshHandler.SyncSSHKeys)
			ssh.GET("/status/:containerId", sshHandler.CheckSSHStatus)
			ssh.POST("/install/:containerId", sshHandler.InstallSSH)
			
			// Remote Hosts (Jump Hosts)
			ssh.GET("/hosts", sshHandler.ListRemoteHosts)
			ssh.POST("/hosts", sshHandler.AddRemoteHost)
			ssh.DELETE("/hosts/:id", sshHandler.DeleteRemoteHost)
		}

		// Port Forwarding
		portforward := api.Group("/containers/:id/port-forwards")
		{
			portforward.POST("", portForwardHandler.CreatePortForward)
			portforward.GET("", portForwardHandler.ListPortForwards)
			portforward.DELETE("/:forwardId", portForwardHandler.DeletePortForward)
		}

		// Snippets & Macros
		snippets := api.Group("/snippets")
		{
			snippets.GET("", snippetHandler.ListSnippets)
			snippets.POST("", snippetHandler.CreateSnippet)
			snippets.DELETE("/:id", snippetHandler.DeleteSnippet)
		}

		// WebSocket for real-time container events
		api.GET("/containers/events", containerEventsHub.HandleWebSocket)

		// Collaboration endpoints
		collab := api.Group("/collab")
		{
			collab.POST("/start", collabHandler.StartSession)
			collab.GET("/join/:code", collabHandler.JoinSession)
			collab.DELETE("/sessions/:id", collabHandler.EndSession)
			collab.GET("/sessions", collabHandler.GetActiveSessions)
		}

		// Recording endpoints
		recordings := api.Group("/recordings")
		{
			recordings.GET("", recordingHandler.GetRecordings)
			recordings.POST("/start", recordingHandler.StartRecording)
			recordings.POST("/stop/:containerId", recordingHandler.StopRecording)
			recordings.GET("/status/:containerId", recordingHandler.GetRecordingStatus)
			recordings.GET("/:id", recordingHandler.GetRecording)
			recordings.GET("/:id/stream", recordingHandler.StreamRecording)
			recordings.PATCH("/:id", recordingHandler.UpdateRecording)
			recordings.DELETE("/:id", recordingHandler.DeleteRecording)
		}

		// Billing endpoints (if enabled)
		if billingHandler != nil {
			billing := api.Group("/billing")
			{
				billing.GET("/plans", billingHandler.GetPlans)
				billing.GET("/subscription", billingHandler.GetSubscription)
				billing.POST("/checkout", billingHandler.CreateCheckoutSession)
				billing.POST("/portal", billingHandler.CreatePortalSession)
				billing.POST("/cancel", billingHandler.CancelSubscription)
			}
		}

		// Admin routes
		admin := api.Group("/admin")
		admin.Use(middleware.AdminOnly(store))
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)

			admin.GET("/containers", adminHandler.ListContainers)
			admin.DELETE("/containers/:id", adminHandler.DeleteContainer)

			admin.GET("/terminals", adminHandler.ListTerminals)
		}
	}

	// Stripe webhook (public, verified by signature)
	if billingHandler != nil {
		router.POST("/api/billing/webhook", billingHandler.HandleWebhook)
	}

	// WebSocket terminal endpoint - with rate limiting
	router.GET("/ws/terminal/:containerId", wsLimiter.Middleware(), middleware.AuthMiddleware(), terminalHandler.HandleWebSocket)

	// WebSocket collaboration endpoint
	router.GET("/ws/collab/:code", wsLimiter.Middleware(), middleware.AuthMiddleware(), collabHandler.HandleCollabWebSocket)

	// WebSocket for Port Forwarding
	router.GET("/ws/port-forward/:forwardId", wsLimiter.Middleware(), middleware.AuthMiddleware(), portForwardHandler.HandlePortForwardWebSocket)

	// Public recording access (no auth required for shared recordings)
	router.GET("/r/:token", recordingHandler.GetRecordingByToken)
	router.GET("/r/:token/stream", recordingHandler.StreamRecordingByToken)

	// Serve static files (frontend)
	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		// Default to ./web directory relative to binary
		webDir = "web"
	}

	// Check if web directory exists
	if _, err := os.Stat(webDir); err == nil {
		indexFile := filepath.Join(webDir, "index.html")

		router.StaticFile("/", indexFile)
		router.StaticFile("/index.html", indexFile)
		router.Static("/assets", filepath.Join(webDir, "assets"))
		router.StaticFile("/favicon.ico", filepath.Join(webDir, "favicon.svg")) // Serve SVG for .ico requests
		router.StaticFile("/favicon.svg", filepath.Join(webDir, "favicon.svg"))
		router.StaticFile("/manifest.json", filepath.Join(webDir, "manifest.json"))
		router.StaticFile("/manifest.webmanifest", filepath.Join(webDir, "manifest.webmanifest"))
		router.StaticFile("/sw.js", filepath.Join(webDir, "sw.js"))
		router.StaticFile("/pwa-192x192.png", filepath.Join(webDir, "pwa-192x192.png"))
		router.StaticFile("/pwa-512x512.png", filepath.Join(webDir, "pwa-512x512.png"))
		router.StaticFile("/robots.txt", filepath.Join(webDir, "robots.txt"))
		router.StaticFile("/sitemap.xml", filepath.Join(webDir, "sitemap.xml"))
		router.StaticFile("/og-image.svg", filepath.Join(webDir, "og-image.svg"))

		// Apple touch icons - serve favicon for these requests
		router.StaticFile("/apple-touch-icon.png", filepath.Join(webDir, "favicon.svg"))
		router.StaticFile("/apple-touch-icon-precomposed.png", filepath.Join(webDir, "favicon.svg"))

		// SPA routes - serve index.html for client-side routing
		router.GET("/guides", func(c *gin.Context) {
			c.File(indexFile)
		})
		router.GET("/use-cases", func(c *gin.Context) {
			c.File(indexFile)
		})
		// Legacy routes - redirect or serve index
		router.GET("/ai-tools", func(c *gin.Context) {
			c.File(indexFile)
		})
		router.GET("/agentic", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Terminal URL routes - serve index.html for SPA routing
		router.GET("/terminal/:id", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Join session route
		router.GET("/join/:code", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Also support direct container ID in URL path
		router.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			// Only serve index.html if it looks like a container ID (64 hex chars or UUID)
			if len(id) == 64 || (len(id) == 36 && id[8] == '-' && id[13] == '-') {
				c.File(indexFile)
				return
			}
			// Otherwise, let it fall through to 404
			c.Status(404)
		})

		// Catch-all for SPA routing
		router.NoRoute(func(c *gin.Context) {
			c.File(indexFile)
		})
	}

	// Get port from env or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Rexec server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
