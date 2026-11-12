package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

func registerWikiTools(s *server.MCPServer, r *Resolver, pageSvc *service.PageService) {
	// 1. list_wiki_pages
	s.AddTool(
		mcp.NewTool("list_wiki_pages",
			mcp.WithDescription("List knowledge base wiki pages in a project, optionally filtered by collection."),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("collection_id", mcp.Description("Optional collection ID to filter pages")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			collectionID := req.GetString("collection_id", "")
			pages, err := pageSvc.ListPages(ctx, projID, collectionID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list wiki pages: %v", err)), nil
			}
			return jsonResult(pages)
		},
	)

	// 2. get_wiki_page
	s.AddTool(
		mcp.NewTool("get_wiki_page",
			mcp.WithDescription("Get the full Markdown content, title, and metadata of a knowledge base wiki page."),
			mcp.WithString("page_id", mcp.Required(), mcp.Description("The ID of the wiki page")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			pageID, err := req.RequireString("page_id")
			if err != nil {
				return mcp.NewToolResultError("page_id is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			page, err := pageSvc.GetPage(ctx, projID, pageID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get wiki page: %v", err)), nil
			}
			return jsonResult(page)
		},
	)

	// 3. create_wiki_page
	s.AddTool(
		mcp.NewTool("create_wiki_page",
			mcp.WithDescription("Create a new wiki page with Markdown content in a project collection."),
			mcp.WithString("title", mcp.Required(), mcp.Description("Page title")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Markdown body of the document")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("collection_id", mcp.Description("Target collection ID (auto-selects first collection if omitted)")),
			mcp.WithString("parent_page_id", mcp.Description("Optional parent page ID for nested document trees")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			title, err := req.RequireString("title")
			if err != nil {
				return mcp.NewToolResultError("title is required"), nil
			}
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

			collectionID := req.GetString("collection_id", "")
			if collectionID == "" {
				// Pick first collection in project
				cols, cErr := pageSvc.ListCollections(ctx, projID)
				if cErr == nil && len(cols) > 0 {
					collectionID = cols[0].ID
				}
			}

			input := service.CreatePageInput{
				Title:        title,
				Content:      content,
				CollectionID: collectionID,
				ParentPageID: req.GetString("parent_page_id", ""),
			}

			created, err := pageSvc.CreatePage(ctx, projID, r.cfg.User, input)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to create wiki page: %v", err)), nil
			}
			return jsonResult(created)
		},
	)

	// 4. update_wiki_page
	s.AddTool(
		mcp.NewTool("update_wiki_page",
			mcp.WithDescription("Update an existing wiki page (creates an automatic version history snapshot; fails if page is locked)."),
			mcp.WithString("page_id", mcp.Required(), mcp.Description("The ID of the wiki page to update")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("title", mcp.Description("Updated title")),
			mcp.WithString("content", mcp.Description("Updated Markdown body")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			pageID, err := req.RequireString("page_id")
			if err != nil {
				return mcp.NewToolResultError("page_id is required"), nil
			}
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleMember); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			input := service.UpdatePageInput{}
			args := req.GetArguments()
			if v, ok := args["title"].(string); ok {
				input.Title = &v
			}
			if v, ok := args["content"].(string); ok {
				input.Content = &v
			}

			updated, err := pageSvc.UpdatePage(ctx, projID, pageID, r.cfg.User, input)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to update wiki page: %v", err)), nil
			}
			return jsonResult(updated)
		},
	)

	// 5. list_collections
	s.AddTool(
		mcp.NewTool("list_collections",
			mcp.WithDescription("List knowledge base collections in a project."),
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

			cols, err := pageSvc.ListCollections(ctx, projID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list collections: %v", err)), nil
			}
			return jsonResult(cols)
		},
	)
}
