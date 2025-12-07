package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	DefaultPipeOpsBaseURL = "https://staging.pipeops.sh"
	DefaultClientID       = "9714a0a39b0ab0701cac36c8a011d830"
)

// PKCEChallenge holds PKCE code verifier and challenge
type PKCEChallenge struct {
	CodeVerifier  string
	CodeChallenge string
	Method        string
}

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	ClientID    string
	BaseURL     string
	RedirectURI string
	Scopes      []string
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RedirectURL  string `json:"redirect_url,omitempty"`
}

// UserInfo represents user information from PipeOps
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
	Picture  string `json:"picture,omitempty"`
}

// PKCEOAuthService handles OAuth2 authentication with PKCE
type PKCEOAuthService struct {
	config *OAuthConfig
	client *http.Client
}

// NewPKCEOAuthService creates a new PKCE OAuth service
func NewPKCEOAuthService() *PKCEOAuthService {
	baseURL := os.Getenv("PIPEOPS_OAUTH_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultPipeOpsBaseURL
	}

	clientID := os.Getenv("PIPEOPS_CLIENT_ID")
	if clientID == "" {
		clientID = DefaultClientID
	}

	redirectURI := os.Getenv("PIPEOPS_REDIRECT_URI")
	if redirectURI == "" {
		// Default redirect URI - should be configured in production
		redirectURI = "http://localhost:8080/auth/signin"
	}

	return &PKCEOAuthService{
		config: &OAuthConfig{
			ClientID:    clientID,
			BaseURL:     baseURL,
			RedirectURI: redirectURI,
			Scopes:      []string{"user:read"},
		},
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// GeneratePKCEChallenge generates a PKCE code verifier and challenge
func GeneratePKCEChallenge() (*PKCEChallenge, error) {
	// Generate 32 random bytes for code verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Base64 URL encode the verifier (no padding)
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Create SHA256 hash of the verifier
	hash := sha256.Sum256([]byte(codeVerifier))

	// Base64 URL encode the hash (no padding)
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEChallenge{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        "S256",
	}, nil
}

// GenerateRandomState generates a random state parameter
func GenerateRandomState() (string, error) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(stateBytes), nil
}

// GetAuthorizationURL returns the authorization URL for OAuth flow
func (s *PKCEOAuthService) GetAuthorizationURL(state, codeChallenge string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {s.config.ClientID},
		"redirect_uri":          {s.config.RedirectURI},
		"scope":                 {strings.Join(s.config.Scopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"oauth":                 {"true"},
	}

	return fmt.Sprintf("%s/auth/signin?%s", s.config.BaseURL, params.Encode())
}

// ExchangeCodeForToken exchanges an authorization code for tokens
func (s *PKCEOAuthService) ExchangeCodeForToken(code, codeVerifier string) (*TokenResponse, error) {
	// Use form-encoded data for token exchange (standard OAuth2)
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.config.RedirectURI},
		"client_id":     {s.config.ClientID},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest("POST", s.config.BaseURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken uses the refresh token to obtain a new access token
func (s *PKCEOAuthService) RefreshToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {s.config.ClientID},
	}

	req, err := http.NewRequest("POST", s.config.BaseURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	return &tokenResp, nil
}

// GetUserInfo fetches user information using the access token
func (s *PKCEOAuthService) GetUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", s.config.BaseURL+"/oauth/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	return &userInfo, nil
}

// GetConfig returns the OAuth configuration
func (s *PKCEOAuthService) GetConfig() *OAuthConfig {
	return s.config
}
