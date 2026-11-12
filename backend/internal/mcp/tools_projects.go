package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// ProjectDetailView represents a project enriched with member list.
type ProjectDetailView struct {
	Project *model.Project        `json:"project"`
	Members []model.ProjectMember `json:"members"`
}

func registerProjectTools(s *server.MCPServer, r *Resolver, projSvc *service.ProjectService) {
	// 1. list_projects
	s.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all projects accessible to the authenticated agent / user."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projects, err := projSvc.ListProjectsForUser(ctx, r.cfg.User)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list projects: %v", err)), nil
			}
			return jsonResult(projects)
		},
	)

	// 2. get_project
	s.AddTool(
		mcp.NewTool("get_project",
			mcp.WithDescription("Get project details, workflow settings, and member roster."),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			proj, err := projSvc.GetProject(ctx, projID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get project: %v", err)), nil
			}

			members, _ := projSvc.ListMembers(ctx, projID)

			view := ProjectDetailView{
				Project: proj,
				Members: members,
			}
			return jsonResult(view)
		},
	)
}
