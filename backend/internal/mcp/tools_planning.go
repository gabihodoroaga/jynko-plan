package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

func registerPlanningTools(
	s *server.MCPServer,
	r *Resolver,
	cycleSvc *service.CycleService,
	epicSvc *service.EpicService,
	moduleSvc *service.ModuleService,
	stickySvc *service.StickyService,
) {
	// 1. list_cycles
	s.AddTool(
		mcp.NewTool("list_cycles",
			mcp.WithDescription("List sprint cycles in a project (active, planned, completed)."),
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

			cycles, err := cycleSvc.List(ctx, projID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list cycles: %v", err)), nil
			}
			return jsonResult(cycles)
		},
	)

	// 2. list_epics
	s.AddTool(
		mcp.NewTool("list_epics",
			mcp.WithDescription("List strategic epics and their status in a project."),
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

			epics, err := epicSvc.List(ctx, projID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list epics: %v", err)), nil
			}
			return jsonResult(epics)
		},
	)

	// 3. list_modules
	s.AddTool(
		mcp.NewTool("list_modules",
			mcp.WithDescription("List architectural feature modules in a project."),
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

			modules, err := moduleSvc.List(ctx, projID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list modules: %v", err)), nil
			}
			return jsonResult(modules)
		},
	)

	// 4. list_stickies
	s.AddTool(
		mcp.NewTool("list_stickies",
			mcp.WithDescription("List brainstorming sticky notes from the project stickies board."),
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

			stickies, err := stickySvc.ListStickies(ctx, projID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list stickies: %v", err)), nil
			}
			return jsonResult(stickies)
		},
	)

	// 5. create_sticky
	s.AddTool(
		mcp.NewTool("create_sticky",
			mcp.WithDescription("Create a quick sticky note on the brainstorming board."),
			mcp.WithString("content", mcp.Required(), mcp.Description("The content / thought on the note")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("color", mcp.Description("Color palette: BANANA, FLAMINGO, MINT, LAGOON, LAVENDER, PEACH")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			content, err := req.RequireString("content")
			if err != nil {
				return mcp.NewToolResultError("content is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleMember); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			input := service.CreateStickyInput{
				Content: content,
				Color:   req.GetString("color", ""),
			}

			created, err := stickySvc.CreateSticky(ctx, projID, r.UserFromContext(ctx), input)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create sticky note: %v", err)), nil
			}
			return jsonResult(created)
		},
	)
}
