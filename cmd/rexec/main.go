package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
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
	"github.com/rexec/rexec/internal/firecracker"
	"github.com/rexec/rexec/internal/providers"
	"github.com/rexec/rexec/internal/pubsub"
	sshgateway "github.com/rexec/rexec/internal/ssh/gateway"
	"github.com/rexec/rexec/internal/storage"
)

var (
	apiLimiter       = middleware.APIRateLimiter()
	authLimiter      = middleware.AuthRateLimiter()
	containerLimiter = middleware.ContainerRateLimiter()
	wsLimiter        = middleware.WebSocketRateLimiter()
)

// SEO overrides for server-rendered routes so curl/bots get correct metadata.
type seoConfig struct {
	Title              string
	Description        string
	OGTitle            string
	OGDescription      string
	OGType             string
	OGImage            string
	TwitterTitle       string
	TwitterDescription string
	TwitterImage       string
}

var (
	reTitleTag    = regexp.MustCompile(`(?is)<title[^>]*>.*?</title>`)
	reMetaContent = func(attrName, attrValue string) *regexp.Regexp {
		return regexp.MustCompile(`(?i)(<meta[^>]+` + attrName + `=["']` + regexp.QuoteMeta(attrValue) + `["'][^>]*content=["'])([^"']*)(["'][^>]*>)`)
	}
	reCanonicalHref = regexp.MustCompile(`(?i)(<link[^>]+rel=["']canonical["'][^>]*href=["'])([^"']*)(["'][^>]*>)`)
)

func applySEO(baseHTML string, seo seoConfig, canonical string) string {
	safe := func(s string) string {
		v := html.EscapeString(s)
		return strings.ReplaceAll(v, "$", "$$")
	}

	pageTitle := seo.Title
	pageDesc := seo.Description

	ogTitle := seo.OGTitle
	if ogTitle == "" {
		ogTitle = pageTitle
	}
	ogDesc := seo.OGDescription
	if ogDesc == "" {
		ogDesc = pageDesc
	}

	twitterTitle := seo.TwitterTitle
	if twitterTitle == "" {
		twitterTitle = pageTitle
	}
	twitterDesc := seo.TwitterDescription
	if twitterDesc == "" {
		twitterDesc = pageDesc
	}

	title := safe(pageTitle)
	desc := safe(pageDesc)
	ogTitleSafe := safe(ogTitle)
	ogDescSafe := safe(ogDesc)
	twitterTitleSafe := safe(twitterTitle)
	twitterDescSafe := safe(twitterDesc)
	canon := safe(canonical)

	out := reTitleTag.ReplaceAllString(baseHTML, fmt.Sprintf("<title>%s</title>", title))
	out = reMetaContent("name", "title").ReplaceAllString(out, "${1}"+title+"${3}")
	out = reMetaContent("name", "description").ReplaceAllString(out, "${1}"+desc+"${3}")
	out = reMetaContent("property", "og:title").ReplaceAllString(out, "${1}"+ogTitleSafe+"${3}")
	out = reMetaContent("property", "og:description").ReplaceAllString(out, "${1}"+ogDescSafe+"${3}")
	out = reMetaContent("property", "og:url").ReplaceAllString(out, "${1}"+canon+"${3}")
	out = reMetaContent("name", "twitter:title").ReplaceAllString(out, "${1}"+twitterTitleSafe+"${3}")
	out = reMetaContent("name", "twitter:description").ReplaceAllString(out, "${1}"+twitterDescSafe+"${3}")
	out = reMetaContent("name", "twitter:url").ReplaceAllString(out, "${1}"+canon+"${3}")
	out = reCanonicalHref.ReplaceAllString(out, "${1}"+canon+"${3}")
	if seo.OGType != "" {
		ogTypeSafe := safe(seo.OGType)
		out = reMetaContent("property", "og:type").ReplaceAllString(out, "${1}"+ogTypeSafe+"${3}")
	}
	// Handle image overrides for og:image and twitter:image
	if seo.OGImage != "" {
		ogImageSafe := safe(seo.OGImage)
		out = reMetaContent("property", "og:image").ReplaceAllString(out, "${1}"+ogImageSafe+"${3}")
		out = reMetaContent("property", "og:image:alt").ReplaceAllString(out, "${1}"+ogTitleSafe+"${3}")
	}
	if seo.TwitterImage != "" {
		twitterImageSafe := safe(seo.TwitterImage)
		out = reMetaContent("name", "twitter:image").ReplaceAllString(out, "${1}"+twitterImageSafe+"${3}")
		out = reMetaContent("name", "twitter:image:alt").ReplaceAllString(out, "${1}"+twitterTitleSafe+"${3}")
	}
	return out
}

func canonicalURL(c *gin.Context) string {
	base := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if base == "" {
		scheme := "https"
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS == nil {
			scheme = "http"
		}
		base = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}
	return base + c.Request.URL.Path
}

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

	// Initialize provider registry
	providerRegistry := providers.NewRegistry()
	
	// Register Docker provider (always available if Docker is running)
	dockerProvider := providers.NewDockerProvider(containerManager)
	providerRegistry.Register(dockerProvider)
	
	// Register Firecracker provider (if available)
	firecrackerManager, err := firecracker.NewManager()
	if err != nil {
		log.Printf("⚠️  Firecracker manager initialization failed: %v (Firecracker provider disabled)", err)
	} else {
		providerRegistry.Register(firecrackerManager)
		log.Println("✅ Firecracker provider registered")
	}

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
	securityHandler := handlers.NewSecurityHandler(store, jwtSecret)
	containerHandler := handlers.NewContainerHandler(containerManager, store, adminEventsHub)
	containerEventsHub := handlers.NewContainerEventsHub(containerManager, store)
	containerHandler.SetEventsHub(containerEventsHub)
	terminalHandler := handlers.NewTerminalHandler(containerManager, store, adminEventsHub)
	terminalHandler.SetProviderRegistry(providerRegistry) // Enable VM terminal support
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

	// Initialize tutorial handler
	tutorialHandler := handlers.NewTutorialHandler(store)

	// Initialize token handler for API tokens
	tokenHandler := handlers.NewTokenHandler(store)

	// Initialize sessions handler
	sessionsHandler := handlers.NewSessionsHandler(store)

	// Initialize admin handler
	adminHandler := handlers.NewAdminHandler(store, adminEventsHub)

	// Initialize agent handler
	agentHandler := handlers.NewAgentHandler(store, jwtSecret)

	// Connect agent handler to container handler for unified API
	containerHandler.SetAgentHandler(agentHandler)

	// Initialize VM handler with provider registry
	vmHandler := handlers.NewVMHandler(providerRegistry)

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

	// Optional: start pprof on a separate, opt-in listener (recommended: localhost + SSH tunnel).
	if os.Getenv("REXEC_PPROF") == "1" || strings.EqualFold(os.Getenv("REXEC_PPROF"), "true") {
		addr := os.Getenv("REXEC_PPROF_ADDR")
		if addr == "" {
			addr = "127.0.0.1:6060"
		}
		go func() {
			log.Printf("[pprof] Listening on %s", addr)
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Printf("[pprof] Stopped: %v", err)
			}
		}()
	}

	// Gzip compression for faster transfers (skip WebSocket)
	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/ws/", "/ws/admin/events"})))

	// Cache control middleware for static assets and HTML pages
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
		} else if strings.HasSuffix(path, ".html") ||
			path == "/" ||
			strings.HasPrefix(path, "/terminal/") ||
			strings.HasPrefix(path, "/agent:") ||
			strings.HasPrefix(path, "/join/") ||
			strings.HasPrefix(path, "/use-cases/") ||
			strings.HasPrefix(path, "/account") ||
			path == "/pricing" ||
			path == "/guides" ||
			path == "/resources" ||
			path == "/marketplace" ||
			path == "/admin" ||
			path == "/billing" ||
			path == "/docs" ||
			strings.HasPrefix(path, "/docs/") ||
			path == "/agents" {
			// HTML pages and SPA routes - no caching to ensure fresh content
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
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

	// Internal SSH Gateway routes (called by the SSH gateway, no user auth)
	// These should only be accessible from localhost or with an internal API key
	internalSSH := router.Group("/api/internal/ssh")
	internalSSH.Use(func(c *gin.Context) {
		// Only allow requests from localhost or with valid internal key
		internalKey := os.Getenv("SSH_GATEWAY_INTERNAL_KEY")
		if internalKey != "" {
			providedKey := c.GetHeader("X-Internal-Key")
			if providedKey != internalKey {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid internal key"})
				return
			}
		}
		c.Next()
	})
	{
		internalSSH.GET("/fingerprint/:fingerprint", sshHandler.LookupByFingerprint)
		internalSSH.POST("/fingerprint", sshHandler.RegisterFingerprint)
		internalSSH.GET("/user/:username", sshHandler.LookupByUsername)
		internalSSH.POST("/session", sshHandler.CreateSSHSession)
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

		// Security (server-enforced screen lock)
		api.GET("/security", securityHandler.GetScreenLock)
		api.PATCH("/security", securityHandler.UpdateSettings)
		api.POST("/security/passcode", securityHandler.SetPasscode)
		api.DELETE("/security/passcode", securityHandler.RemovePasscode)
		api.POST("/security/lock", securityHandler.Lock)
		api.POST("/security/unlock", securityHandler.Unlock)
		api.POST("/security/single-session", securityHandler.SetSingleSessionMode)

		// Terminal MFA lock (requires MFA to be enabled)
		api.GET("/security/terminal/:id/mfa-status", securityHandler.GetTerminalMFAStatus)
		api.POST("/security/terminal/:id/mfa-lock", securityHandler.LockTerminalWithMFA)
		api.POST("/security/terminal/:id/mfa-unlock", securityHandler.UnlockTerminalWithMFA)
		api.POST("/security/terminal/:id/mfa-verify", securityHandler.VerifyTerminalMFAAccess)

		// Auth sessions
		sessions := api.Group("/sessions")
		{
			sessions.GET("", sessionsHandler.List)
			sessions.DELETE("/:id", sessionsHandler.Revoke)
			sessions.POST("/revoke-others", sessionsHandler.RevokeOthers)
		}

		// MFA
		mfa := api.Group("/mfa")
		{
			mfa.GET("/setup", authHandler.SetupMFA)
			mfa.POST("/verify", authHandler.VerifyMFA)
			mfa.POST("/disable", authHandler.DisableMFA)
			mfa.POST("/validate", authHandler.ValidateMFA)
			mfa.POST("/complete-login", authHandler.CompleteMFALogin)
			mfa.GET("/backup-codes/count", authHandler.GetBackupCodesCount)
			mfa.POST("/backup-codes/regenerate", authHandler.RegenerateBackupCodes)
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
			agents.GET("/:id", agentHandler.GetAgent)
			agents.GET("/:id/status", agentHandler.GetAgentStatus)
			agents.PATCH("/:id", agentHandler.UpdateAgent)
			agents.DELETE("/:id", agentHandler.DeleteAgent)
		}

		// VM/Terminal endpoints (unified provider API)
		vms := api.Group("/vms")
		{
			vms.GET("", vmHandler.List)
			vms.POST("", containerLimiter.Middleware(), vmHandler.Create)
			vms.GET("/:id", vmHandler.Get)
			vms.DELETE("/:id", vmHandler.Delete)
			vms.POST("/:id/start", vmHandler.Start)
			vms.POST("/:id/stop", vmHandler.Stop)
		}

		// Provider endpoints
		api.GET("/providers", vmHandler.ListProviders)

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

			// Debug/runtime info (admin-only)
			admin.GET("/runtime", handlers.GetRuntimeStats)

			// Tutorial management (admin-only)
			tutorials := admin.Group("/tutorials")
			{
				tutorials.GET("", tutorialHandler.ListAllTutorials)
				tutorials.POST("", tutorialHandler.CreateTutorial)
				tutorials.PUT("/:id", tutorialHandler.UpdateTutorial)
				tutorials.DELETE("/:id", tutorialHandler.DeleteTutorial)
				tutorials.POST("/images", tutorialHandler.UploadImage)
			}
		}

		// Public tutorials endpoint (authenticated users)
		api.GET("/tutorials", tutorialHandler.ListPublicTutorials)
		api.GET("/tutorials/:id", tutorialHandler.GetTutorial)
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

	// Public tutorials (no auth required)
	router.GET("/api/public/tutorials", tutorialHandler.ListPublicTutorials)
	router.GET("/api/public/tutorials/images/*path", tutorialHandler.GetImage)

	// Serve static files (frontend)
	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		// Default to ./web directory relative to binary
		webDir = "web"
	}

	// Check if web directory exists
	if _, err := os.Stat(webDir); err == nil {
		indexFile := filepath.Join(webDir, "index.html")
		baseIndexHTML := ""
		if b, err := os.ReadFile(indexFile); err == nil {
			baseIndexHTML = string(b)
		} else {
			log.Printf("Warning: failed to read index.html for SEO overrides: %v", err)
		}

		docsSEO := seoConfig{
			Title:       "Documentation | Rexec - Terminal as a Service",
			Description: "Complete documentation for Rexec - learn about instant cloud terminals, BYOS agents, CLI tools, security features, and API integration.",
		}
		agentDocsSEO := seoConfig{
			Title:       "Agent Documentation | Rexec",
			Description: "Install and run rexec-agent to connect your own servers, VMs, or local machines as terminals in Rexec.",
		}
		cliDocsSEO := seoConfig{
			Title:       "CLI Documentation | Rexec",
			Description: "Install and use the rexec CLI/TUI to manage terminals, snippets, and agents from your local shell.",
		}
		embedDocsSEO := seoConfig{
			Title:       "Embeddable Terminal Widget | Rexec",
			Description: "Add a cloud terminal to any website with a single script tag. Like Google Cloud Shell for your docs and tutorials.",
		}
		guidesSEO := seoConfig{
			Title:        "Rexec Product Guide - Instant Terminal Architecture",
			Description:  "Learn how Rexec delivers instant access to Linux terminals while silently provisioning complex environments in the background.",
			OGTitle:      "Rexec Product Guide",
			TwitterTitle: "Rexec Product Guide",
		}
		useCasesSEO := seoConfig{
			Title:        "Rexec Use Cases - The Future of Development",
			Description:  "Discover how Rexec powers ephemeral development environments, AI agent execution, collaborative coding, and secure cloud access.",
			OGTitle:      "Rexec Use Cases",
			TwitterTitle: "Rexec Use Cases",
		}
		pricingSEO := seoConfig{
			Title:       "Pricing | Rexec - Terminal as a Service",
			Description: "Simple, transparent pricing for instant cloud terminals, recordings, and BYOS agent connections.",
		}
		marketplaceSEO := seoConfig{
			Title:       "Marketplace | Rexec",
			Description: "Browse public snippets and automation templates for Rexec terminals.",
		}
		promoSEO := seoConfig{
			Title:       "Promotions | Rexec",
			Description: "Limited-time offers and promos for Rexec.",
		}
		launchSEO := seoConfig{
			Title:       "Rexec Launch - Terminal as a Service | Cloud Terminals & Secure Agent",
			Description: "Instant cloud terminals for developers. Safely run AI-generated code, access servers without SSH exposure, and spin up dev environments in seconds. Try free today.",
		}
		snippetsSEO := seoConfig{
			Title:       "Snippets | Rexec",
			Description: "Create, share, and run reusable scripts and macros on Rexec terminals.",
		}
		resourcesSEO := seoConfig{
			Title:       "Resources | Rexec - Guides & Tutorials",
			Description: "Learn how to use Rexec with step-by-step video tutorials and comprehensive guides covering terminals, agents, CLI tools, and more.",
		}
		useCaseDetailSEO := map[string]seoConfig{
			"collaborative-intelligence": {
				Title:              "Collaborative Intelligence | Rexec - Cloud Development Environment",
				Description:        "Let LLMs and AI agents execute code in a real, safe environment while you supervise. Rexec provides the perfect sandbox for autonomous agents to work alongside humans, with full visibility and control over their actions.",
				OGTitle:            "Collaborative Intelligence - Rexec",
				OGDescription:      "A shared workspace for humans and AI agents. Let LLMs and AI agents execute code in a real, safe environment while you supervise. Rexec provides the perfect sandbox for autonomous agents to work alongside humans, with full visibility and control over their actions.",
				OGType:             "article",
				TwitterTitle:       "Collaborative Intelligence - Rexec",
				TwitterDescription: "A shared workspace for humans and AI agents.",
			},
			"edge-device-development": {
				Title:              "Edge Device Development | Rexec - Cloud Development Environment",
				Description:        "Develop and test applications for IoT and edge devices in a simulated or emulated environment. Cross-compile for ARM, RISC-V, and other architectures without physical hardware.",
				OGTitle:            "Edge Device Development - Rexec",
				OGDescription:      "Develop for IoT and edge in the cloud. Develop and test applications for IoT and edge devices in a simulated or emulated environment. Cross-compile for ARM, RISC-V, and other architectures without physical hardware.",
				OGType:             "article",
				TwitterTitle:       "Edge Device Development - Rexec",
				TwitterDescription: "Develop for IoT and edge in the cloud.",
			},
			"ephemeral-dev-environments": {
				Title:              "Ephemeral Dev Environments | Rexec - Cloud Development Environment",
				Description:        "Spin up a fresh, clean environment for every task, PR, or experiment. Ephemeral environments eliminate configuration drift, dependency conflicts, and the dreaded 'works on my machine' syndrome. Each session starts from a known state, ensuring reproducible results every time.",
				OGTitle:            "Ephemeral Dev Environments - Rexec",
				OGDescription:      "The future is disposable. Zero drift, zero cleanup. Spin up a fresh, clean environment for every task, PR, or experiment. Ephemeral environments eliminate configuration drift, dependency conflicts, and the dreaded 'works on my machine' syndrome. Each session starts from a known state, ensuring reproducible results every time.",
				OGType:             "article",
				TwitterTitle:       "Ephemeral Dev Environments - Rexec",
				TwitterDescription: "The future is disposable. Zero drift, zero cleanup.",
			},
			"gpu-terminals": {
				Title:              "GPU Terminals for AI/ML | Rexec - Cloud Development Environment",
				Description:        "Rexec will provide instant-on, powerful GPU-enabled terminals for your team's AI/ML model development, training, and fine-tuning. Manage and share these dedicated GPU resources securely, eliminating the complexities of direct infrastructure access.",
				OGTitle:            "GPU Terminals for AI/ML - Rexec",
				OGDescription:      "Instant-on GPU power for your AI/ML workflows. Rexec will provide instant-on, powerful GPU-enabled terminals for your team's AI/ML model development, training, and fine-tuning. Manage and share these dedicated GPU resources securely, eliminating the complexities of direct infrastructure access.",
				OGType:             "article",
				TwitterTitle:       "GPU Terminals for AI/ML - Rexec",
				TwitterDescription: "Instant-on GPU power for your AI/ML workflows.",
			},
			"hybrid-infrastructure": {
				Title:              "Hybrid Infrastructure Access | Rexec - Cloud Development Environment",
				Description:        "Access everything through a single, unified interface. Seamlessly switch between Rexec's cloud terminals and your on-premise servers without changing tools or context.",
				OGTitle:            "Hybrid Infrastructure Access - Rexec",
				OGDescription:      "Mix cloud-managed terminals with your own infrastructure. Access everything through a single, unified interface. Seamlessly switch between Rexec's cloud terminals and your on-premise servers without changing tools or context.",
				OGType:             "article",
				TwitterTitle:       "Hybrid Infrastructure Access - Rexec",
				TwitterDescription: "Mix cloud-managed terminals with your own infrastructure.",
			},
			"instant-education-onboarding": {
				Title:              "Instant Education & Onboarding | Rexec - Cloud Development Environment",
				Description:        "Provide pre-configured environments for workshops, tutorials, and new hire onboarding. Zero friction means attendees focus on learning, not configuring their machines.",
				OGTitle:            "Instant Education & Onboarding - Rexec",
				OGDescription:      "Onboard new engineers in seconds, not days. Provide pre-configured environments for workshops, tutorials, and new hire onboarding. Zero friction means attendees focus on learning, not configuring their machines.",
				OGType:             "article",
				TwitterTitle:       "Instant Education & Onboarding - Rexec",
				TwitterDescription: "Onboard new engineers in seconds, not days.",
			},
			"open-source-review": {
				Title:              "Open Source Review | Rexec - Cloud Development Environment",
				Description:        "Review Pull Requests by instantly spinning up the branch in a clean container. Test without polluting your local machine or risking your development environment.",
				OGTitle:            "Open Source Review - Rexec",
				OGDescription:      "Review PRs in isolated, disposable environments. Review Pull Requests by instantly spinning up the branch in a clean container. Test without polluting your local machine or risking your development environment.",
				OGType:             "article",
				TwitterTitle:       "Open Source Review - Rexec",
				TwitterDescription: "Review PRs in isolated, disposable environments.",
			},
			"real-time-data-processing": {
				Title:              "Real-time Data Processing | Rexec - Cloud Development Environment",
				Description:        "Build, test, and deploy streaming ETL pipelines and real-time analytics applications. High-performance data ingress/egress with monitoring and debugging tools.",
				OGTitle:            "Real-time Data Processing - Rexec",
				OGDescription:      "Build streaming pipelines in isolated sandboxes. Build, test, and deploy streaming ETL pipelines and real-time analytics applications. High-performance data ingress/egress with monitoring and debugging tools.",
				OGType:             "article",
				TwitterTitle:       "Real-time Data Processing - Rexec",
				TwitterDescription: "Build streaming pipelines in isolated sandboxes.",
			},
			"resumable-sessions": {
				Title:              "Resumable Terminal Sessions | Rexec - Cloud Development Environment",
				Description:        "Run long-running processes, close your browser, and reconnect anytime. Your terminal session continues in the background with full scrollback history. Never lose work to network drops or accidental tab closures again.",
				OGTitle:            "Resumable Terminal Sessions - Rexec",
				OGDescription:      "Start tasks, disconnect, and come back later. Run long-running processes, close your browser, and reconnect anytime. Your terminal session continues in the background with full scrollback history. Never lose work to network drops or accidental tab closures again.",
				OGType:             "article",
				TwitterTitle:       "Resumable Terminal Sessions - Rexec",
				TwitterDescription: "Start tasks, disconnect, and come back later.",
			},
			"rexec-agent": {
				Title:              "Hybrid Cloud & Remote Agents | Rexec - Cloud Development Environment",
				Description:        "Turn any Linux server, IoT device, or cloud instance into a managed Rexec terminal. Install our lightweight binary to instantly connect remote resources to your Rexec dashboard with real-time resource monitoring.",
				OGTitle:            "Hybrid Cloud & Remote Agents - Rexec",
				OGDescription:      "Unify your infrastructure. One dashboard for everything. Turn any Linux server, IoT device, or cloud instance into a managed Rexec terminal. Install our lightweight binary to instantly connect remote resources to your Rexec dashboard with real-time resource monitoring.",
				OGType:             "article",
				TwitterTitle:       "Hybrid Cloud & Remote Agents - Rexec",
				TwitterDescription: "Unify your infrastructure. One dashboard for everything.",
			},
			"rexec-cli": {
				Title:              "Rexec CLI & TUI | Rexec - Cloud Development Environment",
				Description:        "The Rexec CLI brings the power of the platform to your local terminal. Manage sessions, ssh into containers, and use the interactive TUI dashboard without leaving your keyboard.",
				OGTitle:            "Rexec CLI & TUI - Rexec",
				OGDescription:      "Manage your terminals from anywhere using our powerful command-line interface. The Rexec CLI brings the power of the platform to your local terminal. Manage sessions, ssh into containers, and use the interactive TUI dashboard without leaving your keyboard.",
				OGType:             "article",
				TwitterTitle:       "Rexec CLI & TUI - Rexec",
				TwitterDescription: "Manage your terminals from anywhere using our powerful command-line interface.",
			},
			"technical-interviews": {
				Title:              "Technical Interviews | Rexec - Cloud Development Environment",
				Description:        "Conduct real-time coding interviews in a real Linux environment, not a constrained web editor. See how candidates actually work, not just whether they can pass synthetic tests.",
				OGTitle:            "Technical Interviews - Rexec",
				OGDescription:      "Real coding assessments in real environments. Conduct real-time coding interviews in a real Linux environment, not a constrained web editor. See how candidates actually work, not just whether they can pass synthetic tests.",
				OGType:             "article",
				TwitterTitle:       "Technical Interviews - Rexec",
				TwitterDescription: "Real coding assessments in real environments.",
			},
			"universal-jump-host": {
				Title:              "Secure Jump Host & Gateway | Rexec - Cloud Development Environment",
				Description:        "Replace complex VPNs and bastion hosts. Rexec provides a secure, audited gateway to your private infrastructure. Enforce MFA, restrict IP access, and log every command for complete compliance and security.",
				OGTitle:            "Secure Jump Host & Gateway - Rexec",
				OGDescription:      "Zero-trust access to private infrastructure. Replace complex VPNs and bastion hosts. Rexec provides a secure, audited gateway to your private infrastructure. Enforce MFA, restrict IP access, and log every command for complete compliance and security.",
				OGType:             "article",
				TwitterTitle:       "Secure Jump Host & Gateway - Rexec",
				TwitterDescription: "Zero-trust access to private infrastructure.",
			},
			"remote-debugging": {
				Title:              "Remote Debugging & Troubleshooting | Rexec - Cloud Development Environment",
				Description:        "Connect to any server running the Rexec agent for instant access. Troubleshoot live systems with full terminal capabilities, share sessions with colleagues, and resolve incidents faster.",
				OGTitle:            "Remote Debugging & Troubleshooting - Rexec",
				OGDescription:      "Debug production issues directly from your browser. Connect to any server running the Rexec agent for instant access. Troubleshoot live systems with full terminal capabilities, share sessions with colleagues, and resolve incidents faster.",
				OGType:             "article",
				TwitterTitle:       "Remote Debugging & Troubleshooting - Rexec",
				TwitterDescription: "Debug production issues directly from your browser.",
			},
		}
		serveSEO := func(c *gin.Context, seo seoConfig) {
			if baseIndexHTML == "" {
				c.File(indexFile)
				return
			}
			canonical := canonicalURL(c)
			html := applySEO(baseIndexHTML, seo, canonical)
			c.Data(200, "text/html; charset=utf-8", []byte(html))
		}

		router.StaticFile("/", indexFile)
		// Explicitly handle /index.html to avoid redirect issues with Service Worker
		router.GET("/index.html", func(c *gin.Context) {
			c.File(indexFile)
		})
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
		router.StaticFile("/ai.txt", filepath.Join(webDir, "ai.txt"))
		router.StaticFile("/.well-known/ai-plugin.json", filepath.Join(webDir, ".well-known/ai-plugin.json"))
		router.StaticFile("/og-image.svg", filepath.Join(webDir, "og-image.svg"))
		router.StaticFile("/og-image.png", filepath.Join(webDir, "og-image.png"))
		router.StaticFile("/screenshot-desktop.png", filepath.Join(webDir, "screenshot-desktop.png"))
		router.StaticFile("/screenshot-mobile.png", filepath.Join(webDir, "screenshot-mobile.png"))

		// Apple touch icons - serve favicon for these requests
		router.StaticFile("/apple-touch-icon.png", filepath.Join(webDir, "favicon.svg"))
		router.StaticFile("/apple-touch-icon-precomposed.png", filepath.Join(webDir, "favicon.svg"))

		// Embed widget - serve CDN bundle for embeddable terminal widget
		// Use individual StaticFile routes with a group for CORS headers
		embedDir := filepath.Join(webDir, "embed")
		if _, err := os.Stat(embedDir); err == nil {
			embedGroup := router.Group("/embed")
			embedGroup.Use(func(c *gin.Context) {
				// CORS headers for CDN usage
				c.Header("Access-Control-Allow-Origin", "*")
				c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
				c.Header("Cache-Control", "public, max-age=86400") // 24 hour cache
				c.Next()
			})
			embedGroup.StaticFile("/rexec.min.js", filepath.Join(embedDir, "rexec.min.js"))
			embedGroup.StaticFile("/rexec.esm.js", filepath.Join(embedDir, "rexec.esm.js"))
			embedGroup.StaticFile("/rexec.min.js.map", filepath.Join(embedDir, "rexec.min.js.map"))
			embedGroup.StaticFile("/rexec.esm.js.map", filepath.Join(embedDir, "rexec.esm.js.map"))
			embedGroup.StaticFile("/rexec.css", filepath.Join(embedDir, "rexec.css"))
		}

		// Install scripts - served with correct content type for curl | bash
		scriptsDir := os.Getenv("SCRIPTS_DIR")
		if scriptsDir == "" {
			scriptsDir = "./scripts"
		}
		router.GET("/install-cli.sh", func(c *gin.Context) {
			scriptPath := filepath.Join(scriptsDir, "install-cli.sh")
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				c.String(404, "#!/bin/bash\necho 'Error: install script not found on server'\nexit 1\n")
				return
			}
			c.Header("Content-Type", "text/x-shellscript")
			c.File(scriptPath)
		})
		router.GET("/install-agent.sh", func(c *gin.Context) {
			scriptPath := filepath.Join(scriptsDir, "install-agent.sh")
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				c.String(404, "#!/bin/bash\necho 'Error: install script not found on server'\nexit 1\n")
				return
			}
			c.Header("Content-Type", "text/x-shellscript")
			c.File(scriptPath)
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
			serveSEO(c, guidesSEO)
		})
		router.GET("/use-cases", func(c *gin.Context) {
			serveSEO(c, useCasesSEO)
		})
		router.GET("/use-cases/:slug", func(c *gin.Context) {
			slug := c.Param("slug")
			if seo, ok := useCaseDetailSEO[slug]; ok {
				serveSEO(c, seo)
				return
			}
			serveSEO(c, useCasesSEO)
		})
		router.GET("/snippets", func(c *gin.Context) {
			serveSEO(c, snippetsSEO)
		})
		// Legacy routes - redirect or serve index
		router.GET("/ai-tools", func(c *gin.Context) {
			c.File(indexFile)
		})
		router.GET("/agentic", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Terminal URL routes - serve index.html for SPA routing
		// Use wildcard to handle both container IDs and agent:uuid format
		router.GET("/terminal/*path", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Join session route
		router.GET("/join/:code", func(c *gin.Context) {
			c.File(indexFile)
		})

		// Explicitly serve index.html for known SPA routes to avoid /:id catch-all 404
		router.GET("/pricing", func(c *gin.Context) {
			serveSEO(c, pricingSEO)
		})

		router.GET("/promo", func(c *gin.Context) {
			serveSEO(c, promoSEO)
		})

		router.GET("/launch", func(c *gin.Context) {
			serveSEO(c, launchSEO)
		})

		router.GET("/admin", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/marketplace", func(c *gin.Context) {
			serveSEO(c, marketplaceSEO)
		})

		router.GET("/resources", func(c *gin.Context) {
			id := c.Query("id")
			if id != "" {
				if tutorial, err := store.GetTutorialByID(c.Request.Context(), id); err == nil && tutorial.IsPublished {
					// Determine the thumbnail image URL
					thumbnail := tutorial.Thumbnail
					if thumbnail == "" && tutorial.Type == "video" && tutorial.VideoURL != "" {
						// Generate YouTube thumbnail if possible
						ytMatch := regexp.MustCompile(`(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/\s]{11})`).FindStringSubmatch(tutorial.VideoURL)
						if len(ytMatch) > 1 {
							thumbnail = fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", ytMatch[1])
						}
					}
					// Fall back to default og-image if no thumbnail
					if thumbnail == "" {
						thumbnail = canonicalURL(c)
						// Extract base URL (without path)
						if idx := strings.Index(thumbnail, "/resources"); idx != -1 {
							thumbnail = thumbnail[:idx] + "/og-image.png"
						} else {
							thumbnail = "/og-image.png"
						}
					}
					customSEO := seoConfig{
						Title:              fmt.Sprintf("%s | Rexec Resources", tutorial.Title),
						Description:        tutorial.Description,
						OGTitle:            tutorial.Title,
						OGDescription:      tutorial.Description,
						OGType:             "article",
						OGImage:            thumbnail,
						TwitterTitle:       tutorial.Title,
						TwitterDescription: tutorial.Description,
						TwitterImage:       thumbnail,
					}
					serveSEO(c, customSEO)
					return
				}
			}
			serveSEO(c, resourcesSEO)
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
			serveSEO(c, docsSEO)
		})

		router.GET("/docs/agent", func(c *gin.Context) {
			serveSEO(c, agentDocsSEO)
		})

		router.GET("/docs/cli", func(c *gin.Context) {
			serveSEO(c, cliDocsSEO)
		})

		router.GET("/docs/embed", func(c *gin.Context) {
			serveSEO(c, embedDocsSEO)
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

		router.GET("/account/snippets", func(c *gin.Context) {
			serveSEO(c, snippetsSEO)
		})

		router.GET("/account/ssh", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/account/sshkeys", func(c *gin.Context) {
			c.File(indexFile)
		})

		router.GET("/account/billing", func(c *gin.Context) {
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
			path := c.Request.URL.Path
			// Don't mask API/WebSocket 404s with HTML; return JSON instead.
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/") {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			c.File(indexFile)
		})
	}

	// Get port from env or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Rexec server starting on port %s", port)

	// Start SSH gateway if enabled
	// Use SSH_GATEWAY_ENABLED=true to enable on default port 22
	// Or SSH_GATEWAY_PORT=2222 for custom port
	sshEnabled := os.Getenv("SSH_GATEWAY_ENABLED") == "true"
	sshPort := os.Getenv("SSH_GATEWAY_PORT")
	if sshEnabled && sshPort == "" {
		sshPort = "22" // Default SSH port
	}
	if sshPort != "" {
		go startSSHGateway(sshPort, port)
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// startSSHGateway starts the SSH gateway server
func startSSHGateway(sshPort, apiPort string) {
	hostKeyPath := os.Getenv("SSH_GATEWAY_HOST_KEY")
	if hostKeyPath == "" {
		hostKeyPath = ".ssh/rexec_host_key"
	}

	// Determine API URL for the gateway to connect back to
	apiURL := os.Getenv("SSH_GATEWAY_API_URL")
	if apiURL == "" {
		apiURL = fmt.Sprintf("http://localhost:%s", apiPort)
	}

	// Create the gateway
	gw, err := sshgateway.New(sshgateway.Config{
		APIURL: apiURL,
	})
	if err != nil {
		log.Printf("⚠️  Failed to create SSH gateway: %v", err)
		return
	}

	// Check if host key exists, generate if not
	if _, err := os.Stat(hostKeyPath); os.IsNotExist(err) {
		log.Printf("⚠️  SSH host key not found at %s, SSH gateway disabled", hostKeyPath)
		log.Printf("   Generate one with: ssh-keygen -t ed25519 -f %s -N \"\"", hostKeyPath)
		return
	}

	// Create wish server with middleware
	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort("", sshPort)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithPublicKeyAuth(gw.PublicKeyHandler),
		wish.WithPasswordAuth(gw.PasswordHandler),
		wish.WithMiddleware(
			bubbletea.Middleware(gw.TeaHandler),
			gw.LoggingMiddleware,
			gw.SessionMiddleware,
		),
	)
	if err != nil {
		log.Printf("⚠️  Failed to create SSH server: %v", err)
		return
	}

	log.Printf("🔐 SSH Gateway starting on port %s", sshPort)
	if sshPort == "22" {
		log.Printf("   Connect with: ssh localhost")
	} else {
		log.Printf("   Connect with: ssh -p %s localhost", sshPort)
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Printf("⚠️  SSH server error: %v", err)
	}
}
