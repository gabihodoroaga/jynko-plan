package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// AttachmentDownloadResult contains metadata and data for a downloaded attachment.
type AttachmentDownloadResult struct {
	Attachment  *model.Attachment `json:"attachment"`
	Data        string            `json:"data"`
	Text        string            `json:"text,omitempty"`
	FileName    string            `json:"file_name"`
	ContentType string            `json:"content_type"`
	SizeBytes   int64             `json:"size_bytes"`
}

func registerAttachmentTools(
	s *server.MCPServer,
	r *Resolver,
	attSvc *service.AttachmentService,
	itemSvc *service.ItemService,
) {
	// 1. list_attachments
	s.AddTool(
		mcp.NewTool("list_attachments",
			mcp.WithDescription("List attachments in a project, optionally filtered by work item."),
			mcp.WithString("project_id", mcp.Description("Target project ID (e.g. 'proj-jynko-core'). Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("item_id", mcp.Description("Work item ID or human key (e.g. 'PLAN-8') to filter attachments by")),
			mcp.WithString("item_key_or_id", mcp.Description("Alias for item_id")),
			mcp.WithString("work_item_id", mcp.Description("Alias for item_id")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projID, err := r.ResolveProjectID(req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if attSvc == nil {
				return mcp.NewToolResultError("attachment service is not available"), nil
			}

			rawItemID := req.GetString("item_id", "")
			if rawItemID == "" {
				rawItemID = req.GetString("item_key_or_id", "")
			}
			if rawItemID == "" {
				rawItemID = req.GetString("work_item_id", "")
			}

			var itemID string
			if rawItemID != "" {
				if itemSvc == nil {
					return mcp.NewToolResultError("item service is not available to resolve item_id"), nil
				}
				targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, rawItemID)
				if err != nil || targetItem == nil {
					return mcp.NewToolResultError(fmt.Sprintf("work item %q not found in project %q", rawItemID, projID)), nil
				}
				itemID = targetItem.ID
			}

			attachments, err := attSvc.ListAttachments(ctx, projID, itemID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list attachments: %v", err)), nil
			}
			if attachments == nil {
				attachments = []model.Attachment{}
			}
			return jsonResult(attachments)
		},
	)

	// Shared handler for get_attachment and download_attachment
	downloadHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projID, err := r.ResolveProjectID(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := r.CheckRole(ctx, projID, model.RoleViewer); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if attSvc == nil {
			return mcp.NewToolResultError("attachment service is not available"), nil
		}

		attID := req.GetString("attachment_id", "")
		if attID == "" {
			attID = req.GetString("id", "")
		}

		if attID == "" {
			rawItemID := req.GetString("item_id", "")
			if rawItemID == "" {
				rawItemID = req.GetString("item_key_or_id", "")
			}
			if rawItemID == "" {
				rawItemID = req.GetString("work_item_id", "")
			}
			if rawItemID != "" && itemSvc != nil {
				targetItem, err := itemSvc.GetByIDOrKey(ctx, projID, rawItemID)
				if err != nil || targetItem == nil {
					return mcp.NewToolResultError(fmt.Sprintf("work item %q not found in project %q", rawItemID, projID)), nil
				}
				items, err := attSvc.ListAttachments(ctx, projID, targetItem.ID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list attachments for item: %v", err)), nil
				}
				if len(items) == 0 {
					return mcp.NewToolResultError(fmt.Sprintf("no attachments found for work item %q", rawItemID)), nil
				}
				if len(items) == 1 {
					attID = items[0].ID
				} else {
					var ids []string
					for _, a := range items {
						ids = append(ids, fmt.Sprintf("%s (%s)", a.ID, a.FileName))
					}
					return mcp.NewToolResultError(fmt.Sprintf("multiple attachments found for %s, please specify attachment_id: %s", rawItemID, strings.Join(ids, ", "))), nil
				}
			}
		}

		if attID == "" {
			return mcp.NewToolResultError("attachment_id is required"), nil
		}

		att, rc, err := attSvc.GetAttachment(ctx, projID, attID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return mcp.NewToolResultError(fmt.Sprintf("attachment %q not found in project %q", attID, projID)), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("failed to get attachment: %v", err)), nil
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read attachment content: %v", err)), nil
		}

		base64Data := base64.StdEncoding.EncodeToString(data)
		resView := AttachmentDownloadResult{
			Attachment:  att,
			Data:        base64Data,
			FileName:    att.FileName,
			ContentType: att.ContentType,
			SizeBytes:   att.SizeBytes,
		}

		if isTextMIMEType(att.ContentType) && utf8.Valid(data) {
			resView.Text = string(data)
		}

		jsonBytes, err := json.MarshalIndent(resView, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize attachment result: %v", err)), nil
		}

		contents := []mcp.Content{
			mcp.TextContent{
				Type: mcp.ContentTypeText,
				Text: string(jsonBytes),
			},
		}

		if isImageMIMEType(att.ContentType) {
			contents = append(contents, mcp.ImageContent{
				Type:     mcp.ContentTypeImage,
				Data:     base64Data,
				MIMEType: att.ContentType,
			})
		}

		return &mcp.CallToolResult{
			Content: contents,
		}, nil
	}

	// 2. get_attachment
	s.AddTool(
		mcp.NewTool("get_attachment",
			mcp.WithDescription("Get an attachment's metadata and content/data by ID."),
			mcp.WithString("attachment_id", mcp.Required(), mcp.Description("ID of the attachment to retrieve")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("id", mcp.Description("Alias for attachment_id")),
			mcp.WithString("item_id", mcp.Description("Optional work item ID or key fallback if only one attachment exists")),
		),
		downloadHandler,
	)

	// 3. download_attachment
	s.AddTool(
		mcp.NewTool("download_attachment",
			mcp.WithDescription("Download an attachment's content (base64 data and text if applicable) and metadata by attachment ID."),
			mcp.WithString("attachment_id", mcp.Required(), mcp.Description("ID of the attachment to download")),
			mcp.WithString("project_id", mcp.Description("Target project ID. Uses DEFAULT_PROJECT_ID if omitted.")),
			mcp.WithString("id", mcp.Description("Alias for attachment_id")),
			mcp.WithString("item_id", mcp.Description("Optional work item ID or key fallback if only one attachment exists")),
		),
		downloadHandler,
	)
}

func isImageMIMEType(contentType string) bool {
	mt := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(mt, ";"); idx != -1 {
		mt = strings.TrimSpace(mt[:idx])
	}
	return strings.HasPrefix(mt, "image/")
}

func isTextMIMEType(contentType string) bool {
	mt := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(mt, ";"); idx != -1 {
		mt = strings.TrimSpace(mt[:idx])
	}
	if strings.HasPrefix(mt, "text/") {
		return true
	}
	switch mt {
	case "application/json", "application/xml", "application/x-yaml", "application/yaml",
		"application/javascript", "application/sql", "application/csv":
		return true
	}
	return false
}
