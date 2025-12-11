package main

import (
	"bufio"
	"context"
	"encoding/hex"
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
	admin_events "github.com/rexec/rexec/internal/api/handlers/admin_events"
	"github.com/rexec/rexec/internal/api/middleware"
	"github.com/rexec/rexec/internal/auth"
	"github.com/rexec/rexec/internal/billing"
	"github.com/rexec/rexec/internal/container"
	"github.com/rexec/rexec/internal/crypto"
	"github.com/rexec/rexec/internal/pubsub"
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
			// For CLI admin commands, we need a minimal setup for database access
			// and potentially to broadcast events (even if no WS clients are connected)
			err := godotenv.Load()
			if err != nil && !os.IsNotExist(err) {
				log.Printf("Warning: Could not load .env file for admin command: %v", err)
			}

			databaseURL := os.Getenv("DATABASE_URL")
			if databaseURL == "" {
				databaseURL = "postgres://rexec:rexec@localhost:5432/rexec?sslmode=disable"
			}
			encryptor, _ := crypto.NewEncryptor("dummy-key-for-admin-cli-operation") // Dummy key for CLI
			store, err := storage.NewPostgresStore(databaseURL, encryptor)
			if err != nil {
				log.Fatalf("Failed to connect to database for admin command: %v", err)
			}
			defer store.Close()

			// Create a dummy containerManager and adminEventsHub for CLI context
			// The adminEventsHub will not have active WebSocket connections, but its Broadcast method can still be called
			dummyContainerManager, _ := container.NewManager() // This might still try to connect to Docker, might need a mock for truly disconnected CLI. For now, assume Docker might be running or handle error.
			dummyAdminEventsHub := admin_events.NewAdminEventsHub(store, dummyContainerManager)

			handleAdminCommand(os.Args[2:], store, dummyContainerManager, dummyAdminEventsHub)
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

func handleAdminCommand(args []string, store *storage.PostgresStore, containerManager *container.Manager, adminEventsHub *admin_events.AdminEventsHub) {
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
		promoteUser(args[1], store, adminEventsHub) // Pass store and adminEventsHub
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
		fmt.Println("-----------------------------") // Added newline for better formatting
		fmt.Print("Select an option: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			fmt.Print("Enter email to promote: ")
			email, _ := reader.ReadString('\n')

			// For interactive menu, create a temporary setup for database and hub
			err := godotenv.Load()
			if err != nil && !os.IsNotExist(err) {
				log.Printf("Warning: Could not load .env file for admin menu: %v", err)
			}
			databaseURL := os.Getenv("DATABASE_URL")
			if databaseURL == "" {
				databaseURL = "postgres://rexec:rexec@localhost:5432/rexec?sslmode=disable"
			}
			encryptor, _ := crypto.NewEncryptor("dummy-key-for-admin-cli-operation")
			store, err := storage.NewPostgresStore(databaseURL, encryptor)
			if err != nil {
				log.Fatalf("Failed to connect to database for admin menu: %v", err)
			}
			defer store.Close()

			dummyContainerManager, _ := container.NewManager() // Might still try to connect to Docker
			dummyAdminEventsHub := admin_events.NewAdminEventsHub(store, dummyContainerManager)

			promoteUser(strings.TrimSpace(email), store, dummyAdminEventsHub) // Pass store and hub
		case "2":
			return
		default:
			fmt.Println("Invalid option.")
		}
	}
}

func promoteUser(email string, store *storage.PostgresStore, adminEventsHub *admin_events.AdminEventsHub) { // Corrected signature
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

	if err := store.UpdateUser(ctx, user); err != nil {
		log.Printf("Failed to promote user: %v", err)
		return
	}

	// Broadcast user updated event
	if adminEventsHub != nil {
		adminEventsHub.Broadcast("user_updated", user)
	}

	fmt.Printf("✅ User %s (%s) successfully promoted to Admin.\n", user.Email, user.ID)
}

func runServer() {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) { // Handle non-existent .env gracefully
		log.Printf("Warning: Could not load .env file for server: %v", err)
	}

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

	if os.Getenv("GIN_MODE") == "release" && encryptionKey == "rexec-dev-key-do-not-use-in-prod" {
		log.Fatal("REXEC_ENCRYPTION_KEY must be set to a production key in release mode")
	}

	// Initialize JWT secret for auth
	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		if os.Getenv("GIN_MODE") == "release" {
			log.Fatal("JWT_SECRET must be set in production")
		}
		log.Println("⚠️  JWT_SECRET not set, using default dev secret")
		jwtSecretStr = "rexec-dev-secret-change-in-production"
	}
	if os.Getenv("GIN_MODE") == "release" && jwtSecretStr == "rexec-dev-secret-change-in-production" {
		log.Fatal("JWT_SECRET must be a secure production secret in release mode")
	}
	if os.Getenv("GIN_MODE") == "release" && len(jwtSecretStr) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters in release mode")
	}
	jwtSecret := []byte(jwtSecretStr)

	var keyBytes []byte
	
	// Support hex encoded keys (common for 32-byte keys represented as 64-char hex strings)
	if len(encryptionKey) == 64 {
		var err error
		keyBytes, err = hex.DecodeString(encryptionKey)
		if err != nil {
			// If not valid hex, assume it's just a very long password? 
			// crypto.NewEncryptor only supports 16/24/32 bytes.
			// 64 bytes is not supported by AES-GCM standard key sizes (128/192/256 bits).
			// So if it's 64 chars and fails hex decode, it's invalid length for raw key.
			log.Fatalf("Invalid REXEC_ENCRYPTION_KEY: 64-char key must be valid hex: %v", err)
		}
	} else if len(encryptionKey) == 16 || len(encryptionKey) == 24 || len(encryptionKey) == 32 {
		keyBytes = []byte(encryptionKey)
	} else {
		log.Fatalf("Invalid REXEC_ENCRYPTION_KEY length. Must be 16, 24, or 32 bytes (raw), or 64 hex characters (32 bytes decoded). Got %d characters.", len(encryptionKey))
	}

	// Pass the byte slice directly to NewEncryptor if we modified it to accept []byte, 
	// OR convert back to string if it accepts string but we've validated/transformed it.
	// Looking at crypto package, NewEncryptor(key string). 
	// If we decoded hex, we have []byte. converting []byte to string might not work if NewEncryptor expects readable chars? 
	// No, AES key is just bytes. string(keyBytes) is fine in Go as long as NewEncryptor converts it back to []byte.
	
	encryptor, err := crypto.NewEncryptor(string(keyBytes))
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

	// Initialize MFA service
	mfaService := auth.NewMFAService("Rexec")

	// --- Initialize AdminEventsHub FIRST ---
	adminEventsHub := admin_events.NewAdminEventsHub(store, containerManager)

	// --- Initialize Redis Pub/Sub for horizontal scaling ---
	var pubsubHub *pubsub.Hub
	var wsManager *pubsub.WSManager
	if os.Getenv("REDIS_URL") != "" {
		var err error
		pubsubHub, err = pubsub.NewHub()
		if err != nil {
			log.Printf("⚠️  Redis pub/sub failed to initialize: %v (running in single-instance mode)", err)
		} else {
			pubsubHub.Start()
			defer pubsubHub.Stop()
			wsManager = pubsub.NewWSManager(pubsubHub)
			log.Println("✅ Redis pub/sub enabled for horizontal scaling")
		}
	} else {
		log.Println("⚠️  REDIS_URL not configured (running in single-instance mode)")
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(store, adminEventsHub, jwtSecret)
	containerHandler := handlers.NewContainerHandler(containerManager, store, adminEventsHub)
	containerEventsHub := handlers.NewContainerEventsHub(containerManager, store)
	containerHandler.SetEventsHub(containerEventsHub)
	terminalHandler := handlers.NewTerminalHandler(containerManager, store, adminEventsHub)
	fileHandler := handlers.NewFileHandler(containerManager, store)
	sshHandler := handlers.NewSSHHandler(store, containerManager)
	collabHandler := handlers.NewCollabHandler(store, containerManager, terminalHandler)
	recordingHandler := handlers.NewRecordingHandler(store, os.Getenv("RECORDINGS_PATH"), containerManager)
	// Connect recording handler to terminal handler to capture events
	terminalHandler.SetRecordingHandler(recordingHandler)
	// Connect collab handler to terminal handler for shared session access
	terminalHandler.SetCollabHandler(collabHandler)
	
	billingHandler := handlers.NewBillingHandler(billingService, store)

	// Initialize port forward handler
	portForwardHandler := handlers.NewPortForwardHandler(store, containerManager)

	// Initialize snippet handler
	snippetHandler := handlers.NewSnippetHandler(store)

	// Initialize token handler for API tokens
	tokenHandler := handlers.NewTokenHandler(store)

	// Initialize admin handler
	adminHandler := handlers.NewAdminHandler(store, adminEventsHub)

	// Initialize agent handler
	agentHandler := handlers.NewAgentHandler(store, jwtSecret)
	
	// Connect agent handler to container handler for unified API
	containerHandler.SetAgentHandler(agentHandler)
	
	// Connect events hub to agent handler for real-time updates
	agentHandler.SetEventsHub(containerEventsHub)
	
	// Connect agent handler to events hub for including agents in WebSocket list
	containerEventsHub.SetAgentHandler(agentHandler)

	// Connect Redis pub/sub for horizontal scaling
	if pubsubHub != nil {
		agentHandler.SetPubSubHub(pubsubHub)
		containerEventsHub.SetPubSubHub(pubsubHub)
		log.Println("✅ Handlers connected to Redis pub/sub")
	}
	_ = wsManager // Will be used for WebSocket management

	// Setup Gin router
	router := gin.Default()

	// Gzip compression for faster transfers (skip WebSocket)
	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/ws/", "/ws/admin/events"})))

	// Cache control middleware for static assets
	router.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Service worker - check order matters! Must be before generic .js check
		if path == "/sw.js" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Service-Worker-Allowed", "/")
		} else if strings.HasPrefix(path, "/assets/") {
			// Long cache for hashed assets (immutable)
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
			c.Header("Cache-Control", "public, max-age=86400")
		}
		c.Next()
	})

	// CORS middleware (see internal/api/middleware/cors.go)
	router.Use(middleware.CORSMiddleware())

	// Security Headers
	router.Use(middleware.SecurityHeaders())

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

	// Version endpoint for agent installer
	router.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "v1.0.0",
			"build":   "2024-12-10",
		})
	})

	// Auth routes (public) - strict rate limiting
	authGroup := router.Group("/api/auth") // Renamed to avoid conflict with `authHandler`
	authGroup.Use(authLimiter.Middleware())
	{
		// Guest login (1-hour session limit)
		authGroup.POST("/guest", authHandler.GuestLogin)

		// PipeOps OAuth routes
		authGroup.GET("/oauth/url", authHandler.GetOAuthURL)
		authGroup.GET("/callback", authHandler.OAuthCallback)
		authGroup.GET("/signin", authHandler.OAuthCallback) // Alternative callback path
		authGroup.POST("/oauth/exchange", authHandler.OAuthExchange)
	}

	// API routes (protected) - with rate limiting
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware(store, mfaService, jwtSecret))
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

		// MFA
		mfa := api.Group("/mfa")
		{
			mfa.GET("/setup", authHandler.SetupMFA)
			mfa.POST("/verify", authHandler.VerifyMFA)
			mfa.POST("/disable", authHandler.DisableMFA)
			mfa.POST("/validate", authHandler.ValidateMFA)
			mfa.POST("/complete-login", authHandler.CompleteMFALogin)
		}

		// Audit Logs
		api.GET("/audit-logs", authHandler.GetAuditLogs)

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
			snippets.PUT("/:id", snippetHandler.UpdateSnippet)
			snippets.DELETE("/:id", snippetHandler.DeleteSnippet)
			snippets.POST("/:id/use", snippetHandler.UseSnippet)
		}

		// API Tokens for CLI/API authentication
		tokens := api.Group("/tokens")
		{
			tokens.GET("", tokenHandler.ListTokens)
			tokens.POST("", tokenHandler.CreateToken)
			tokens.DELETE("/:id", tokenHandler.RevokeToken)
			tokens.DELETE("/:id/permanent", tokenHandler.DeleteToken)
		}

		// Token validation endpoint (for CLI)
		api.GET("/tokens/validate", tokenHandler.ValidateToken)

		// Public snippets marketplace (authenticated users can see who owns)
		api.GET("/snippets/marketplace", snippetHandler.ListPublicSnippets)

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

		// Billing endpoints
		billing := api.Group("/billing")
		{
			billing.GET("/plans", billingHandler.GetPlans)
			billing.GET("/subscription", billingHandler.GetSubscription)
			
			if billingService != nil {
				billing.GET("/history", billingHandler.GetBillingHistory)
				billing.POST("/checkout", billingHandler.CreateCheckoutSession)
				billing.POST("/portal", billingHandler.CreatePortalSession)
				billing.POST("/cancel", billingHandler.CancelSubscription)
			}
		}

		// Agent endpoints
		agents := api.Group("/agents")
		{
			agents.POST("/register", agentHandler.RegisterAgent)
			agents.GET("", agentHandler.ListAgents)
			agents.GET("/:id/status", agentHandler.GetAgentStatus)
			agents.DELETE("/:id", agentHandler.DeleteAgent)
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
			admin.GET("/agents", adminHandler.ListAgents)
		}
	}

	// Stripe webhook (public, verified by signature)
	if billingService != nil {
		router.POST("/api/billing/webhook", billingHandler.HandleWebhook)
	}

	// WebSocket terminal endpoint - with rate limiting
	router.GET("/ws/terminal/:containerId", wsLimiter.Middleware(), middleware.AuthMiddleware(store, mfaService, jwtSecret), terminalHandler.HandleWebSocket)

	// WebSocket collaboration endpoint
	router.GET("/ws/collab/:code", wsLimiter.Middleware(), middleware.AuthMiddleware(store, mfaService, jwtSecret), collabHandler.HandleCollabWebSocket)

	// WebSocket for Port Forwarding
	router.GET("/ws/port-forward/:forwardId", wsLimiter.Middleware(), middleware.AuthMiddleware(store, mfaService, jwtSecret), portForwardHandler.HandlePortForwardWebSocket)

	// HTTP Proxy for Port Forwarding (public access via UUID path)
	// Supports all HTTP methods for full web app proxying
	proxyGroup := router.Group("/p/:forwardId")
	{
		proxyGroup.Any("", portForwardHandler.HandleHTTPProxy)
		proxyGroup.Any("/*path", portForwardHandler.HandleHTTPProxy)
	}

	// WebSocket for Admin Events (NEW)
	router.GET("/ws/admin/events", wsLimiter.Middleware(), middleware.AuthMiddleware(store, mfaService, jwtSecret), middleware.AdminOnly(store), adminEventsHub.HandleWebSocket)

	// WebSocket for Agent connections
	router.GET("/ws/agent/:id", wsLimiter.Middleware(), agentHandler.HandleAgentWebSocket)
	router.GET("/ws/agent/:id/terminal", wsLimiter.Middleware(), agentHandler.HandleUserWebSocket)

	// Public recording access (no auth required for shared recordings)
	router.GET("/r/:token", recordingHandler.GetRecordingByToken)
	router.GET("/r/:token/stream", recordingHandler.StreamRecordingByToken)

	// Public snippets marketplace (no auth required, but authenticated users see ownership)
	router.GET("/api/marketplace/snippets", snippetHandler.ListPublicSnippets)

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
		router.StaticFile("/pwa-96x96.png", filepath.Join(webDir, "pwa-96x96.png"))
		router.StaticFile("/pwa-192x192.png", filepath.Join(webDir, "pwa-192x192.png"))
		router.StaticFile("/pwa-512x512.png", filepath.Join(webDir, "pwa-512x512.png"))
		router.StaticFile("/robots.txt", filepath.Join(webDir, "robots.txt"))
		router.StaticFile("/sitemap.xml", filepath.Join(webDir, "sitemap.xml"))
		router.StaticFile("/og-image.svg", filepath.Join(webDir, "og-image.svg"))
		router.StaticFile("/og-image.png", filepath.Join(webDir, "og-image.png"))
		router.StaticFile("/screenshot-desktop.png", filepath.Join(webDir, "screenshot-desktop.png"))
		router.StaticFile("/screenshot-mobile.png", filepath.Join(webDir, "screenshot-mobile.png"))

		// Apple touch icons - serve favicon for these requests
		router.StaticFile("/apple-touch-icon.png", filepath.Join(webDir, "favicon.svg"))
		router.StaticFile("/apple-touch-icon-precomposed.png", filepath.Join(webDir, "favicon.svg"))

		// Install scripts - served with correct content type for curl | bash
		scriptsDir := os.Getenv("SCRIPTS_DIR")
		if scriptsDir == "" {
			scriptsDir = "./scripts"
		}
		router.GET("/install-cli.sh", func(c *gin.Context) {
			c.Header("Content-Type", "text/x-shellscript")
			c.File(filepath.Join(scriptsDir, "install-cli.sh"))
		})
		router.GET("/install-agent.sh", func(c *gin.Context) {
			c.Header("Content-Type", "text/x-shellscript")
			c.File(filepath.Join(scriptsDir, "install-agent.sh"))
		})

		// Serve downloadable binaries (agent, cli)
		downloadsDir := os.Getenv("DOWNLOADS_DIR")
		if downloadsDir == "" {
			downloadsDir = "downloads"
		}
		if _, err := os.Stat(downloadsDir); err == nil {
			router.GET("/downloads/:filename", func(c *gin.Context) {
				filename := c.Param("filename")
				// Security: only allow specific filenames
				allowedPrefixes := []string{"rexec-agent-", "rexec-cli-", "rexec-"}
				allowed := false
				for _, prefix := range allowedPrefixes {
					if len(filename) > len(prefix) && filename[:len(prefix)] == prefix {
						allowed = true
						break
					}
				}
				if !allowed {
					c.JSON(404, gin.H{"error": "File not found"})
					return
				}
				filePath := filepath.Join(downloadsDir, filename)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					c.JSON(404, gin.H{"error": "Binary not found. Please build from source or wait for release."})
					return
				}
				c.Header("Content-Type", "application/octet-stream")
				c.Header("Content-Disposition", "attachment; filename="+filename)
				c.File(filePath)
			})
		}

		// SPA routes - serve index.html for client-side routing
		router.GET("/guides", func(c *gin.Context) {
			c.File(indexFile)
		})
		router.GET("/use-cases", func(c *gin.Context) {
			c.File(indexFile)
		})
		router.GET("/use-cases/:slug", func(c *gin.Context) {
			c.File(indexFile)
		})
		router.GET("/snippets", func(c *gin.Context) {
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

		// Explicitly serve index.html for known SPA routes to avoid /:id catch-all 404
		router.GET("/pricing", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/promo", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/admin", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/marketplace", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/billing", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/billing/success", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/billing/cancel", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/agents", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/docs", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/docs/agent", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/docs/cli", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Account routes
		router.GET("/account", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/account/settings", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/account/api", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/account/recordings", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Also support direct container ID or agent URL in URL path
		router.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			// Serve index.html if it looks like a container ID (64 hex chars or UUID) or agent URL
			if len(id) == 64 || (len(id) == 36 && id[8] == '-' && id[13] == '-') {
				c.File(indexFile)
				return
			}
			// Handle agent:uuid format
			if len(id) > 6 && id[:6] == "agent:" {
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
