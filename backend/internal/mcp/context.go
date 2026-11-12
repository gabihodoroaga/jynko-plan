package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// Config holds runtime configuration and active user identity for the MCP server.
type Config struct {
	DefaultProjectID string
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
	user := r.cfg.User
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
