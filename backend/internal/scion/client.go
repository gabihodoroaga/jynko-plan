package scion

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	// ErrAgentNotFound is returned when Scion Hub reports 404 for an agent.
	ErrAgentNotFound = errors.New("scion agent not found")
	// ErrConflict is returned when Scion Hub reports 409 Conflict (e.g. duplicate slug).
	ErrConflict = errors.New("scion conflict")
)

// HubClient defines the Scion Hub REST API operations required by Jynko.
type HubClient interface {
	ListAgents(ctx context.Context, projectSlug string) ([]Agent, error)
	GetAgent(ctx context.Context, agentID string) (*Agent, error)
	CreateAgent(ctx context.Context, req CreateAgentRequest) (*Agent, error)
	StartAgent(ctx context.Context, agentID string) (*Agent, error)
	StopAgent(ctx context.Context, agentID string) error
	SendMessage(ctx context.Context, agentID, messageText string, wake bool) error
	EnsureRunningAndSendMessage(ctx context.Context, agentID, messageText string) error
}

// ClientConfig configures the Scion Hub REST client.
type ClientConfig struct {
	BaseURL      string
	Token        string
	IapAudience  string
	HTTPClient   *http.Client
	PollInterval time.Duration
	StartTimeout time.Duration
}

// Agent represents a Scion Hub agent summary/status.
type Agent struct {
	ID            string `json:"id"`
	Slug          string `json:"slug,omitempty"`
	Name          string `json:"name"`
	ProjectID     string `json:"projectId,omitempty"`
	Project       string `json:"project,omitempty"`
	Template      string `json:"template,omitempty"`
	Harness       string `json:"harness,omitempty"`
	HarnessConfig string `json:"harnessConfig,omitempty"`
	Phase         string `json:"phase"`
	Activity      string `json:"activity,omitempty"`
	URL           string `json:"url,omitempty"`
}

func normalizeAgent(a *Agent) {
	if a == nil {
		return
	}
	if a.Harness == "" && a.HarnessConfig != "" {
		a.Harness = a.HarnessConfig
	}
}

// AgentConfig holds runtime container configuration passed to Scion Hub on agent creation.
type AgentConfig struct {
	Harness       string            `json:"harness,omitempty"`
	HarnessConfig string            `json:"harness_config,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
}

// CreateAgentRequest represents the payload sent to POST /api/v1/agents.
type CreateAgentRequest struct {
	Name              string       `json:"name"`
	ProjectID         string       `json:"projectId"`
	Template          string       `json:"template,omitempty"`
	HarnessConfig     string       `json:"harnessConfig,omitempty"`
	Task              string       `json:"task,omitempty"`
	AcceptAsyncLaunch bool         `json:"acceptAsyncLaunch"`
	Config            *AgentConfig `json:"config,omitempty"`
}

// Client is an HTTP client for Scion Hub with optional GCP metadata OIDC IAP token caching.
type Client struct {
	baseURL      string
	token        string
	iapAudience  string
	httpClient   *http.Client
	pollInterval time.Duration
	startTimeout time.Duration

	mu        sync.Mutex
	oidcToken string
	oidcExp   time.Time
}

// NewClient constructs a Scion Hub REST client.
func NewClient(cfg ClientConfig) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	startTimeout := cfg.StartTimeout
	if startTimeout <= 0 {
		startTimeout = 25 * time.Second
	}
	return &Client{
		baseURL:      strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		token:        strings.TrimSpace(cfg.Token),
		iapAudience:  strings.TrimSpace(cfg.IapAudience),
		httpClient:   httpClient,
		pollInterval: pollInterval,
		startTimeout: startTimeout,
	}
}

// ListAgents returns agents from Scion Hub, optionally filtered by projectId (slug or UUID).
func (c *Client) ListAgents(ctx context.Context, projectSlug string) ([]Agent, error) {
	path := "/api/v1/agents"
	if slug := strings.TrimSpace(projectSlug); slug != "" {
		path += "?projectId=" + url.QueryEscape(slug)
	}
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	var list []Agent
	if err := json.Unmarshal(raw, &list); err == nil {
		for i := range list {
			normalizeAgent(&list[i])
		}
		return list, nil
	}
	var wrapped struct {
		Agents []Agent `json:"agents"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("unexpected list agents response: %w", err)
	}
	if wrapped.Agents == nil {
		return []Agent{}, nil
	}
	for i := range wrapped.Agents {
		normalizeAgent(&wrapped.Agents[i])
	}
	return wrapped.Agents, nil
}

// GetAgent fetches a single agent by ID or slug.
func (c *Client) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	var raw json.RawMessage
	path := "/api/v1/agents/" + url.PathEscape(strings.TrimSpace(agentID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return decodeAgent(raw)
}

// CreateAgent creates a new Scion agent, retrying with a unique name suffix on 409 Conflict.
func (c *Client) CreateAgent(ctx context.Context, req CreateAgentRequest) (*Agent, error) {
	req.AcceptAsyncLaunch = true
	baseName := strings.TrimSpace(req.Name)
	if baseName == "" {
		baseName = "agent"
		req.Name = baseName
	}

	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
			req.Name = baseName + "-" + suffix
		}
		var raw json.RawMessage
		err := c.doJSON(ctx, http.MethodPost, "/api/v1/agents", req, &raw)
		if err == nil {
			return decodeAgent(raw)
		}
		if !errors.Is(err, ErrConflict) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%w: exhausted unique agent name retries", ErrConflict)
}

// StartAgent calls POST /api/v1/agents/{id}/start.
func (c *Client) StartAgent(ctx context.Context, agentID string) (*Agent, error) {
	path := "/api/v1/agents/" + url.PathEscape(strings.TrimSpace(agentID)) + "/start"
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, path, map[string]any{}, &raw); err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return &Agent{ID: agentID, Phase: "starting"}, nil
	}
	return decodeAgent(raw)
}

// StopAgent calls POST /api/v1/agents/{id}/stop.
func (c *Client) StopAgent(ctx context.Context, agentID string) error {
	path := "/api/v1/agents/" + url.PathEscape(strings.TrimSpace(agentID)) + "/stop"
	return c.doJSON(ctx, http.MethodPost, path, map[string]any{}, nil)
}

// SendMessage posts a structured instruction message to an agent.
func (c *Client) SendMessage(ctx context.Context, agentID, messageText string, wake bool) error {
	path := "/api/v1/agents/" + url.PathEscape(strings.TrimSpace(agentID)) + "/message"
	payload := map[string]any{
		"structured_message": map[string]string{
			"type": "instruction",
			"msg":  messageText,
		},
		"wake": wake,
	}
	return c.doJSON(ctx, http.MethodPost, path, payload, nil)
}

// EnsureRunningAndSendMessage inspects the agent's phase and starts it if stopped/halted/error/created
// before sending the instruction message with wake=true.
func (c *Client) EnsureRunningAndSendMessage(ctx context.Context, agentID, messageText string) error {
	agent, err := c.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}

	phase := strings.ToLower(strings.TrimSpace(agent.Phase))
	switch phase {
	case "running", "suspended":
		// Ready for wake=true message directly.
	case "starting", "provisioning":
		if err := c.waitForAgentRunning(ctx, agentID); err != nil {
			return err
		}
	default:
		if _, err := c.StartAgent(ctx, agentID); err != nil {
			return fmt.Errorf("failed to start agent %s: %w", agentID, err)
		}
		if err := c.waitForAgentRunning(ctx, agentID); err != nil {
			return err
		}
	}

	return c.SendMessage(ctx, agentID, messageText, true)
}

func (c *Client) waitForAgentRunning(ctx context.Context, agentID string) error {
	deadline := time.Now().Add(c.startTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.pollInterval):
		}
		curr, err := c.GetAgent(ctx, agentID)
		if err != nil {
			return err
		}
		currPhase := strings.ToLower(strings.TrimSpace(curr.Phase))
		if currPhase == "running" || currPhase == "suspended" {
			return nil
		}
		if currPhase == "error" || currPhase == "failed" {
			return fmt.Errorf("agent %s entered %s phase while starting", agentID, currPhase)
		}
	}
	return nil
}

func decodeAgent(raw json.RawMessage) (*Agent, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return &Agent{}, nil
	}
	var direct Agent
	if err := json.Unmarshal(trimmed, &direct); err == nil && (direct.ID != "" || direct.Name != "" || direct.Phase != "") {
		normalizeAgent(&direct)
		return &direct, nil
	}
	var wrapped struct {
		Agent *Agent `json:"agent"`
	}
	if err := json.Unmarshal(trimmed, &wrapped); err == nil && wrapped.Agent != nil {
		normalizeAgent(wrapped.Agent)
		return wrapped.Agent, nil
	}
	return nil, fmt.Errorf("unexpected agent payload: %s", string(trimmed))
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, out any) error {
	if c.baseURL == "" {
		return errors.New("scion hub url is not configured")
	}

	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("X-Scion-Authorization", "Bearer "+c.token)
	}
	if c.iapAudience != "" {
		oidc, err := c.getOIDCToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch IAP OIDC token: %w", err)
		}
		req.Header.Set("Proxy-Authorization", "Bearer "+oidc)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrAgentNotFound, strings.TrimSpace(string(respBytes)))
	}
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("%w: %s", ErrConflict, strings.TrimSpace(string(respBytes)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("scion hub %s %s returned HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	if out != nil && len(bytes.TrimSpace(respBytes)) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("failed to decode scion hub response: %w", err)
		}
	}
	return nil
}

func (c *Client) getOIDCToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.oidcToken != "" && time.Now().Add(2*time.Minute).Before(c.oidcExp) {
		tok := c.oidcToken
		c.mu.Unlock()
		return tok, nil
	}
	c.mu.Unlock()

	host := strings.TrimSpace(os.Getenv("GCE_METADATA_HOST"))
	if host == "" {
		host = "metadata.google.internal"
	}
	metaURL := fmt.Sprintf(
		"http://%s/computeMetadata/v1/instance/service-accounts/default/identity?audience=%s&format=full",
		host,
		url.QueryEscape(c.iapAudience),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metaURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rawTok, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(rawTok)))
	}

	tok := strings.TrimSpace(string(rawTok))
	exp := parseJWTExp(tok)
	if exp.IsZero() {
		exp = time.Now().Add(45 * time.Minute)
	}

	c.mu.Lock()
	c.oidcToken = tok
	c.oidcExp = exp
	c.mu.Unlock()

	return tok, nil
}

func parseJWTExp(jwtToken string) time.Time {
	parts := strings.Split(jwtToken, ".")
	if len(parts) < 2 {
		return time.Time{}
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil || claims.Exp <= 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}
