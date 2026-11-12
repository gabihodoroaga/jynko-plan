package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

func registerPrompts(
	s *server.MCPServer,
	r *Resolver,
	itemSvc *service.ItemService,
	cycleSvc *service.CycleService,
) {
	// 1. sprint-standup prompt
	s.AddPrompt(
		mcp.NewPrompt(
			"sprint-standup",
			mcp.WithPromptDescription("Generate a structured agile standup update based on active sprint items in Jynko."),
			mcp.WithArgument("project_id", mcp.ArgumentDescription("Target project ID")),
		),
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			projID := req.Params.Arguments["project_id"]
			if projID == "" {
				projID = r.cfg.DefaultProjectID
			}
			if projID == "" {
				return nil, fmt.Errorf("project_id is required")
			}

			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return nil, err
			}

			// Find active cycle
			cycles, _ := cycleSvc.List(ctx, projID)
			var activeCycle *model.Cycle
			for i := range cycles {
				if cycles[i].IsActive {
					activeCycle = &cycles[i]
					break
				}
			}

			var cycleInfo string
			var cycleID string
			if activeCycle != nil {
				cycleInfo = fmt.Sprintf("Active Sprint: %s (%s to %s)", activeCycle.Name, activeCycle.StartDate.Format("2006-01-02"), activeCycle.EndDate.Format("2006-01-02"))
				cycleID = activeCycle.ID
			} else {
				cycleInfo = "No active sprint cycle marked. Listing general project work items."
			}

			items, _ := itemSvc.List(ctx, projID, model.WorkItemFilter{CycleID: cycleID})

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Project: %s\n%s\n\nWork Items in Sprint:\n", projID, cycleInfo))
			for _, it := range items {
				assignee := it.AssigneeID
				if assignee == "" {
					assignee = "Unassigned"
				}
				sb.WriteString(fmt.Sprintf("- [%s] %s (Status: %s, Priority: %s, Assignee: %s)\n", it.Key, it.Title, it.Status, it.Priority, assignee))
			}

			instruction := fmt.Sprintf(`Please format an executive daily agile standup update based on the following Jynko data:

%s

Structure the response into:
1. 🚀 What was Completed recently (DONE)
2. 🔄 In Progress & Active Work (IN_PROGRESS, IN_REVIEW)
3. 📋 Planned Next (TODO)
4. ⚠️ Blockers & Risks (if any items are urgent or delayed)`, sb.String())

			return mcp.NewGetPromptResult(
				"Jynko Daily Sprint Standup",
				[]mcp.PromptMessage{
					mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(instruction)),
				},
			), nil
		},
	)

	// 2. feature-breakdown prompt
	s.AddPrompt(
		mcp.NewPrompt(
			"feature-breakdown",
			mcp.WithPromptDescription("Analyze a feature description and produce a structured breakdown ready for creation in Jynko."),
			mcp.WithArgument("feature_description", mcp.RequiredArgument(), mcp.ArgumentDescription("Description, PRD, or requirement summary of the feature")),
			mcp.WithArgument("project_id", mcp.ArgumentDescription("Target project ID")),
		),
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			feat := req.Params.Arguments["feature_description"]
			projID := req.Params.Arguments["project_id"]
			if projID == "" {
				projID = r.cfg.DefaultProjectID
			}

			promptText := fmt.Sprintf(`You are the Jynko project planning assistant. Break down the following feature specification into structured Jynko entities:

Target Project: %s
Feature Requirement:
%s

Please output:
1. Proposed **Epic** name and strategic objective.
2. Architectural **Module** grouping.
3. List of individual **Work Items** with:
   - Proposed Title
   - Priority (LOW, MEDIUM, HIGH, URGENT)
   - Initial Status (BACKLOG or TODO)
   - Story point estimate (1, 2, 3, 5, 8)
   - Detailed Markdown description
   - Checklist of granular sub-tasks
4. Instructions on how to call the Jynko MCP tools ('create_work_item', 'add_sub_item') to persist this plan.`, projID, feat)

			return mcp.NewGetPromptResult(
				"Jynko Feature Breakdown & Planning",
				[]mcp.PromptMessage{
					mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(promptText)),
				},
			), nil
		},
	)
}
