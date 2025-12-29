package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// APIClient handles communication with the Rexec API
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// doRequest performs an authenticated HTTP request
func (c *APIClient) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// FetchTerminals fetches containers from the API
func (c *APIClient) FetchTerminals() ([]Terminal, error) {
	resp, err := c.doRequest("GET", "/api/containers", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch terminals: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Containers []Terminal `json:"containers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Try decoding as array directly
		resp.Body.Close()
		resp, _ = c.doRequest("GET", "/api/containers", nil)
		defer resp.Body.Close()

		var terminals []Terminal
		if err := json.NewDecoder(resp.Body).Decode(&terminals); err != nil {
			return nil, fmt.Errorf("failed to decode terminals: %w", err)
		}
		return terminals, nil
	}

	return result.Containers, nil
}

// FetchAgents fetches agents from the API
func (c *APIClient) FetchAgents() ([]Agent, error) {
	resp, err := c.doRequest("GET", "/api/agents", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Agents []Agent `json:"agents"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Try decoding as array directly
		resp.Body.Close()
		resp, _ = c.doRequest("GET", "/api/agents", nil)
		defer resp.Body.Close()

		var agents []Agent
		if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
			return nil, fmt.Errorf("failed to decode agents: %w", err)
		}
		return agents, nil
	}

	return result.Agents, nil
}

// FetchSnippets fetches snippets from the API
func (c *APIClient) FetchSnippets() ([]Snippet, error) {
	resp, err := c.doRequest("GET", "/api/snippets", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch snippets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Snippets []Snippet `json:"snippets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Try decoding as array directly
		resp.Body.Close()
		resp, _ = c.doRequest("GET", "/api/snippets", nil)
		defer resp.Body.Close()

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			return nil, fmt.Errorf("failed to decode snippets: %w", err)
		}
		return snippets, nil
	}

	return result.Snippets, nil
}

// LinkAccount links an SSH fingerprint to an account
func (c *APIClient) LinkAccount(fingerprint, emailOrToken string) error {
	// First try as token
	testClient := NewAPIClient(c.baseURL, emailOrToken)
	resp, err := testClient.doRequest("GET", "/api/auth/me", nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		// Valid token - link fingerprint
		return c.linkFingerprint(fingerprint, emailOrToken)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// TODO: Handle email-based linking (send verification email)
	return fmt.Errorf("invalid token or email linking not yet implemented")
}

// linkFingerprint links a fingerprint to a user account
func (c *APIClient) linkFingerprint(fingerprint, token string) error {
	// This would call an internal API endpoint to link the fingerprint
	// For now, this is a placeholder
	_ = fingerprint
	_ = token
	return nil
}

// CreateTerminalCmd returns a tea.Cmd that fetches terminals
func CreateTerminalCmd(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		terminals, err := client.FetchTerminals()
		if err != nil {
			return errMsg{err: err}
		}
		return terminalsMsg(terminals)
	}
}

// CreateAgentCmd returns a tea.Cmd that fetches agents
func CreateAgentCmd(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		agents, err := client.FetchAgents()
		if err != nil {
			return errMsg{err: err}
		}
		return agentsMsg(agents)
	}
}

// CreateSnippetCmd returns a tea.Cmd that fetches snippets
func CreateSnippetCmd(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		snippets, err := client.FetchSnippets()
		if err != nil {
			return errMsg{err: err}
		}
		return snippetsMsg(snippets)
	}
}

// CreateLinkAccountCmd returns a tea.Cmd that attempts to link an account
func CreateLinkAccountCmd(client *APIClient, fingerprint, emailOrToken string) tea.Cmd {
	return func() tea.Msg {
		err := client.LinkAccount(fingerprint, emailOrToken)
		if err != nil {
			return linkFailMsg{err: err.Error()}
		}
		return linkSuccessMsg{}
	}
}
