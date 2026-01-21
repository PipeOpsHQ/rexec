package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/rexec/rexec/internal/ssh/gateway"
)

const (
	defaultPort    = "2222"
	defaultHostKey = ".ssh/rexec_host_key"
)

func main() {
	// Parse command line flags
	port := flag.String("port", getEnv("SSH_GATEWAY_PORT", defaultPort), "SSH server port")
	hostKeyPath := flag.String("host-key", getEnv("SSH_GATEWAY_HOST_KEY", defaultHostKey), "Path to SSH host key")
	apiURL := flag.String("api-url", getEnv("SSH_GATEWAY_API_URL", "http://localhost:8080"), "Rexec API URL")
	flag.Parse()

	// Create the gateway
	gw, err := gateway.New(gateway.Config{
		APIURL: *apiURL,
	})
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}

	// Create wish server with middleware
	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort("", *port)),
		wish.WithHostKeyPath(*hostKeyPath),
		wish.WithPublicKeyAuth(gw.PublicKeyHandler),
		wish.WithPasswordAuth(gw.PasswordHandler),
		wish.WithMiddleware(
			bubbletea.Middleware(gw.TeaHandler),
			gw.LoggingMiddleware,
			gw.SessionMiddleware,
		),
	)
	if err != nil {
		log.Fatalf("Failed to create SSH server: %v", err)
	}

	// Handle shutdown gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Starting Rexec SSH Gateway on port %s", *port)
	log.Printf("API URL: %s", *apiURL)
	log.Printf("Host key: %s", *hostKeyPath)
	log.Println("")
	log.Println("Connect with: ssh -p " + *port + " localhost")
	log.Println("Or as guest:  ssh -p " + *port + " guest@localhost")

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Fatalf("SSH server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-done

	log.Println("Shutting down SSH gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("SSH gateway stopped")
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func printBanner() {
	fmt.Print(`
██████╗ ███████╗██╗  ██╗███████╗ ██████╗
██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝
██████╔╝█████╗   ╚███╔╝ █████╗  ██║
██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║
██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗
╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝
         SSH Gateway
`)
}
