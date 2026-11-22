package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

type mcpUserCtxKey struct{}
type mcpBaseURLCtxKey struct{}

// WithUserContext attaches a per-request authenticated UserContext to ctx (used in HTTP/SSE mode).
func WithUserContext(ctx context.Context, user model.UserContext) context.Context {
	return context.WithValue(ctx, mcpUserCtxKey{}, user)
}

// WithBaseURL attaches the request's public base URL (scheme://host) to ctx.
func WithBaseURL(ctx context.Context, baseURL string) context.Context {
	return context.WithValue(ctx, mcpBaseURLCtxKey{}, strings.TrimRight(strings.TrimSpace(baseURL), "/"))
}

// Config holds runtime configuration and fallback user identity (for STDIO mode) for the MCP server.
type Config struct {
	DefaultProjectID string
	BaseURL          string
	User             model.UserContext
}

// Resolver validates permissions and parameter fallbacks for MCP tool invocations.
type Resolver struct {
	cfg         Config
	projectRepo repository.ProjectRepository
}

// NewResolver creates a new Resolver.
func NewResolver(cfg Config, projectRepo repository.ProjectRepository) *Resolver {
	return &Resolver{
		cfg:         cfg,
		projectRepo: projectRepo,
	}
}

// UserFromContext returns the per-request UserContext from ctx if present, falling back to Config.User.
func (r *Resolver) UserFromContext(ctx context.Context) model.UserContext {
	if ctx != nil {
		if u, ok := ctx.Value(mcpUserCtxKey{}).(model.UserContext); ok && u.UserID != "" {
			return u
		}
	}
	return r.cfg.User
}

// BaseURLFromContext returns the request's base URL from ctx if present, falling back to Config.BaseURL.
func (r *Resolver) BaseURLFromContext(ctx context.Context) string {
	if ctx != nil {
		if baseURL, ok := ctx.Value(mcpBaseURLCtxKey{}).(string); ok && baseURL != "" {
			return baseURL
		}
	}
	return strings.TrimRight(strings.TrimSpace(r.cfg.BaseURL), "/")
}

// ResolveProjectID extracts the project_id argument or falls back to Config.DefaultProjectID.
func (r *Resolver) ResolveProjectID(req mcp.CallToolRequest) (string, error) {
	projID := req.GetString("project_id", "")
	if projID == "" {
		projID = r.cfg.DefaultProjectID
	}
	if projID == "" {
		return "", errors.New("project_id is required (or set DEFAULT_PROJECT_ID environment variable)")
	}
	return projID, nil
}

// CheckRole verifies the authenticated user has at least minRole in the specified project.
func (r *Resolver) CheckRole(ctx context.Context, projectID string, minRole model.ProjectRole) error {
	user := r.UserFromContext(ctx)
	if user.UserID == "" {
		return errors.New("unauthorized: missing agent/user context")
	}

	member, err := r.projectRepo.GetMember(ctx, projectID, user.UserID)
	if err != nil || member == nil {
		member, err = r.projectRepo.GetMemberByEmail(ctx, projectID, user.Email)
		if err != nil || member == nil {
			return fmt.Errorf("access denied: user %s (%s) is not a member of project %s", user.UserID, user.Email, projectID)
		}
	}

	if !middleware.HasPermission(member.Role, minRole) {
		return fmt.Errorf("insufficient permissions: project role %s is below required role %s", member.Role, minRole)
	}
	return nil
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
