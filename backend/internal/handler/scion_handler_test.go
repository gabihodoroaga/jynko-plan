package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/scion"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

type fakeProjectRepo struct {
	projects map[string]*model.Project
}

func (f *fakeProjectRepo) Create(_ context.Context, p *model.Project, _ *model.ProjectMember) error {
	f.projects[p.ID] = p
	return nil
}
func (f *fakeProjectRepo) GetByID(_ context.Context, id string) (*model.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *p
	if p.Settings != nil {
		s := *p.Settings
		cp.Settings = &s
	}
	return &cp, nil
}
func (f *fakeProjectRepo) ListByUser(_ context.Context, _, _ string) ([]model.Project, error) {
	return []model.Project{}, nil
}
func (f *fakeProjectRepo) Update(_ context.Context, p *model.Project) error {
	f.projects[p.ID] = p
	return nil
}
func (f *fakeProjectRepo) Delete(_ context.Context, id string) error {
	delete(f.projects, id)
	return nil
}
func (f *fakeProjectRepo) GetMember(_ context.Context, _, _ string) (*model.ProjectMember, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeProjectRepo) GetMemberByEmail(_ context.Context, _, _ string) (*model.ProjectMember, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeProjectRepo) ListMembers(_ context.Context, _ string) ([]model.ProjectMember, error) {
	return []model.ProjectMember{}, nil
}
func (f *fakeProjectRepo) UpsertMember(_ context.Context, _ string, _ *model.ProjectMember) error {
	return nil
}
func (f *fakeProjectRepo) RemoveMember(_ context.Context, _, _ string) error {
	return nil
}

type fakeModuleRepo struct {
	modules map[string]*model.Module
}

func (f *fakeModuleRepo) List(_ context.Context, _ string) ([]model.Module, error) {
	return []model.Module{}, nil
}
func (f *fakeModuleRepo) GetByID(_ context.Context, _, moduleID string) (*model.Module, error) {
	m, ok := f.modules[moduleID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *m
	return &cp, nil
}
func (f *fakeModuleRepo) Create(_ context.Context, m *model.Module) error {
	f.modules[m.ID] = m
	return nil
}
func (f *fakeModuleRepo) Update(_ context.Context, m *model.Module) error {
	f.modules[m.ID] = m
	return nil
}
func (f *fakeModuleRepo) Delete(_ context.Context, _, moduleID string) error {
	delete(f.modules, moduleID)
	return nil
}

type fakeItemRepo struct {
	items    map[string]*model.WorkItem
	comments []*model.Comment
}

func (f *fakeItemRepo) List(_ context.Context, _ string, _ model.WorkItemFilter) ([]model.WorkItem, error) {
	return []model.WorkItem{}, nil
}
func (f *fakeItemRepo) GetByID(_ context.Context, _, itemID string) (*model.WorkItem, error) {
	it, ok := f.items[itemID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *it
	return &cp, nil
}
func (f *fakeItemRepo) GetByKey(_ context.Context, _, _ string) (*model.WorkItem, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeItemRepo) Create(_ context.Context, item *model.WorkItem) error {
	f.items[item.ID] = item
	return nil
}
func (f *fakeItemRepo) Update(_ context.Context, item *model.WorkItem) error {
	cp := *item
	f.items[item.ID] = &cp
	return nil
}
func (f *fakeItemRepo) Delete(_ context.Context, _, itemID string) error {
	delete(f.items, itemID)
	return nil
}
func (f *fakeItemRepo) ListSubItems(_ context.Context, _, _ string) ([]model.SubWorkItem, error) {
	return []model.SubWorkItem{}, nil
}
func (f *fakeItemRepo) GetSubItemByID(_ context.Context, _, _, _ string) (*model.SubWorkItem, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeItemRepo) CreateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (f *fakeItemRepo) UpdateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (f *fakeItemRepo) DeleteSubItem(_ context.Context, _, _, _ string) error {
	return nil
}
func (f *fakeItemRepo) ListComments(_ context.Context, _, _ string) ([]model.Comment, error) {
	out := make([]model.Comment, 0, len(f.comments))
	for _, c := range f.comments {
		out = append(out, *c)
	}
	return out, nil
}
func (f *fakeItemRepo) GetCommentByID(_ context.Context, _, _, id string) (*model.Comment, error) {
	for _, c := range f.comments {
		if c.ID == id {
			cp := *c
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (f *fakeItemRepo) CreateComment(_ context.Context, _ string, c *model.Comment) error {
	if c.ID == "" {
		c.ID = "comment-" + string(rune('1'+len(f.comments)))
	}
	cp := *c
	f.comments = append(f.comments, &cp)
	return nil
}
func (f *fakeItemRepo) UpdateComment(_ context.Context, _, _, _, _ string) (*model.Comment, error) {
	return nil, nil
}
func (f *fakeItemRepo) DeleteComment(_ context.Context, _, _, _ string) error {
	return nil
}

type fakeHubClient struct {
	agents      map[string]scion.Agent
	createdReqs []scion.CreateAgentRequest
	messaged    []string
	stopped     []string
}

func (f *fakeHubClient) ListAgents(_ context.Context, _ string) ([]scion.Agent, error) {
	out := make([]scion.Agent, 0, len(f.agents))
	for _, a := range f.agents {
		out = append(out, a)
	}
	return out, nil
}
func (f *fakeHubClient) GetAgent(_ context.Context, agentID string) (*scion.Agent, error) {
	a, ok := f.agents[agentID]
	if !ok {
		return nil, scion.ErrAgentNotFound
	}
	cp := a
	return &cp, nil
}
func (f *fakeHubClient) CreateAgent(_ context.Context, req scion.CreateAgentRequest) (*scion.Agent, error) {
	f.createdReqs = append(f.createdReqs, req)
	id := "11111111-1111-4111-8111-" + req.Name
	if len(id) > 36 {
		id = id[:36]
	}
	harness := req.HarnessConfig
	if harness == "" && req.Config != nil {
		harness = req.Config.Harness
	}
	a := scion.Agent{
		ID:            id,
		Slug:          req.Name,
		Name:          req.Name,
		ProjectID:     req.ProjectID,
		Harness:       harness,
		HarnessConfig: req.HarnessConfig,
		Phase:         "running",
	}
	f.agents[id] = a
	return &a, nil
}
func (f *fakeHubClient) StartAgent(_ context.Context, agentID string) (*scion.Agent, error) {
	a := f.agents[agentID]
	a.Phase = "running"
	f.agents[agentID] = a
	return &a, nil
}
func (f *fakeHubClient) StopAgent(_ context.Context, agentID string) error {
	f.stopped = append(f.stopped, agentID)
	a := f.agents[agentID]
	a.Phase = "stopped"
	f.agents[agentID] = a
	return nil
}
func (f *fakeHubClient) SendMessage(_ context.Context, agentID, messageText string, _ bool) error {
	f.messaged = append(f.messaged, agentID+":"+messageText)
	return nil
}
func (f *fakeHubClient) EnsureRunningAndSendMessage(_ context.Context, agentID, messageText string) error {
	f.messaged = append(f.messaged, agentID+":"+messageText)
	return nil
}

func setupScionTestEnv() (*CommentHandler, *ScionHandler, *fakeProjectRepo, *fakeItemRepo, *fakeHubClient) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		ScionHubURL:             "https://scion.example.com",
		ScionHubToken:           "scion_pat_test",
		ScionDefaultProjectSlug: "global",
		ScionDefaultTemplate:    "jynko-dev",
	}
	projRepo := &fakeProjectRepo{
		projects: map[string]*model.Project{
			"proj-1": {
				ID:        "proj-1",
				Name:      "Jynko",
				KeyPrefix: "JYN",
				Settings: &model.ProjectSettings{
					ScionEnabled:      true,
					DefaultGitRepoURL: "https://github.com/org/repo",
					DefaultBaseBranch: "dev",
					ScionProjectSlug:  "global",
					ScionTemplate:     "jynko-dev",
					ImplementHarness:  "claude",
					ReviewHarness:     "antigravity",
				},
			},
		},
	}
	modRepo := &fakeModuleRepo{modules: map[string]*model.Module{}}
	itemRepo := &fakeItemRepo{
		items: map[string]*model.WorkItem{
			"item-1": {
				ID:          "item-1",
				ProjectID:   "proj-1",
				Key:         "JYN-42",
				Title:       "Implement Scion commands",
				Description: "Wire slash commands",
			},
		},
	}
	hub := &fakeHubClient{agents: map[string]scion.Agent{}}

	projSvc := service.NewProjectService(projRepo)
	modSvc := service.NewModuleService(modRepo)
	itemSvc := service.NewItemService(itemRepo)
	orch := service.NewScionOrchestrator(cfg, hub, projSvc, modSvc, itemSvc)

	ch := NewCommentHandlerWithScion(itemSvc, orch)
	sh := NewScionHandler(cfg, hub, projSvc, itemSvc)
	return ch, sh, projRepo, itemRepo, hub
}

func callCommentCreate(ch *CommentHandler, body string) *httptest.ResponseRecorder {
	r := gin.New()
	r.Use(middleware.IAPAuthMiddleware(&config.Config{
		Env:          "dev",
		DevUserID:    "u-1",
		DevUserEmail: "alice@example.com",
		DevUserName:  "Alice",
	}))
	r.POST("/projects/:projectId/items/:itemId/comments", ch.CreateComment)
	payload, _ := json.Marshal(map[string]string{"body": body})
	req := httptest.NewRequest(http.MethodPost, "/projects/proj-1/items/item-1/comments", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCommentHandler_SlashCommandsAndScionHandler(t *testing.T) {
	ch, sh, projRepo, itemRepo, hub := setupScionTestEnv()

	// 1. Regular human comment does not invoke Scion
	w := callCommentCreate(ch, "Looks good to me!")
	if w.Code != http.StatusCreated {
		t.Fatalf("regular comment status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	if len(hub.createdReqs) != 0 {
		t.Fatalf("expected 0 scion agents created for normal comment, got %d", len(hub.createdReqs))
	}

	// 2. Disabled project returns 400 and does not save comment
	projRepo.projects["proj-1"].Settings.ScionEnabled = false
	beforeComments := len(itemRepo.comments)
	w = callCommentCreate(ch, "/implement build it")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("disabled project status = %d, want 400", w.Code)
	}
	if len(itemRepo.comments) != beforeComments {
		t.Fatalf("expected comment not to be saved on precheck failure")
	}

	// 3. Missing repo URL returns 400
	projRepo.projects["proj-1"].Settings.ScionEnabled = true
	projRepo.projects["proj-1"].Settings.DefaultGitRepoURL = ""
	w = callCommentCreate(ch, "/implement build it")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing repo status = %d, want 400", w.Code)
	}

	// 4. /implement creates & binds impl agent and returns dispatch + dispatch_comment
	projRepo.projects["proj-1"].Settings.DefaultGitRepoURL = "https://github.com/org/repo"
	w = callCommentCreate(ch, "/implement build the feature")
	if w.Code != http.StatusCreated {
		t.Fatalf("/implement status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	var implResp CreateCommentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &implResp); err != nil {
		t.Fatalf("unmarshal /implement response: %v", err)
	}
	if implResp.SlashCommand != "implement" {
		t.Errorf("slash_command = %q, want implement", implResp.SlashCommand)
	}
	if implResp.Dispatch == nil || !implResp.Dispatch.OK || implResp.Dispatch.AgentID == "" {
		t.Fatalf("unexpected dispatch summary: %+v", implResp.Dispatch)
	}
	if implResp.DispatchComment == nil || !implResp.DispatchComment.IsSystem {
		t.Fatalf("expected system dispatch comment, got %+v", implResp.DispatchComment)
	}
	if itemRepo.items["item-1"].ImplAgentID == "" {
		t.Fatalf("expected item.ImplAgentID to be persisted")
	}

	// 5. /review creates & binds review agent
	w = callCommentCreate(ch, "/review check edge cases")
	if w.Code != http.StatusCreated {
		t.Fatalf("/review status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	if itemRepo.items["item-1"].ReviewAgentID == "" {
		t.Fatalf("expected item.ReviewAgentID to be persisted")
	}
	if len(hub.createdReqs) != 2 {
		t.Fatalf("expected 2 created agent requests, got %d", len(hub.createdReqs))
	}
	revReq := hub.createdReqs[1]
	if revReq.ProjectID != "global" || revReq.HarnessConfig != "antigravity" ||
		revReq.Config == nil || revReq.Config.Harness != "antigravity" || revReq.Config.HarnessConfig != "antigravity" {
		t.Fatalf("unexpected review CreateAgentRequest wire contract: %+v", revReq)
	}

	// 5b. Inline --agent=<slug> resolves slug via ListAgents to Hub UUID before messaging/stopping
	hub.agents["22222222-2222-4222-8222-222222222222"] = scion.Agent{
		ID:    "22222222-2222-4222-8222-222222222222",
		Slug:  "custom-slug-agent",
		Name:  "custom-slug-agent",
		Phase: "running",
	}
	w = callCommentCreate(ch, "/ask --agent=custom-slug-agent what is the status?")
	if w.Code != http.StatusCreated {
		t.Fatalf("/ask --agent=slug status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	foundUUIDMsg := false
	for _, m := range hub.messaged {
		if m == "22222222-2222-4222-8222-222222222222:"+m[len("22222222-2222-4222-8222-222222222222:"):] &&
			len(m) > len("22222222-2222-4222-8222-222222222222:") {
			foundUUIDMsg = true
			break
		}
	}
	if !foundUUIDMsg {
		t.Fatalf("expected /ask --agent=custom-slug-agent to resolve to UUID, messaged=%v", hub.messaged)
	}

	// 6. /stop --all stops both agents
	w = callCommentCreate(ch, "/stop --all")
	if w.Code != http.StatusCreated {
		t.Fatalf("/stop --all status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	if len(hub.stopped) != 2 {
		t.Fatalf("expected 2 agents stopped, got %d (%v)", len(hub.stopped), hub.stopped)
	}

	// 7. ScionHandler endpoints
	r := gin.New()
	r.GET("/projects/:projectId/scion/defaults", sh.GetDefaults)
	r.GET("/projects/:projectId/scion/agents", sh.ListAgents)
	r.GET("/projects/:projectId/items/:itemId/agents", sh.GetItemAgents)

	wDef := httptest.NewRecorder()
	r.ServeHTTP(wDef, httptest.NewRequest(http.MethodGet, "/projects/proj-1/scion/defaults", nil))
	if wDef.Code != http.StatusOK {
		t.Fatalf("GetDefaults status = %d", wDef.Code)
	}

	wAgents := httptest.NewRecorder()
	r.ServeHTTP(wAgents, httptest.NewRequest(http.MethodGet, "/projects/proj-1/scion/agents", nil))
	if wAgents.Code != http.StatusOK {
		t.Fatalf("ListAgents status = %d", wAgents.Code)
	}

	wItemAgents := httptest.NewRecorder()
	r.ServeHTTP(wItemAgents, httptest.NewRequest(http.MethodGet, "/projects/proj-1/items/item-1/agents", nil))
	if wItemAgents.Code != http.StatusOK {
		t.Fatalf("GetItemAgents status = %d", wItemAgents.Code)
	}
}
