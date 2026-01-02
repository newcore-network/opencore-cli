package txadmin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client handles communication with txAdmin API
type Client struct {
	baseURL       string
	username      string
	password      string
	httpClient    *http.Client
	csrfToken     string
	session       *Session
	sessionCookie string // Raw cookie string (name=value) - txAdmin uses non-RFC cookie names
}

// NewClient creates a new txAdmin client
func NewClient(baseURL, username, password string) (*Client, error) {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		},
	}

	return client, nil
}

// Login authenticates with txAdmin and stores the session
func (c *Client) Login() error {
	// Prepare login payload
	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login payload: %w", err)
	}

	// Make login request
	req, err := http.NewRequest("POST", c.baseURL+"/auth/password", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get CSRF token
	var loginResp struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}
	c.csrfToken = loginResp.CSRFToken

	// Manually extract session cookie from Set-Cookie header
	// txAdmin uses cookie names like "tx:abc123" which violate RFC 6265
	// and are rejected by Go's cookie jar
	setCookieHeaders := resp.Header["Set-Cookie"]
	for _, setCookie := range setCookieHeaders {
		// Format: "tx:abc123=value; path=/; ..."
		// We need to extract "tx:abc123=value"
		parts := strings.Split(setCookie, ";")
		if len(parts) > 0 {
			cookiePart := strings.TrimSpace(parts[0])
			if strings.HasPrefix(cookiePart, "tx:") {
				c.sessionCookie = cookiePart
				// Also extract name and value for session caching
				eqIdx := strings.Index(cookiePart, "=")
				if eqIdx > 0 {
					c.session = &Session{
						BaseURL:    c.baseURL,
						CookieName: cookiePart[:eqIdx],
						Cookie:     cookiePart[eqIdx+1:],
						CSRFToken:  c.csrfToken,
						ExpiresAt:  time.Now().Add(23 * time.Hour),
					}
				}
				break
			}
		}
	}

	if c.sessionCookie == "" {
		return fmt.Errorf("no session cookie received from txAdmin")
	}

	return nil
}

// extractCSRFFromCookies tries to get CSRF token from cookies
func (c *Client) extractCSRFFromCookies() string {
	parsedURL, _ := url.Parse(c.baseURL)
	for _, cookie := range c.httpClient.Jar.Cookies(parsedURL) {
		if strings.Contains(strings.ToLower(cookie.Name), "csrf") {
			return cookie.Value
		}
	}
	return ""
}

// RestoreSession restores a session from cache
func (c *Client) RestoreSession(session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if time.Now().After(session.ExpiresAt) {
		return fmt.Errorf("session has expired")
	}

	c.session = session
	c.csrfToken = session.CSRFToken
	// Restore the raw cookie string for manual header injection
	c.sessionCookie = session.CookieName + "=" + session.Cookie

	return nil
}

// GetSession returns the current session for caching
func (c *Client) GetSession() *Session {
	return c.session
}

// IsAuthenticated checks if the client has a valid session
func (c *Client) IsAuthenticated() bool {
	if c.session == nil {
		return false
	}
	return time.Now().Before(c.session.ExpiresAt)
}

// EnsureAuthenticated ensures the client is logged in, re-authenticating if needed
func (c *Client) EnsureAuthenticated() error {
	if c.IsAuthenticated() {
		return nil
	}
	return c.Login()
}

// ValidateSession checks if the current session is still valid on the server
// This is useful after restoring a cached session to ensure the server still recognizes it
func (c *Client) ValidateSession() error {
	if c.session == nil {
		return fmt.Errorf("no session to validate")
	}

	// Make a simple authenticated request to check if session is valid
	// We use /fxserver/commands as it's a known endpoint that requires authentication
	// Send an empty/invalid command - we just want to see if we get 401/403 or not
	payload := map[string]string{
		"action":    "status",
		"parameter": "",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal validation payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/fxserver/commands", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Add authentication headers
	if c.csrfToken != "" {
		req.Header.Set("x-txadmin-csrftoken", c.csrfToken)
	}
	if c.sessionCookie != "" {
		req.Header.Set("Cookie", c.sessionCookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	// If we get 401 or 403, the session is invalid
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("session is no longer valid on server (status %d)", resp.StatusCode)
	}

	// Any other status code means we're authenticated
	// (the command itself might fail, but we're not being rejected for auth)
	return nil
}

// RestartResource restarts a specific resource via txAdmin
func (c *Client) RestartResource(resourceName string) error {
	if err := c.EnsureAuthenticated(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return c.executeCommand("restart_res", resourceName)
}

// StartResource starts a specific resource via txAdmin
func (c *Client) StartResource(resourceName string) error {
	if err := c.EnsureAuthenticated(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return c.executeCommand("start_res", resourceName)
}

// StopResource stops a specific resource via txAdmin
func (c *Client) StopResource(resourceName string) error {
	if err := c.EnsureAuthenticated(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return c.executeCommand("stop_res", resourceName)
}

// RefreshResources refreshes all resources
func (c *Client) RefreshResources() error {
	if err := c.EnsureAuthenticated(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return c.executeCommand("refresh_res", "")
}

// CommandResponse represents txAdmin command response
// txAdmin uses: { type: "success" | "warning" | "error", msg: "..." }
type CommandResponse struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
}

// executeCommand sends a command to txAdmin
func (c *Client) executeCommand(action, parameter string) error {
	// txAdmin requires both action AND parameter to be defined
	// parameter can be empty string but must be present
	payload := map[string]string{
		"action":    action,
		"parameter": parameter,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal command payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/fxserver/commands", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create command request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.csrfToken != "" {
		req.Header.Set("x-txadmin-csrftoken", c.csrfToken)
	}
	// Manually add session cookie (txAdmin uses non-RFC cookie names with ":")
	if c.sessionCookie != "" {
		req.Header.Set("Cookie", c.sessionCookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send command request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read command response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// Session expired, clear it and return error with status code
		c.session = nil
		return fmt.Errorf("authentication failed (status %d): session expired or invalid", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("command failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to check for success
	// txAdmin returns { type: "success"|"warning"|"error", msg: "..." }
	// "warning" is also valid (means command was sent)
	var cmdResp CommandResponse
	if err := json.Unmarshal(body, &cmdResp); err == nil {
		if cmdResp.Type == "error" {
			return fmt.Errorf("txAdmin: %s", cmdResp.Msg)
		}
	}

	return nil
}

// HealthCheck verifies connectivity to txAdmin
func (c *Client) HealthCheck() error {
	req, err := http.NewRequest("GET", c.baseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("txAdmin is not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("txAdmin returned error status: %d", resp.StatusCode)
	}

	return nil
}
