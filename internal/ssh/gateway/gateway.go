package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	gossh "golang.org/x/crypto/ssh"
)

// Context keys for session data
type contextKey string

const (
	ctxKeyFingerprint contextKey = "fingerprint"
	ctxKeyUserID      contextKey = "user_id"
	ctxKeyUsername    contextKey = "username"
	ctxKeyEmail       contextKey = "email"
	ctxKeyToken       contextKey = "token"
	ctxKeyIsGuest     contextKey = "is_guest"
	ctxKeySessionID   contextKey = "session_id"
)

// Config holds gateway configuration
type Config struct {
	APIURL string
}

// Gateway handles SSH connections and authentication
type Gateway struct {
	config   Config
	sessions sync.Map // fingerprint -> *Session
	client   *http.Client
}

// Session represents an authenticated SSH session
type Session struct {
	ID          string
	Fingerprint string
	UserID      string
	Username    string
	Email       string
	Token       string
	IsGuest     bool
	CreatedAt   time.Time
	LastSeenAt  time.Time
}

// UserInfo represents user data from the API
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Tier     string `json:"tier"`
}

// New creates a new SSH gateway
func New(cfg Config) (*Gateway, error) {
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}

	return &Gateway{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// PublicKeyHandler handles SSH public key authentication
// This accepts ANY valid public key (terminal.shop style)
func (g *Gateway) PublicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	fingerprint := gossh.FingerprintSHA256(key)
	log.Printf("[SSH] Public key auth attempt: %s (fingerprint: %s)", ctx.User(), fingerprint)

	// Store fingerprint in context for later use
	ctx.SetValue(ctxKeyFingerprint, fingerprint)

	// Accept ALL valid public keys - this is the terminal.shop approach
	// The actual user lookup happens in SessionMiddleware
	return true
}

// PasswordHandler handles SSH password authentication
// Used for token-based auth when username is "token" or a known username
func (g *Gateway) PasswordHandler(ctx ssh.Context, password string) bool {
	username := ctx.User()
	log.Printf("[SSH] Password auth attempt for user: %s", username)

	// If username is "token", treat password as API token
	if username == "token" {
		userInfo, err := g.validateToken(password)
		if err != nil {
			log.Printf("[SSH] Token validation failed: %v", err)
			return false
		}

		ctx.SetValue(ctxKeyUserID, userInfo.ID)
		ctx.SetValue(ctxKeyUsername, userInfo.Username)
		ctx.SetValue(ctxKeyEmail, userInfo.Email)
		ctx.SetValue(ctxKeyToken, password)
		ctx.SetValue(ctxKeyIsGuest, false)
		log.Printf("[SSH] Token auth successful for user: %s", userInfo.Username)
		return true
	}

	// For other usernames, try to validate as username + password/token
	userInfo, err := g.validateCredentials(username, password)
	if err != nil {
		log.Printf("[SSH] Credential validation failed for %s: %v", username, err)
		return false
	}

	ctx.SetValue(ctxKeyUserID, userInfo.ID)
	ctx.SetValue(ctxKeyUsername, userInfo.Username)
	ctx.SetValue(ctxKeyEmail, userInfo.Email)
	ctx.SetValue(ctxKeyToken, password)
	ctx.SetValue(ctxKeyIsGuest, false)
	log.Printf("[SSH] Password auth successful for user: %s", userInfo.Username)
	return true
}

// SessionMiddleware creates or retrieves a session for the connection
func (g *Gateway) SessionMiddleware(next ssh.Handler) ssh.Handler {
	return func(sess ssh.Session) {
		ctx := sess.Context()
		fingerprint, _ := ctx.Value(ctxKeyFingerprint).(string)
		username := sess.User()

		log.Printf("[SSH] Session started: user=%s fingerprint=%s", username, fingerprint)

		// Check if we already have auth data from password auth
		if userID, ok := ctx.Value(ctxKeyUserID).(string); ok && userID != "" {
			// Already authenticated via password
			session := &Session{
				ID:          generateSessionID(),
				Fingerprint: fingerprint,
				UserID:      userID,
				Username:    ctx.Value(ctxKeyUsername).(string),
				Email:       ctx.Value(ctxKeyEmail).(string),
				Token:       ctx.Value(ctxKeyToken).(string),
				IsGuest:     false,
				CreatedAt:   time.Now(),
				LastSeenAt:  time.Now(),
			}
			g.sessions.Store(session.ID, session)
			ctx.SetValue(ctxKeySessionID, session.ID)
			next(sess)
			return
		}

		// Public key auth - look up or create session by fingerprint
		if fingerprint != "" {
			// Try to look up existing session by fingerprint
			session, err := g.lookupOrCreateSession(fingerprint, username)
			if err != nil {
				log.Printf("[SSH] Failed to create session: %v", err)
				wish.Fatalln(sess, "Failed to create session")
				return
			}

			ctx.SetValue(ctxKeySessionID, session.ID)
			ctx.SetValue(ctxKeyUserID, session.UserID)
			ctx.SetValue(ctxKeyUsername, session.Username)
			ctx.SetValue(ctxKeyEmail, session.Email)
			ctx.SetValue(ctxKeyToken, session.Token)
			ctx.SetValue(ctxKeyIsGuest, session.IsGuest)
		} else {
			// No fingerprint and no password auth - treat as guest
			session := &Session{
				ID:         generateSessionID(),
				IsGuest:    true,
				CreatedAt:  time.Now(),
				LastSeenAt: time.Now(),
			}
			g.sessions.Store(session.ID, session)
			ctx.SetValue(ctxKeySessionID, session.ID)
			ctx.SetValue(ctxKeyIsGuest, true)
		}

		next(sess)
	}
}

// LoggingMiddleware logs SSH session activity
func (g *Gateway) LoggingMiddleware(next ssh.Handler) ssh.Handler {
	return func(sess ssh.Session) {
		username := sess.User()
		remoteAddr := sess.RemoteAddr().String()
		log.Printf("[SSH] Connection from %s as %s", remoteAddr, username)

		start := time.Now()
		next(sess)
		duration := time.Since(start)

		log.Printf("[SSH] Session ended for %s (duration: %s)", username, duration)
	}
}

// TeaHandler returns the Bubble Tea program for a session
func (g *Gateway) TeaHandler(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
	ctx := sess.Context()

	// Get session info from context
	sessionID, _ := ctx.Value(ctxKeySessionID).(string)
	userID, _ := ctx.Value(ctxKeyUserID).(string)
	username, _ := ctx.Value(ctxKeyUsername).(string)
	email, _ := ctx.Value(ctxKeyEmail).(string)
	token, _ := ctx.Value(ctxKeyToken).(string)
	isGuest, _ := ctx.Value(ctxKeyIsGuest).(bool)
	fingerprint, _ := ctx.Value(ctxKeyFingerprint).(string)

	// Get terminal size
	pty, _, _ := sess.Pty()
	width := pty.Window.Width
	height := pty.Window.Height

	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	// Create the TUI model
	model := NewModel(ModelConfig{
		SessionID:   sessionID,
		UserID:      userID,
		Username:    username,
		Email:       email,
		Token:       token,
		IsGuest:     isGuest,
		Fingerprint: fingerprint,
		APIURL:      g.config.APIURL,
		Width:       width,
		Height:      height,
	})

	return model, []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}
}

// lookupOrCreateSession looks up an existing session by fingerprint or creates a new one
func (g *Gateway) lookupOrCreateSession(fingerprint, username string) (*Session, error) {
	// First, try to look up the fingerprint in the API
	session, err := g.lookupFingerprint(fingerprint)
	if err == nil && session != nil {
		// Found existing linked session
		session.LastSeenAt = time.Now()
		g.sessions.Store(session.ID, session)
		return session, nil
	}

	// If username is "guest" or lookup failed, create a guest session
	isGuest := username == "guest" || username == ""

	session = &Session{
		ID:          generateSessionID(),
		Fingerprint: fingerprint,
		IsGuest:     isGuest,
		CreatedAt:   time.Now(),
		LastSeenAt:  time.Now(),
	}

	// For non-guest usernames, try to look up by username
	if !isGuest && username != "" {
		userInfo, err := g.lookupUsername(username)
		if err == nil && userInfo != nil {
			session.UserID = userInfo.ID
			session.Username = userInfo.Username
			session.Email = userInfo.Email
			session.IsGuest = false
		}
	}

	g.sessions.Store(session.ID, session)
	return session, nil
}

// lookupFingerprint looks up a user by SSH fingerprint
func (g *Gateway) lookupFingerprint(fingerprint string) (*Session, error) {
	url := fmt.Sprintf("%s/api/internal/ssh/fingerprint/%s", g.config.APIURL, fingerprint)

	resp, err := g.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Not found is not an error
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Token    string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Session{
		ID:          generateSessionID(),
		Fingerprint: fingerprint,
		UserID:      result.UserID,
		Username:    result.Username,
		Email:       result.Email,
		Token:       result.Token,
		IsGuest:     false,
		CreatedAt:   time.Now(),
		LastSeenAt:  time.Now(),
	}, nil
}

// lookupUsername looks up a user by username
func (g *Gateway) lookupUsername(username string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/internal/ssh/user/%s", g.config.APIURL, username)

	resp, err := g.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// validateToken validates an API token and returns user info
func (g *Gateway) validateToken(token string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/auth/me", g.config.APIURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token (status %d)", resp.StatusCode)
	}

	var result struct {
		User UserInfo `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.User, nil
}

// validateCredentials validates username and password/token
func (g *Gateway) validateCredentials(username, password string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/auth/login", g.config.APIURL)

	payload := map[string]string{
		"email":    username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := g.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try as token instead
		return g.validateToken(password)
	}

	var result struct {
		User  UserInfo `json:"user"`
		Token string   `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.User, nil
}

// GetSession retrieves a session by ID
func (g *Gateway) GetSession(sessionID string) (*Session, bool) {
	if val, ok := g.sessions.Load(sessionID); ok {
		return val.(*Session), true
	}
	return nil, false
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("ssh_%d", time.Now().UnixNano())
}

// Helper to read response body
func readBody(r io.Reader) string {
	body, _ := io.ReadAll(r)
	return strings.TrimSpace(string(body))
}
