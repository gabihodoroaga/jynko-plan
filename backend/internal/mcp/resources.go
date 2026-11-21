package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

func registerResources(
	s *server.MCPServer,
	r *Resolver,
	itemSvc *service.ItemService,
	pageSvc *service.PageService,
	attSvc *service.AttachmentService,
) {
	// 1. Work Item Resource Template
	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"jynko://projects/{projectId}/items/{itemKey}",
			"Jynko Work Item View",
			mcp.WithTemplateDescription("Live Markdown view of a Jynko work item including properties, sub-tasks, and comments."),
			mcp.WithTemplateMIMEType("text/markdown"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			uri := req.Params.URI
			if !strings.HasPrefix(uri, "jynko://") {
				return nil, fmt.Errorf("invalid work item URI: %s", uri)
			}
			path := strings.TrimPrefix(uri, "jynko://")
			parts := strings.Split(path, "/")
			// Expected strictly: projects/{projectId}/items/{itemKey}
			if len(parts) != 4 || parts[0] != "projects" || parts[1] == "" || parts[2] != "items" || parts[3] == "" {
				return nil, fmt.Errorf("invalid work item URI: %s", uri)
			}
			projID := parts[1]
			keyOrID := parts[3]

			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return nil, err
			}

			targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, keyOrID)
			if err != nil || targetItem == nil {
				return nil, fmt.Errorf("work item %s not found in project %s", keyOrID, projID)
			}

			subItems, _ := itemSvc.ListSubItems(ctx, projID, targetItem.ID)
			comments, _ := itemSvc.ListComments(ctx, projID, targetItem.ID)

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("# [%s] %s\n\n", targetItem.Key, targetItem.Title))
			sb.WriteString(fmt.Sprintf("- **Status:** %s\n", targetItem.Status))
			sb.WriteString(fmt.Sprintf("- **Priority:** %s\n", targetItem.Priority))
			if targetItem.AssigneeID != "" {
				sb.WriteString(fmt.Sprintf("- **Assignee:** %s\n", targetItem.AssigneeID))
			}
			if targetItem.Estimate != nil {
				sb.WriteString(fmt.Sprintf("- **Estimate:** %d pts\n", *targetItem.Estimate))
			}
			sb.WriteString("\n## Description\n\n")
			if targetItem.Description != "" {
				sb.WriteString(targetItem.Description)
				sb.WriteString("\n\n")
			} else {
				sb.WriteString("_No description provided._\n\n")
			}

			if len(subItems) > 0 {
				sb.WriteString("## Sub-Work Items\n\n")
				for _, sub := range subItems {
					check := "[ ]"
					if sub.IsCompleted {
						check = "[x]"
					}
					sb.WriteString(fmt.Sprintf("- %s %s\n", check, sub.Title))
				}
				sb.WriteString("\n")
			}

			if len(comments) > 0 {
				sb.WriteString("## Discussion Comments\n\n")
				for _, c := range comments {
					sb.WriteString(fmt.Sprintf("### %s (%s)\n\n%s\n\n", c.AuthorName, c.CreatedAt.Format("2006-01-02 15:04"), c.Body))
				}
			}

			if attSvc != nil {
				attachments, _ := attSvc.ListAttachments(ctx, projID, targetItem.ID)
				if len(attachments) > 0 {
					sb.WriteString("## Attachments\n\n")
					for _, att := range attachments {
						sb.WriteString(fmt.Sprintf("- [%s](jynko://projects/%s/attachments/%s) (%s, %d bytes)\n", att.FileName, projID, att.ID, att.ContentType, att.SizeBytes))
					}
					sb.WriteString("\n")
				}
			}

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     sb.String(),
				},
			}, nil
		},
	)

	// 2. Wiki Page Resource Template
	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"jynko://projects/{projectId}/pages/{pageId}",
			"Jynko Wiki Page Document",
			mcp.WithTemplateDescription("Live Markdown document from Jynko knowledge base."),
			mcp.WithTemplateMIMEType("text/markdown"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			uri := req.Params.URI
			if !strings.HasPrefix(uri, "jynko://") {
				return nil, fmt.Errorf("invalid wiki page URI: %s", uri)
			}
			path := strings.TrimPrefix(uri, "jynko://")
			parts := strings.Split(path, "/")
			// Expected strictly: projects/{projectId}/pages/{pageId}
			if len(parts) != 4 || parts[0] != "projects" || parts[1] == "" || parts[2] != "pages" || parts[3] == "" {
				return nil, fmt.Errorf("invalid wiki page URI: %s", uri)
			}
			projID := parts[1]
			pageID := parts[3]

			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return nil, err
			}

			page, err := pageSvc.GetPage(ctx, projID, pageID)
			if err != nil {
				return nil, fmt.Errorf("failed to get wiki page: %w", err)
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("# %s\n\n", page.Title))
			if page.IsLocked {
				sb.WriteString("> 🔒 _This wiki page is locked from edits._\n\n")
			}
			sb.WriteString(page.Content)
			sb.WriteString("\n")

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     sb.String(),
				},
			}, nil
		},
	)

	// 3. Attachment Resource Template
	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"jynko://projects/{projectId}/attachments/{attachmentId}",
			"Jynko Attachment Content",
			mcp.WithTemplateDescription("Live content and data of a project attachment file."),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			uri := req.Params.URI
			if !strings.HasPrefix(uri, "jynko://") {
				return nil, fmt.Errorf("invalid attachment URI: %s", uri)
			}
			path := strings.TrimPrefix(uri, "jynko://")
			parts := strings.Split(path, "/")
			// Expected strictly: projects/{projectId}/attachments/{attachmentId}
			if len(parts) != 4 || parts[0] != "projects" || parts[1] == "" || parts[2] != "attachments" || parts[3] == "" {
				return nil, fmt.Errorf("invalid attachment URI: %s", uri)
			}
			projID := parts[1]
			attID := parts[3]

			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return nil, err
			}

			if attSvc == nil {
				return nil, fmt.Errorf("attachment service is not available")
			}

			att, rc, err := attSvc.GetAttachment(ctx, projID, attID)
			if err != nil {
				return nil, fmt.Errorf("failed to get attachment: %w", err)
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read attachment content: %w", err)
			}

			if isTextMIMEType(att.ContentType) && utf8.Valid(data) {
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      uri,
						MIMEType: att.ContentType,
						Text:     string(data),
					},
				}, nil
			}

			return []mcp.ResourceContents{
				mcp.BlobResourceContents{
					URI:      uri,
					MIMEType: att.ContentType,
					Blob:     base64.StdEncoding.EncodeToString(data),
				},
			}, nil
		},
	)
}
