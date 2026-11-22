package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod/lib/proto"
	"google.golang.org/api/idtoken"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/e2e"
	"github.com/gabihodoroaga/jynko-plan/internal/handler"
	"github.com/gabihodoroaga/jynko-plan/internal/mcp"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

type stubProjectRepo struct {
	members map[string]*model.ProjectMember
}

func (s *stubProjectRepo) Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error {
	return nil
}
func (s *stubProjectRepo) GetByID(ctx context.Context, projectID string) (*model.Project, error) {
	return &model.Project{ID: projectID, Name: "Core", KeyPrefix: "COR", MemberIDs: []string{"dev-user-1"}}, nil
}
func (s *stubProjectRepo) ListByUser(ctx context.Context, userID, userEmail string) ([]model.Project, error) {
	return []model.Project{{ID: "proj-1", Name: "Core", KeyPrefix: "COR", MemberIDs: []string{"dev-user-1"}}}, nil
}
func (s *stubProjectRepo) Update(ctx context.Context, project *model.Project) error { return nil }
func (s *stubProjectRepo) Delete(ctx context.Context, projectID string) error       { return nil }
func (s *stubProjectRepo) GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
	if mem, ok := s.members[projectID+":"+userID]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (s *stubProjectRepo) GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error) {
	if mem, ok := s.members[projectID+":"+strings.ToLower(strings.TrimSpace(email))]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (s *stubProjectRepo) ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	return []model.ProjectMember{}, nil
}
func (s *stubProjectRepo) UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error {
	return nil
}
func (s *stubProjectRepo) RemoveMember(ctx context.Context, projectID, userID string) error {
	return nil
}

func TestRouter_MCPEndpointBehindCloudRunIAP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><body>Jynko SPA</body></html>"), 0o644); err != nil {
		t.Fatalf("failed to write dummy index.html: %v", err)
	}

	saEmail := "jynko-scion-agent@aparaturafiscala-178410.iam.gserviceaccount.com"
	saSub := "109876543210987654321"
	projRepo := &stubProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:" + saEmail: {
				UserID: saEmail,
				Email:  saEmail,
				Role:   model.RoleMember,
			},
		},
	}

	cfg := &config.Config{
		Env:                config.EnvPrd,
		RawEnv:             config.EnvPrd,
		DisableDevAuth:     true,
		GoogleCloudProject: "aparaturafiscala-178410",
		IAPAudience:        "/projects/12345/locations/europe-west3/services/jynko-plan-prd",
		IAPRequireJWT:      true,
		StaticDir:          staticDir,
	}

	mockValidator := func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		if token == "valid-sa-token" && audience == cfg.IAPAudience {
			return &idtoken.Payload{
				Subject: saSub,
				Claims: map[string]interface{}{
					"email": saEmail,
				},
			}, nil
		}
		return nil, errors.New("invalid token")
	}

	projectSvc := service.NewProjectService(projRepo)
	mcpSrv := mcp.NewServer(mcp.ServerDeps{
		Config:       mcp.Config{DefaultProjectID: "proj-1"},
		AppConfig:    cfg,
		JWTValidator: mockValidator,
		ProjectRepo:  projRepo,
		ProjectSvc:   projectSvc,
	})

	router := SetupRouterWithMCP(
		cfg,
		projRepo,
		handler.NewAuthHandler(),
		handler.NewProjectHandler(projectSvc),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		mcpSrv.HTTPHandler(),
	)

	// 1. Unauthenticated POST /mcp returns 401 Unauthorized (never 200 index.html)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated POST /mcp, got %d (%s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "Jynko SPA") {
		t.Fatalf("unauthenticated /mcp must not return index.html")
	}

	// 2. Unmatched /mcp/subpath returns 404 JSON (never 200 index.html)
	reqSub := httptest.NewRequest(http.MethodGet, "/mcp/unknown", nil)
	wSub := httptest.NewRecorder()
	router.ServeHTTP(wSub, reqSub)
	if wSub.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for /mcp/unknown, got %d (%s)", wSub.Code, wSub.Body.String())
	}
	if !strings.Contains(wSub.Body.String(), "route not found") || strings.Contains(wSub.Body.String(), "Jynko SPA") {
		t.Fatalf("expected JSON 404 for /mcp/unknown, got: %s", wSub.Body.String())
	}

	doAuthMCP := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
		r.RemoteAddr = "127.0.0.1:12345"
		r.Host = "jynko-plan-prd-d2lb6vh75a-ey.a.run.app"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-sa-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, r)
		return rec
	}

	// 3. Authenticated initialize behind Cloud Run local reverse proxy
	wInit := doAuthMCP(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"scion","version":"1.0"}}}`)
	if wInit.Code != http.StatusOK {
		t.Fatalf("expected 200 on authenticated initialize, got %d: %s", wInit.Code, wInit.Body.String())
	}

	// 4. Authenticated tools/list without session state
	wList := doAuthMCP(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if wList.Code != http.StatusOK || !strings.Contains(wList.Body.String(), "list_projects") {
		t.Fatalf("expected 200 with tools list on stateless tools/list, got %d: %s", wList.Code, wList.Body.String())
	}

	// 5. Stateless tools/call across independent requests without Mcp-Session-Id
	wCall := doAuthMCP(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_projects","arguments":{}}}`)
	if wCall.Code != http.StatusOK || !strings.Contains(wCall.Body.String(), "proj-1") {
		t.Fatalf("expected 200 with proj-1 on stateless tools/call, got %d: %s", wCall.Code, wCall.Body.String())
	}
}

func TestRouter_APINoStoreCacheControl(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projRepo := &stubProjectRepo{members: map[string]*model.ProjectMember{}}
	cfg := &config.Config{
		Env:          config.EnvDev,
		RawEnv:       config.EnvDev,
		DevUserID:    "dev-user-1",
		DevUserEmail: "dev@jynko.local",
		DevUserName:  "Local Developer",
	}
	projectSvc := service.NewProjectService(projRepo)

	router := SetupRouterWithMCP(
		cfg,
		projRepo,
		handler.NewAuthHandler(),
		handler.NewProjectHandler(projectSvc),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /api/v1/projects, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control: no-store on /api/v1/projects, got %q", got)
	}
}

func TestPWA_TopBarAndDragRefresh(t *testing.T) {
	staticDir := filepath.Clean(filepath.Join("..", "..", "..", "frontend", "dist", "jynko-frontend", "browser"))
	if _, err := os.Stat(filepath.Join(staticDir, "index.html")); err != nil {
		t.Skipf("skipping browser PWA refresh test: production bundle not built at %s", staticDir)
	}

	browser, err := e2e.LaunchBrowser()
	if err != nil {
		t.Skipf("skipping browser PWA refresh test: headless browser unavailable: %v", err)
	}
	defer browser.MustClose()

	gin.SetMode(gin.TestMode)
	projRepo := &stubProjectRepo{members: map[string]*model.ProjectMember{}}
	cfg := &config.Config{
		Env:          config.EnvDev,
		RawEnv:       config.EnvDev,
		DevUserID:    "dev-user-1",
		DevUserEmail: "dev@jynko.local",
		DevUserName:  "Local Developer",
		StaticDir:    staticDir,
	}
	projectSvc := service.NewProjectService(projRepo)
	router := SetupRouterWithMCP(
		cfg,
		projRepo,
		handler.NewAuthHandler(),
		handler.NewProjectHandler(projectSvc),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	var mu sync.Mutex
	items := []map[string]any{
		{
			"id":          "item-1",
			"project_id":  "proj-1",
			"key":         "COR-1",
			"sequence_id": 1,
			"title":       "Initial Item",
			"description": "",
			"status":      "TODO",
			"priority":    "MEDIUM",
			"assignee_id": "dev-user-1",
			"reporter_id": "dev-user-1",
			"label_ids":   []string{},
			"created_at":  "2026-11-22T10:00:00Z",
			"updated_at":  "2026-11-22T10:00:00Z",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/proj-1/items":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			mu.Lock()
			snapshot := make([]map[string]any, len(items))
			copy(snapshot, items)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(snapshot)
			return
		case "/api/v1/projects/proj-1/members":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"user_id":"dev-user-1","email":"dev@jynko.local","name":"Local Developer","role":"ADMIN"}]`))
			return
		case "/api/v1/projects/proj-1/modules",
			"/api/v1/projects/proj-1/epics",
			"/api/v1/projects/proj-1/cycles",
			"/api/v1/projects/proj-1/labels",
			"/api/v1/projects/proj-1/collections",
			"/api/v1/projects/proj-1/pages",
			"/api/v1/projects/proj-1/stickies":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
			return
		default:
			router.ServeHTTP(w, r)
		}
	}))
	defer srv.Close()

	page := browser.MustPage("")
	defer page.MustClose()
	_ = proto.EmulationSetDeviceMetricsOverride{
		Width:             1280,
		Height:            800,
		DeviceScaleFactor: 1,
		Mobile:            false,
	}.Call(page)
	page.MustNavigate(srv.URL + "/home").MustWaitLoad()

	waitForText := func(substr string) {
		deadline := time.Now().Add(10 * time.Second)
		lastBody := ""
		for time.Now().Before(deadline) {
			if bodyEl, err := page.Element("body"); err == nil && bodyEl != nil {
				if text, err := bodyEl.Text(); err == nil {
					lastBody = text
					if strings.Contains(text, substr) {
						return
					}
				}
			}
			time.Sleep(150 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for %q on page; body=%q", substr, lastBody)
	}

	waitForReload := func(markerExpr string, failMsg string) {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			res, err := page.Eval(markerExpr)
			if err == nil && res.Value.Bool() {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatal(failMsg)
	}

	waitForText("Initial Item")

	// 1. Add item on server and click top-bar refresh button
	mu.Lock()
	items = append(items, map[string]any{
		"id":          "item-2",
		"project_id":  "proj-1",
		"key":         "COR-2",
		"sequence_id": 2,
		"title":       "Fresh Item After Top Click",
		"description": "",
		"status":      "TODO",
		"priority":    "HIGH",
		"assignee_id": "dev-user-1",
		"reporter_id": "dev-user-1",
		"label_ids":   []string{},
		"created_at":  "2026-11-22T10:05:00Z",
		"updated_at":  "2026-11-22T10:05:00Z",
	})
	mu.Unlock()

	page.MustEval(`() => { window.__beforeTopRefresh = true; }`)
	page.MustElement(`header button[data-testid="app-refresh-btn"]`).MustClick()

	waitForReload(`() => typeof window.__beforeTopRefresh === 'undefined'`, "top-bar refresh button did not reload the page")
	waitForText("Fresh Item After Top Click")

	// 2. Add item on server and perform mouse click-and-drag down gesture
	mu.Lock()
	items = append(items, map[string]any{
		"id":          "item-3",
		"project_id":  "proj-1",
		"key":         "COR-3",
		"sequence_id": 3,
		"title":       "Fresh Item After Mouse Drag",
		"description": "",
		"status":      "TODO",
		"priority":    "URGENT",
		"assignee_id": "dev-user-1",
		"reporter_id": "dev-user-1",
		"label_ids":   []string{},
		"created_at":  "2026-11-22T10:10:00Z",
		"updated_at":  "2026-11-22T10:10:00Z",
	})
	mu.Unlock()

	page.MustEval(`() => { window.__beforeDragRefresh = true; }`)
	coords := page.MustEval(`() => {
		const main = document.querySelector('main');
		const r = main.getBoundingClientRect();
		return { x: r.left + 48, y: r.top + 12 };
	}`)
	startX := coords.Get("x").Num()
	startY := coords.Get("y").Num()
	leftButtons := 1

	_ = proto.InputDispatchMouseEvent{
		Type:       proto.InputDispatchMouseEventTypeMousePressed,
		X:          startX,
		Y:          startY,
		Button:     proto.InputMouseButtonLeft,
		Buttons:    &leftButtons,
		ClickCount: 1,
	}.Call(page)
	time.Sleep(40 * time.Millisecond)
	_ = proto.InputDispatchMouseEvent{
		Type:    proto.InputDispatchMouseEventTypeMouseMoved,
		X:       startX,
		Y:       startY + 30,
		Button:  proto.InputMouseButtonLeft,
		Buttons: &leftButtons,
	}.Call(page)
	time.Sleep(40 * time.Millisecond)
	_ = proto.InputDispatchMouseEvent{
		Type:    proto.InputDispatchMouseEventTypeMouseMoved,
		X:       startX,
		Y:       startY + 140,
		Button:  proto.InputMouseButtonLeft,
		Buttons: &leftButtons,
	}.Call(page)
	time.Sleep(80 * time.Millisecond)

	if hasReady, _, _ := page.Has(`[data-testid="pull-to-refresh-indicator"][data-pull-ready="true"]`); !hasReady {
		t.Fatalf("mouse click-and-drag down did not show ready pull-to-refresh indicator")
	}

	_ = proto.InputDispatchMouseEvent{
		Type:       proto.InputDispatchMouseEventTypeMouseReleased,
		X:          startX,
		Y:          startY + 140,
		Button:     proto.InputMouseButtonLeft,
		ClickCount: 1,
	}.Call(page)

	waitForReload(`() => typeof window.__beforeDragRefresh === 'undefined'`, "mouse click-and-drag down did not reload the page")
	waitForText("Fresh Item After Mouse Drag")

	// 3. Add item on server and perform touch pull-down gesture on mobile viewport
	_ = proto.EmulationSetDeviceMetricsOverride{
		Width:             390,
		Height:            844,
		DeviceScaleFactor: 2,
		Mobile:            true,
	}.Call(page)
	_ = proto.EmulationSetTouchEmulationEnabled{Enabled: true}.Call(page)
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	items = append(items, map[string]any{
		"id":          "item-4",
		"project_id":  "proj-1",
		"key":         "COR-4",
		"sequence_id": 4,
		"title":       "Fresh Item After Touch Pull",
		"description": "",
		"status":      "TODO",
		"priority":    "LOW",
		"assignee_id": "dev-user-1",
		"reporter_id": "dev-user-1",
		"label_ids":   []string{},
		"created_at":  "2026-11-22T10:15:00Z",
		"updated_at":  "2026-11-22T10:15:00Z",
	})
	mu.Unlock()

	page.MustEval(`() => { window.__beforeTouchRefresh = true; }`)
	touchCoords := page.MustEval(`() => {
		const main = document.querySelector('main');
		const r = main.getBoundingClientRect();
		return { x: r.left + 48, y: r.top + 12 };
	}`)
	tx := touchCoords.Get("x").Num()
	ty := touchCoords.Get("y").Num()
	touchID := 1.0

	_ = proto.InputDispatchTouchEvent{
		Type: proto.InputDispatchTouchEventTypeTouchStart,
		TouchPoints: []*proto.InputTouchPoint{
			{X: tx, Y: ty, ID: &touchID},
		},
	}.Call(page)
	time.Sleep(40 * time.Millisecond)
	_ = proto.InputDispatchTouchEvent{
		Type: proto.InputDispatchTouchEventTypeTouchMove,
		TouchPoints: []*proto.InputTouchPoint{
			{X: tx, Y: ty + 30, ID: &touchID},
		},
	}.Call(page)
	time.Sleep(40 * time.Millisecond)
	_ = proto.InputDispatchTouchEvent{
		Type: proto.InputDispatchTouchEventTypeTouchMove,
		TouchPoints: []*proto.InputTouchPoint{
			{X: tx, Y: ty + 140, ID: &touchID},
		},
	}.Call(page)
	time.Sleep(80 * time.Millisecond)

	if hasReady, _, _ := page.Has(`[data-testid="pull-to-refresh-indicator"][data-pull-ready="true"]`); !hasReady {
		t.Fatalf("touch pull-down did not show ready pull-to-refresh indicator")
	}

	_ = proto.InputDispatchTouchEvent{
		Type:        proto.InputDispatchTouchEventTypeTouchEnd,
		TouchPoints: []*proto.InputTouchPoint{},
	}.Call(page)

	waitForReload(`() => typeof window.__beforeTouchRefresh === 'undefined'`, "touch pull-down did not reload the page")
	waitForText("Fresh Item After Touch Pull")
}
