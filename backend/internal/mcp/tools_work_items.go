package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// WorkItemDetailView combines a work item with its child sub-tasks and discussion comments.
type WorkItemDetailView struct {
	Item     *model.WorkItem     `json:"item"`
	SubItems []model.SubWorkItem `json:"sub_items"`
	Comments []model.Comment     `json:"comments"`
}

func registerWorkItemTools(s *server.MCPServer, r *Resolver, itemSvc *service.ItemService) {
	// 1. list_work_items
	s.AddTool(
		mcp.NewTool("list_work_items",
			mcp.WithDescription("List work items in a project with optional filters for status, priority, sprint cycle, epic, assignee, or text search."),
			mcp.WithString("project_id", mcp.Description("Target project ID (e.g. 'proj-jynko-core'). Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("status", mcp.Description("Filter by status: BACKLOG, TODO, IN_PROGRESS, IN_REVIEW, DONE, CANCELLED")),
			mcp.WithString("priority", mcp.Description("Filter by priority: NONE, LOW, MEDIUM, HIGH, URGENT")),
			mcp.WithString("cycle_id", mcp.Description("Filter by sprint cycle ID")),
			mcp.WithString("epic_id", mcp.Description("Filter by strategic epic ID")),
			mcp.WithString("module_id", mcp.Description("Filter by feature module ID")),
			mcp.WithString("assignee_id", mcp.Description("Filter by assignee user ID")),
			mcp.WithString("search", mcp.Description("Text search across title and description")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			filter := model.WorkItemFilter{
				Status:     req.GetString("status", ""),
				Priority:   req.GetString("priority", ""),
				CycleID:    req.GetString("cycle_id", ""),
				EpicID:     req.GetString("epic_id", ""),
				ModuleID:   req.GetString("module_id", ""),
				AssigneeID: req.GetString("assignee_id", ""),
				Search:     req.GetString("search", ""),
			}

			items, err := itemSvc.List(ctx, projID, filter)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list work items: %v", err)), nil
			}
			return jsonResult(items)
		},
	)

	// 2. get_work_item
	s.AddTool(
		mcp.NewTool("get_work_item",
			mcp.WithDescription("Get a full work item by human key (e.g. 'HODO-1') or internal ID, including its sub-tasks and discussion comments."),
			mcp.WithString("item_key_or_id", mcp.Required(), mcp.Description("Human sequential key (e.g. 'HODO-1') or internal Firestore item ID")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			keyOrID, err := req.RequireString("item_key_or_id")
			if err != nil {
				return mcp.NewToolResultError("item_key_or_id is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, keyOrID)
			if err != nil || targetItem == nil {
				return mcp.NewToolResultError(fmt.Sprintf("work item %q not found in project %q", keyOrID, projID)), nil
			}

			subItems, _ := itemSvc.ListSubItems(ctx, projID, targetItem.ID)
			comments, _ := itemSvc.ListComments(ctx, projID, targetItem.ID)

			view := WorkItemDetailView{
				Item:     targetItem,
				SubItems: subItems,
				Comments: comments,
			}
			return jsonResult(view)
		},
	)

	// 3. create_work_item
	s.AddTool(
		mcp.NewTool("create_work_item",
			mcp.WithDescription("Create a new work item with atomic sequential key generation (<KEY_PREFIX>-<SEQ>)."),
			mcp.WithString("title", mcp.Required(), mcp.Description("Title / summary of the work item")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("description", mcp.Description("Markdown description of the work item")),
			mcp.WithString("status", mcp.Description("BACKLOG, TODO, IN_PROGRESS, IN_REVIEW, DONE, CANCELLED (defaults to BACKLOG)")),
			mcp.WithString("priority", mcp.Description("NONE, LOW, MEDIUM, HIGH, URGENT (defaults to NONE)")),
			mcp.WithString("assignee_id", mcp.Description("User ID of the assigned team member")),
			mcp.WithString("cycle_id", mcp.Description("Sprint cycle ID")),
			mcp.WithString("epic_id", mcp.Description("Strategic epic ID")),
			mcp.WithString("module_id", mcp.Description("Feature module ID")),
			mcp.WithInteger("estimate", mcp.Description("Story points estimate (e.g. 1, 2, 3, 5, 8)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			title, err := req.RequireString("title")
			if err != nil {
				return mcp.NewToolResultError("title is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleMember); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			input := service.CreateWorkItemInput{
				Title:       title,
				Description: req.GetString("description", ""),
				Status:      req.GetString("status", "BACKLOG"),
				Priority:    req.GetString("priority", "NONE"),
				AssigneeID:  req.GetString("assignee_id", ""),
				CycleID:     req.GetString("cycle_id", ""),
				EpicID:      req.GetString("epic_id", ""),
				ModuleID:    req.GetString("module_id", ""),
			}

			if est := req.GetInt("estimate", 0); est > 0 {
				input.Estimate = &est
			}

			created, err := itemSvc.Create(ctx, projID, r.UserFromContext(ctx), input)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create work item: %v", err)), nil
			}
			return jsonResult(created)
		},
	)

	// 4. update_work_item
	s.AddTool(
		mcp.NewTool("update_work_item",
			mcp.WithDescription("Update fields of an existing work item."),
			mcp.WithString("item_id", mcp.Required(), mcp.Description("Internal ID of the work item (or key e.g. 'HODO-1')")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("title", mcp.Description("New title")),
			mcp.WithString("description", mcp.Description("New Markdown description")),
			mcp.WithString("status", mcp.Description("New status: BACKLOG, TODO, IN_PROGRESS, IN_REVIEW, DONE, CANCELLED")),
			mcp.WithString("priority", mcp.Description("New priority: NONE, LOW, MEDIUM, HIGH, URGENT")),
			mcp.WithString("assignee_id", mcp.Description("New assignee user ID (or empty string to unassign)")),
			mcp.WithString("cycle_id", mcp.Description("New sprint cycle ID")),
			mcp.WithString("epic_id", mcp.Description("New epic ID")),
			mcp.WithString("module_id", mcp.Description("New module ID")),
			mcp.WithInteger("estimate", mcp.Description("New story points estimate")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rawID, err := req.RequireString("item_id")
			if err != nil {
				return mcp.NewToolResultError("item_id is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleMember); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, rawID)
			if err != nil || targetItem == nil {
				return mcp.NewToolResultError(fmt.Sprintf("work item %q not found in project %q", rawID, projID)), nil
			}

			input := service.UpdateWorkItemInput{}
			args := req.GetArguments()
			if v, ok := args["title"].(string); ok {
				input.Title = &v
			}
			if v, ok := args["description"].(string); ok {
				input.Description = &v
			}
			if v, ok := args["status"].(string); ok {
				input.Status = &v
			}
			if v, ok := args["priority"].(string); ok {
				input.Priority = &v
			}
			if v, ok := args["assignee_id"].(string); ok {
				input.AssigneeID = &v
			}
			if v, ok := args["cycle_id"].(string); ok {
				input.CycleID = &v
			}
			if v, ok := args["epic_id"].(string); ok {
				input.EpicID = &v
			}
			if v, ok := args["module_id"].(string); ok {
				input.ModuleID = &v
			}
			if rawEst, exists := args["estimate"]; exists {
				if rawEst == nil {
					input.Estimate = service.OptionalInt{Set: true, Value: nil}
				} else if est := req.GetInt("estimate", -1); est >= 0 {
					if est == 0 {
						input.Estimate = service.OptionalInt{Set: true, Value: nil}
					} else {
						input.Estimate = service.OptionalInt{Set: true, Value: &est}
					}
				}
			}

			updated, err := itemSvc.Update(ctx, projID, targetItem.ID, input)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to update work item: %v", err)), nil
			}
			return jsonResult(updated)
		},
	)

	// 5. add_comment
	s.AddTool(
		mcp.NewTool("add_comment",
			mcp.WithDescription("Post a Markdown discussion comment to a work item."),
			mcp.WithString("item_id", mcp.Required(), mcp.Description("Work item ID or sequential key")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Markdown comment text")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rawID, err := req.RequireString("item_id")
			if err != nil {
				return mcp.NewToolResultError("item_id is required"), nil
			}
			body, err := req.RequireString("body")
			if err != nil {
				return mcp.NewToolResultError("body is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleMember); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, rawID)
			if err != nil || targetItem == nil {
				return mcp.NewToolResultError(fmt.Sprintf("work item %q not found in project %q", rawID, projID)), nil
			}

			comment, err := itemSvc.CreateComment(ctx, projID, targetItem.ID, r.UserFromContext(ctx), service.CreateCommentInput{Body: body})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to post comment: %v", err)), nil
			}
			return jsonResult(comment)
		},
	)

	// 6. add_sub_item
	s.AddTool(
		mcp.NewTool("add_sub_item",
			mcp.WithDescription("Add a granular checklist sub-item under a work item."),
			mcp.WithString("item_id", mcp.Required(), mcp.Description("Work item ID or sequential key")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Checklist sub-item title")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rawID, err := req.RequireString("item_id")
			if err != nil {
				return mcp.NewToolResultError("item_id is required"), nil
			}
			title, err := req.RequireString("title")
			if err != nil {
				return mcp.NewToolResultError("title is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleMember); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, rawID)
			if err != nil || targetItem == nil {
				return mcp.NewToolResultError(fmt.Sprintf("work item %q not found in project %q", rawID, projID)), nil
			}

			sub, err := itemSvc.CreateSubItem(ctx, projID, targetItem.ID, service.CreateSubItemInput{Title: title})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to add sub-item: %v", err)), nil
			}
			return jsonResult(sub)
		},
	)

	// 7. toggle_sub_item
	s.AddTool(
		mcp.NewTool("toggle_sub_item",
			mcp.WithDescription("Set the completion state of a work item's checklist sub-item."),
			mcp.WithString("item_id", mcp.Required(), mcp.Description("Parent work item ID or sequential key")),
			mcp.WithString("sub_item_id", mcp.Required(), mcp.Description("Sub-item ID")),
			mcp.WithBoolean("is_completed", mcp.Required(), mcp.Description("True to mark done, false for incomplete")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rawID, err := req.RequireString("item_id")
			if err != nil {
				return mcp.NewToolResultError("item_id is required"), nil
			}
			subID, err := req.RequireString("sub_item_id")
			if err != nil {
				return mcp.NewToolResultError("sub_item_id is required"), nil
			}
			isCompleted, err := req.RequireBool("is_completed")
			if err != nil {
				return mcp.NewToolResultError("is_completed is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleMember); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, rawID)
			if err != nil || targetItem == nil {
				return mcp.NewToolResultError(fmt.Sprintf("work item %q not found in project %q", rawID, projID)), nil
			}

			updated, err := itemSvc.UpdateSubItem(ctx, projID, targetItem.ID, subID, service.UpdateSubItemInput{IsCompleted: &isCompleted})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to update sub-item: %v", err)), nil
			}
			return jsonResult(updated)
		},
	)
}
