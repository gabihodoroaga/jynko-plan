package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/idtoken"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
	"github.com/gabihodoroaga/jynko-plan/internal/storage"
)

type testProjectRepo struct {
	members map[string]*model.ProjectMember
}

func (m *testProjectRepo) Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error {
	return nil
}
func (m *testProjectRepo) GetByID(ctx context.Context, projectID string) (*model.Project, error) {
	return &model.Project{ID: projectID, Name: "Core", KeyPrefix: "COR"}, nil
}
func (m *testProjectRepo) ListByUser(ctx context.Context, userID, userEmail string) ([]model.Project, error) {
	return []model.Project{{ID: "proj-1", Name: "P1"}}, nil
}
func (m *testProjectRepo) Update(ctx context.Context, project *model.Project) error { return nil }
func (m *testProjectRepo) Delete(ctx context.Context, projectID string) error       { return nil }
func (m *testProjectRepo) GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
	if mem, ok := m.members[projectID+":"+userID]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (m *testProjectRepo) GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error) {
	if mem, ok := m.members[projectID+":"+email]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (m *testProjectRepo) ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	return []model.ProjectMember{}, nil
}
func (m *testProjectRepo) UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error {
	return nil
}
func (m *testProjectRepo) RemoveMember(ctx context.Context, projectID, userID string) error {
	return nil
}

func TestResolver_RoleChecking(t *testing.T) {
	projRepo := &testProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:user-admin": {
				UserID: "user-admin",
				Email:  "admin@example.com",
				Role:   model.RoleAdmin,
			},
			"proj-1:user-member": {
				UserID: "user-member",
				Email:  "member@example.com",
				Role:   model.RoleMember,
			},
			"proj-1:user-viewer": {
				UserID: "user-viewer",
				Email:  "viewer@example.com",
				Role:   model.RoleViewer,
			},
		},
	}

	ctx := context.Background()

	// 1. Admin user has Viewer, Member, Admin permissions
	adminResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-admin",
			Email:  "admin@example.com",
		},
	}, projRepo)

	if err := adminResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected admin to have Viewer role: %v", err)
	}
	if err := adminResolver.CheckRole(ctx, "proj-1", model.RoleMember); err != nil {
		t.Fatalf("expected admin to have Member role: %v", err)
	}
	if err := adminResolver.CheckRole(ctx, "proj-1", model.RoleAdmin); err != nil {
		t.Fatalf("expected admin to have Admin role: %v", err)
	}

	// 2. Member user has Viewer, Member permissions, but not Admin
	memberResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-member",
			Email:  "member@example.com",
		},
	}, projRepo)

	if err := memberResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected member to have Viewer role: %v", err)
	}
	if err := memberResolver.CheckRole(ctx, "proj-1", model.RoleMember); err != nil {
		t.Fatalf("expected member to have Member role: %v", err)
	}
	if err := memberResolver.CheckRole(ctx, "proj-1", model.RoleAdmin); err == nil {
		t.Fatalf("expected member to NOT have Admin role")
	}

	// 3. Viewer user has Viewer permissions, but not Member
	viewerResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-viewer",
			Email:  "viewer@example.com",
		},
	}, projRepo)

	if err := viewerResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected viewer to have Viewer role: %v", err)
	}
	if err := viewerResolver.CheckRole(ctx, "proj-1", model.RoleMember); err == nil {
		t.Fatalf("expected viewer to NOT have Member role")
	}

	// 4. Non-member user is rejected
	strangerResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-stranger",
			Email:  "stranger@example.com",
		},
	}, projRepo)

	if err := strangerResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err == nil {
		t.Fatalf("expected stranger to be denied access to project")
	}
}

func TestResolver_ResolveProjectID(t *testing.T) {
	r := NewResolver(Config{
		DefaultProjectID: "proj-fallback",
		User: model.UserContext{
			UserID: "user-1",
		},
	}, &testProjectRepo{members: make(map[string]*model.ProjectMember)})

	// Explicit argument provided
	reqWithProj := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"project_id": "proj-explicit",
			},
		},
	}
	proj, err := r.ResolveProjectID(reqWithProj)
	if err != nil || proj != "proj-explicit" {
		t.Fatalf("expected proj-explicit, got %s (err: %v)", proj, err)
	}

	// Fallback used when omitted
	reqEmpty := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}
	fallbackProj, err := r.ResolveProjectID(reqEmpty)
	if err != nil || fallbackProj != "proj-fallback" {
		t.Fatalf("expected proj-fallback, got %s (err: %v)", fallbackProj, err)
	}
}

func TestServer_Initialization(t *testing.T) {
	projRepo := &testProjectRepo{members: make(map[string]*model.ProjectMember)}

	serverDeps := ServerDeps{
		Config: Config{
			DefaultProjectID: "proj-core",
			User: model.UserContext{
				UserID: "test-agent",
				Email:  "agent@jynko.local",
				Name:   "Test Agent",
			},
		},
		ProjectRepo: projRepo,
		ProjectSvc:  service.NewProjectService(projRepo),
	}

	s := NewServer(serverDeps)
	if s == nil {
		t.Fatalf("expected non-nil server")
	}
	if s.MCPServer() == nil {
		t.Fatalf("expected non-nil mcpServer")
	}
}

func TestHTTPHandler_AuthenticationAndPerCallerRBAC(t *testing.T) {
	projRepo := &testProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:uid-viewer": {
				UserID: "uid-viewer",
				Email:  "viewer@company.com",
				Role:   model.RoleViewer,
			},
		},
	}

	appCfg := &config.Config{
		Env:                "prd",
		RawEnv:             "prd",
		IAPAudience:        "/projects/123/apps/jynko-mcp",
		IAPRequireJWT:      true,
		CORSAllowedOrigins: []string{"https://jynko.company.com"},
	}

	mockValidator := func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		if token == "viewer-jwt" && audience == "/projects/123/apps/jynko-mcp" {
			return &idtoken.Payload{
				Subject: "accounts.google.com:uid-viewer",
				Claims:  map[string]interface{}{"email": "viewer@company.com"},
			}, nil
		}
		return nil, errors.New("invalid token")
	}

	s := NewServer(ServerDeps{
		Config:       Config{DefaultProjectID: "proj-1"}, // No static User in prd SSE!
		AppConfig:    appCfg,
		JWTValidator: mockValidator,
		ProjectRepo:  projRepo,
		ProjectSvc:   service.NewProjectService(projRepo),
	})
	handler := s.HTTPHandler()

	// 1. Unauthenticated POST to /mcp -> 401 Unauthorized
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated SSE/HTTP MCP request, got %d", w.Code)
	}

	// 2. Disallowed cross-origin Origin header -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Origin", "https://evil.attacker.com")
	req.Header.Set("X-Goog-IAP-JWT-Assertion", "viewer-jwt")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed Origin header, got %d", w.Code)
	}

	// 3. Per-request context overrides static User in CheckRole
	reqCtx := WithUserContext(context.Background(), model.UserContext{
		UserID: "uid-viewer",
		Email:  "viewer@company.com",
	})
	if err := s.resolver.CheckRole(reqCtx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected viewer context to pass RoleViewer: %v", err)
	}
	err := s.resolver.CheckRole(reqCtx, "proj-1", model.RoleMember)
	if err == nil || !strings.Contains(err.Error(), "insufficient permissions") {
		t.Fatalf("expected viewer context to fail RoleMember, got %v", err)
	}
}

func TestHTTPHandler_CloudRunStatelessAndServiceAccount(t *testing.T) {
	saEmail := "jynko-scion-agent@aparaturafiscala-178410.iam.gserviceaccount.com"
	saSub := "109876543210987654321"

	projRepo := &testProjectRepo{
		members: map[string]*model.ProjectMember{
			// Invited by email only (Firestore doc ID differs from OIDC numeric sub)
			"proj-1:" + saEmail: {
				UserID: saEmail,
				Email:  saEmail,
				Role:   model.RoleMember,
			},
		},
	}

	appCfg := &config.Config{
		Env:           "prd",
		RawEnv:        "prd",
		IAPAudience:   "/projects/123/locations/europe-west3/services/jynko-plan-prd",
		IAPRequireJWT: true,
	}

	mockValidator := func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		if token == "sa-oidc-jwt" && audience == appCfg.IAPAudience {
			return &idtoken.Payload{
				Subject: saSub,
				Claims: map[string]interface{}{
					"email": saEmail,
				},
			}, nil
		}
		return nil, errors.New("invalid token")
	}

	s := NewServer(ServerDeps{
		Config:       Config{DefaultProjectID: "proj-1"},
		AppConfig:    appCfg,
		JWTValidator: mockValidator,
		ProjectRepo:  projRepo,
		ProjectSvc:   service.NewProjectService(projRepo),
	})
	handler := s.HTTPHandler()

	doMCP := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
		req.RemoteAddr = "127.0.0.1:12345"
		req.Host = "jynko-plan-prd-d2lb6vh75a-ey.a.run.app"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer sa-oidc-jwt")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	// 1. initialize behind Cloud Run proxy (127.0.0.1 RemoteAddr + external Host)
	wInit := doMCP(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test-agent","version":"1.0"}}}`)
	if wInit.Code != http.StatusOK {
		t.Fatalf("expected 200 on initialize behind Cloud Run proxy, got %d: %s", wInit.Code, wInit.Body.String())
	}

	// 2. tools/list without Mcp-Session-Id (stateless mode)
	wList := doMCP(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 on stateless tools/list, got %d: %s", wList.Code, wList.Body.String())
	}
	if !strings.Contains(wList.Body.String(), "list_projects") {
		t.Fatalf("expected tools/list to include list_projects, got: %s", wList.Body.String())
	}

	// 3. Stateless tools/call without Mcp-Session-Id
	wCall := doMCP(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_projects","arguments":{}}}`)
	if wCall.Code != http.StatusOK {
		t.Fatalf("expected 200 on stateless tools/call, got %d: %s", wCall.Code, wCall.Body.String())
	}
	if !strings.Contains(wCall.Body.String(), "proj-1") {
		t.Fatalf("expected tools/call list_projects to return proj-1, got: %s", wCall.Body.String())
	}

	// 4. Verify service account RBAC lookup by email works via CheckRole even when sub != email
	saCtx := WithUserContext(context.Background(), model.UserContext{
		UserID: saSub,
		Email:  saEmail,
	})
	if err := s.resolver.CheckRole(saCtx, "proj-1", model.RoleMember); err != nil {
		t.Fatalf("expected service account invited by email to pass RoleMember via GetMemberByEmail: %v", err)
	}
}

type testAttachmentRepo struct {
	items map[string]*model.Attachment
}

func (m *testAttachmentRepo) List(ctx context.Context, projectID string, itemID string) ([]model.Attachment, error) {
	var res []model.Attachment
	for _, a := range m.items {
		if a.ProjectID == projectID {
			if itemID == "" || a.WorkItemID == itemID {
				res = append(res, *a)
			}
		}
	}
	return res, nil
}

func (m *testAttachmentRepo) GetByID(ctx context.Context, projectID, attachmentID string) (*model.Attachment, error) {
	if a, ok := m.items[attachmentID]; ok && a.ProjectID == projectID {
		copy := *a
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *testAttachmentRepo) Create(ctx context.Context, attachment *model.Attachment) error {
	now := time.Now().UTC()
	attachment.CreatedAt = now
	attachment.UpdatedAt = now
	m.items[attachment.ID] = attachment
	return nil
}

func (m *testAttachmentRepo) Delete(ctx context.Context, projectID, attachmentID string) error {
	delete(m.items, attachmentID)
	return nil
}

type testItemRepo struct {
	items map[string]*model.WorkItem
}

func (m *testItemRepo) List(_ context.Context, _ string, _ model.WorkItemFilter) ([]model.WorkItem, error) {
	var res []model.WorkItem
	for _, it := range m.items {
		res = append(res, *it)
	}
	return res, nil
}
func (m *testItemRepo) GetByID(_ context.Context, _, itemID string) (*model.WorkItem, error) {
	if it, ok := m.items[itemID]; ok {
		cp := *it
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}
func (m *testItemRepo) GetByKey(_ context.Context, _, key string) (*model.WorkItem, error) {
	for _, it := range m.items {
		if it.Key == key {
			cp := *it
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (m *testItemRepo) Create(_ context.Context, item *model.WorkItem) error {
	m.items[item.ID] = item
	return nil
}
func (m *testItemRepo) Update(_ context.Context, item *model.WorkItem) error {
	m.items[item.ID] = item
	return nil
}
func (m *testItemRepo) Delete(_ context.Context, _, itemID string) error {
	delete(m.items, itemID)
	return nil
}
func (m *testItemRepo) ListSubItems(_ context.Context, _, _ string) ([]model.SubWorkItem, error) {
	return []model.SubWorkItem{}, nil
}
func (m *testItemRepo) GetSubItemByID(_ context.Context, _, _, _ string) (*model.SubWorkItem, error) {
	return nil, repository.ErrNotFound
}
func (m *testItemRepo) CreateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (m *testItemRepo) UpdateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (m *testItemRepo) DeleteSubItem(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *testItemRepo) ListComments(_ context.Context, _, _ string) ([]model.Comment, error) {
	return []model.Comment{}, nil
}
func (m *testItemRepo) GetCommentByID(_ context.Context, _, _, _ string) (*model.Comment, error) {
	return nil, repository.ErrNotFound
}
func (m *testItemRepo) CreateComment(_ context.Context, _ string, _ *model.Comment) error {
	return nil
}
func (m *testItemRepo) UpdateComment(_ context.Context, _, _, _, _ string) (*model.Comment, error) {
	return nil, repository.ErrNotFound
}
func (m *testItemRepo) DeleteComment(_ context.Context, _, _, _ string) error {
	return nil
}

func TestAttachmentTools_ListAndDownload(t *testing.T) {
	ctx := context.Background()
	projRepo := &testProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:user-viewer": {
				UserID: "user-viewer",
				Email:  "viewer@test.com",
				Role:   model.RoleViewer,
			},
		},
	}

	itemRepo := &testItemRepo{
		items: map[string]*model.WorkItem{
			"item-1": {
				ID:        "item-1",
				ProjectID: "proj-1",
				Key:       "PLAN-8",
				Title:     "Update mcp server to include attachements",
			},
		},
	}
	itemSvc := service.NewItemService(itemRepo)

	store, err := storage.NewLocalStorageProvider(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create local storage provider: %v", err)
	}

	attRepo := &testAttachmentRepo{
		items: make(map[string]*model.Attachment),
	}
	attSvc := service.NewAttachmentService(attRepo, store)

	// Upload a text attachment and an image attachment
	uploader := model.UserContext{UserID: "user-viewer", Name: "Test User"}
	textContent := []byte("Hello, Jynko Plan MCP!")
	textAtt, err := attSvc.UploadAttachment(
		ctx, "proj-1", uploader, "item-1", "notes.txt", "text/plain", int64(len(textContent)), bytes.NewReader(textContent),
	)
	if err != nil {
		t.Fatalf("failed to upload text attachment: %v", err)
	}

	imgContent := []byte("\x89PNG\r\n\x1a\nfakeimagebytes")
	imgAtt, err := attSvc.UploadAttachment(
		ctx, "proj-1", uploader, "item-1", "diagram.png", "image/png", int64(len(imgContent)), bytes.NewReader(imgContent),
	)
	if err != nil {
		t.Fatalf("failed to upload image attachment: %v", err)
	}

	// Another project attachment not linked to any item
	generalContent := []byte("General project readme")
	_, err = attSvc.UploadAttachment(
		ctx, "proj-1", uploader, "", "general.txt", "text/plain", int64(len(generalContent)), bytes.NewReader(generalContent),
	)
	if err != nil {
		t.Fatalf("failed to upload general attachment: %v", err)
	}

	appCfg := &config.Config{
		Env:          config.EnvDev,
		RawEnv:       config.EnvDev,
		DevUserID:    "user-viewer",
		DevUserEmail: "viewer@test.com",
		DevUserName:  "Viewer User",
	}

	serverDeps := ServerDeps{
		Config: Config{
			DefaultProjectID: "proj-1",
			User: model.UserContext{
				UserID: "user-viewer",
				Email:  "viewer@test.com",
			},
		},
		AppConfig:     appCfg,
		ProjectRepo:   projRepo,
		ItemSvc:       itemSvc,
		AttachmentSvc: attSvc,
	}
	s := NewServer(serverDeps)
	handler := s.HTTPHandler()

	doMCP := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	// 1. Initialize
	wInit := doMCP(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test-agent","version":"1.0"}}}`)
	if wInit.Code != http.StatusOK {
		t.Fatalf("initialize failed: %d: %s", wInit.Code, wInit.Body.String())
	}

	// 2. tools/list includes list_attachments, get_attachment, download_attachment
	wListTools := doMCP(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if wListTools.Code != http.StatusOK {
		t.Fatalf("tools/list failed: %d: %s", wListTools.Code, wListTools.Body.String())
	}
	listToolsBody := wListTools.Body.String()
	for _, toolName := range []string{"list_attachments", "get_attachment", "download_attachment"} {
		if !strings.Contains(listToolsBody, toolName) {
			t.Fatalf("expected tools/list to contain %q, got: %s", toolName, listToolsBody)
		}
	}

	// 3. Test list_attachments (filtered by item key PLAN-8)
	wListAtts := doMCP(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_attachments","arguments":{"item_id":"PLAN-8"}}}`)
	if wListAtts.Code != http.StatusOK {
		t.Fatalf("list_attachments call failed: %d: %s", wListAtts.Code, wListAtts.Body.String())
	}
	if !strings.Contains(wListAtts.Body.String(), textAtt.ID) || !strings.Contains(wListAtts.Body.String(), imgAtt.ID) {
		t.Fatalf("expected item-1 attachments in list_attachments, got: %s", wListAtts.Body.String())
	}

	// 4. Test list_attachments (all project attachments)
	wListAll := doMCP(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_attachments","arguments":{}}}`)
	if wListAll.Code != http.StatusOK {
		t.Fatalf("list_attachments all call failed: %d: %s", wListAll.Code, wListAll.Body.String())
	}
	if !strings.Contains(wListAll.Body.String(), "general.txt") {
		t.Fatalf("expected general.txt in list_attachments all, got: %s", wListAll.Body.String())
	}

	// 5. Test get_attachment for text file
	wGetTxt := doMCP(fmt.Sprintf(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_attachment","arguments":{"attachment_id":%q}}}`, textAtt.ID))
	if wGetTxt.Code != http.StatusOK {
		t.Fatalf("get_attachment text failed: %d: %s", wGetTxt.Code, wGetTxt.Body.String())
	}
	getTxtBody := wGetTxt.Body.String()
	if !strings.Contains(getTxtBody, "Hello, Jynko Plan MCP!") {
		t.Fatalf("expected text content in get_attachment, got: %s", getTxtBody)
	}

	// 6. Test download_attachment for image file (includes ImageContent and base64 data)
	wDlImg := doMCP(fmt.Sprintf(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"download_attachment","arguments":{"attachment_id":%q}}}`, imgAtt.ID))
	if wDlImg.Code != http.StatusOK {
		t.Fatalf("download_attachment image failed: %d: %s", wDlImg.Code, wDlImg.Body.String())
	}
	expectedImgB64 := base64.StdEncoding.EncodeToString(imgContent)
	if !strings.Contains(wDlImg.Body.String(), expectedImgB64) {
		t.Fatalf("expected base64 image data in download_attachment, got: %s", wDlImg.Body.String())
	}

	// 7. Test get_work_item includes attachments array
	wGetWorkItem := doMCP(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"get_work_item","arguments":{"item_key_or_id":"PLAN-8"}}}`)
	if wGetWorkItem.Code != http.StatusOK {
		t.Fatalf("get_work_item call failed: %d: %s", wGetWorkItem.Code, wGetWorkItem.Body.String())
	}
	getWIBody := wGetWorkItem.Body.String()
	if !strings.Contains(getWIBody, "attachments") || !strings.Contains(getWIBody, textAtt.ID) {
		t.Fatalf("expected attachments in get_work_item result, got: %s", getWIBody)
	}

	// 8. Test reading attachment resource
	wReadRes := doMCP(fmt.Sprintf(`{"jsonrpc":"2.0","id":8,"method":"resources/read","params":{"uri":"jynko://projects/proj-1/attachments/%s"}}`, textAtt.ID))
	if wReadRes.Code != http.StatusOK {
		t.Fatalf("resources/read failed: %d: %s", wReadRes.Code, wReadRes.Body.String())
	}
	if !strings.Contains(wReadRes.Body.String(), "Hello, Jynko Plan MCP!") {
		t.Fatalf("expected text in resources/read, got: %s", wReadRes.Body.String())
	}
}

// Suppress unused imports
var _ = time.Now
