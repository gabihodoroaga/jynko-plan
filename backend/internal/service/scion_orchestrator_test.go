package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/scion"
)

type scionTestProjectRepo struct {
	projects map[string]*model.Project
}

func (r *scionTestProjectRepo) Create(_ context.Context, p *model.Project, _ *model.ProjectMember) error {
	r.projects[p.ID] = p
	return nil
}
func (r *scionTestProjectRepo) GetByID(_ context.Context, id string) (*model.Project, error) {
	p, ok := r.projects[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return p, nil
}
func (r *scionTestProjectRepo) ListByUser(_ context.Context, _, _ string) ([]model.Project, error) {
	return []model.Project{}, nil
}
func (r *scionTestProjectRepo) Update(_ context.Context, p *model.Project) error {
	r.projects[p.ID] = p
	return nil
}
func (r *scionTestProjectRepo) SetDeleted(_ context.Context, _ string, _ *time.Time, _ string) error {
	return nil
}
func (r *scionTestProjectRepo) ListDeletedBefore(_ context.Context, _ time.Time) ([]model.Project, error) {
	return []model.Project{}, nil
}
func (r *scionTestProjectRepo) Delete(_ context.Context, id string) error {
	delete(r.projects, id)
	return nil
}
func (r *scionTestProjectRepo) GetMember(_ context.Context, _, _ string) (*model.ProjectMember, error) {
	return nil, repository.ErrNotFound
}
func (r *scionTestProjectRepo) GetMemberByEmail(_ context.Context, _, _ string) (*model.ProjectMember, error) {
	return nil, repository.ErrNotFound
}
func (r *scionTestProjectRepo) ListMembers(_ context.Context, _ string) ([]model.ProjectMember, error) {
	return []model.ProjectMember{}, nil
}
func (r *scionTestProjectRepo) UpsertMember(_ context.Context, _ string, _ *model.ProjectMember) error {
	return nil
}
func (r *scionTestProjectRepo) RemoveMember(_ context.Context, _, _ string) error {
	return nil
}

type scionTestModuleRepo struct{}

func (r *scionTestModuleRepo) List(_ context.Context, _ string) ([]model.Module, error) {
	return []model.Module{}, nil
}
func (r *scionTestModuleRepo) GetByID(_ context.Context, _, _ string) (*model.Module, error) {
	return nil, repository.ErrNotFound
}
func (r *scionTestModuleRepo) Create(_ context.Context, _ *model.Module) error { return nil }
func (r *scionTestModuleRepo) Update(_ context.Context, _ *model.Module) error { return nil }
func (r *scionTestModuleRepo) Delete(_ context.Context, _, _ string) error     { return nil }

type scionTestItemRepo struct {
	items    map[string]*model.WorkItem
	comments []*model.Comment
}

func (r *scionTestItemRepo) List(_ context.Context, _ string, _ model.WorkItemFilter) ([]model.WorkItem, error) {
	return []model.WorkItem{}, nil
}
func (r *scionTestItemRepo) GetByID(_ context.Context, _, itemID string) (*model.WorkItem, error) {
	it, ok := r.items[itemID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *it
	return &cp, nil
}
func (r *scionTestItemRepo) GetByKey(_ context.Context, _, _ string) (*model.WorkItem, error) {
	return nil, repository.ErrNotFound
}
func (r *scionTestItemRepo) Create(_ context.Context, item *model.WorkItem) error {
	r.items[item.ID] = item
	return nil
}
func (r *scionTestItemRepo) Update(_ context.Context, item *model.WorkItem) error {
	cp := *item
	r.items[item.ID] = &cp
	return nil
}
func (r *scionTestItemRepo) Delete(_ context.Context, _, itemID string) error {
	delete(r.items, itemID)
	return nil
}
func (r *scionTestItemRepo) ListSubItems(_ context.Context, _, _ string) ([]model.SubWorkItem, error) {
	return []model.SubWorkItem{}, nil
}
func (r *scionTestItemRepo) GetSubItemByID(_ context.Context, _, _, _ string) (*model.SubWorkItem, error) {
	return nil, repository.ErrNotFound
}
func (r *scionTestItemRepo) CreateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (r *scionTestItemRepo) UpdateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (r *scionTestItemRepo) DeleteSubItem(_ context.Context, _, _, _ string) error { return nil }
func (r *scionTestItemRepo) ListComments(_ context.Context, _, _ string) ([]model.Comment, error) {
	return []model.Comment{}, nil
}
func (r *scionTestItemRepo) GetCommentByID(_ context.Context, _, _, _ string) (*model.Comment, error) {
	return nil, repository.ErrNotFound
}
func (r *scionTestItemRepo) CreateComment(_ context.Context, _ string, c *model.Comment) error {
	c.ID = "sys-1"
	cp := *c
	r.comments = append(r.comments, &cp)
	return nil
}
func (r *scionTestItemRepo) UpdateComment(_ context.Context, _, _, _, _ string) (*model.Comment, error) {
	return nil, nil
}
func (r *scionTestItemRepo) DeleteComment(_ context.Context, _, _, _ string) error { return nil }

type scionTestHubClient struct {
	agents      map[string]scion.Agent
	createdReqs []scion.CreateAgentRequest
	messagedIDs []string
}

func (c *scionTestHubClient) ListAgents(_ context.Context, _ string) ([]scion.Agent, error) {
	out := make([]scion.Agent, 0, len(c.agents))
	for _, a := range c.agents {
		out = append(out, a)
	}
	return out, nil
}
func (c *scionTestHubClient) GetAgent(_ context.Context, agentID string) (*scion.Agent, error) {
	a, ok := c.agents[agentID]
	if !ok {
		return nil, scion.ErrAgentNotFound
	}
	cp := a
	return &cp, nil
}
func (c *scionTestHubClient) CreateAgent(_ context.Context, req scion.CreateAgentRequest) (*scion.Agent, error) {
	c.createdReqs = append(c.createdReqs, req)
	a := scion.Agent{
		ID:            "33333333-3333-4333-8333-333333333333",
		Slug:          req.Name,
		Name:          req.Name,
		HarnessConfig: req.HarnessConfig,
		Phase:         "running",
	}
	c.agents[a.ID] = a
	return &a, nil
}
func (c *scionTestHubClient) StartAgent(_ context.Context, agentID string) (*scion.Agent, error) {
	a, ok := c.agents[agentID]
	if !ok {
		return nil, scion.ErrAgentNotFound
	}
	return &a, nil
}
func (c *scionTestHubClient) StopAgent(_ context.Context, _ string) error { return nil }
func (c *scionTestHubClient) SendMessage(_ context.Context, agentID, _ string, _ bool) error {
	if _, ok := c.agents[agentID]; !ok {
		return scion.ErrAgentNotFound
	}
	c.messagedIDs = append(c.messagedIDs, agentID)
	return nil
}
func (c *scionTestHubClient) EnsureRunningAndSendMessage(_ context.Context, agentID, _ string) error {
	if _, ok := c.agents[agentID]; !ok {
		return scion.ErrAgentNotFound
	}
	c.messagedIDs = append(c.messagedIDs, agentID)
	return nil
}

func newScionOrchestratorTestFixture() (*ScionOrchestrator, *scionTestItemRepo, *scionTestHubClient) {
	cfg := &config.Config{
		ScionHubURL:             "https://scion.example.com",
		ScionHubToken:           "scion_pat_test",
		ScionDefaultProjectSlug: "global",
		ScionDefaultTemplate:    "jynko-dev",
	}
	projRepo := &scionTestProjectRepo{
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
	itemRepo := &scionTestItemRepo{
		items: map[string]*model.WorkItem{
			"item-1": {
				ID:        "item-1",
				ProjectID: "proj-1",
				Key:       "JYN-99",
				Title:     "Test explicit target",
			},
		},
	}
	hub := &scionTestHubClient{
		agents: map[string]scion.Agent{
			"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa": {
				ID:    "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
				Slug:  "claude",
				Name:  "claude",
				Phase: "running",
			},
		},
	}
	orch := NewScionOrchestrator(
		cfg,
		hub,
		NewProjectService(projRepo),
		NewModuleService(&scionTestModuleRepo{}),
		NewItemService(itemRepo),
	)
	return orch, itemRepo, hub
}

func TestScionOrchestrator_ExplicitTargetResolutionAndStrictness(t *testing.T) {
	ctx := context.Background()
	user := model.UserContext{UserID: "u-1", Name: "Alice", Email: "alice@example.com"}

	// 1. /implement --agent=global--claude strips "global--" prefix and messages claude UUID without CreateAgent
	orch, itemRepo, hub := newScionOrchestratorTestFixture()
	parsed := scion.ParseSlashCommand("/implement --agent=global--claude fix the bug")
	pre, err := orch.Precheck(ctx, "proj-1", "item-1", parsed)
	if err != nil {
		t.Fatalf("Precheck failed: %v", err)
	}
	cmt, summary := orch.Execute(ctx, pre, user, "")
	if !summary.OK || summary.Action != "messaged" {
		t.Fatalf("expected OK messaged summary, got %+v (comment=%+v)", summary, cmt)
	}
	if len(hub.createdReqs) != 0 {
		t.Fatalf("expected 0 CreateAgent calls for --agent=global--claude, got %d", len(hub.createdReqs))
	}
	if len(hub.messagedIDs) != 1 || hub.messagedIDs[0] != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
		t.Fatalf("expected claude UUID messaged, got %v", hub.messagedIDs)
	}

	// 2. /implement --agent=nonexistent-agent must NOT silently create a new agent; records failure in dispatch comment
	orch, itemRepo, hub = newScionOrchestratorTestFixture()
	parsed = scion.ParseSlashCommand("/implement --agent=nonexistent-agent fix the bug")
	pre, err = orch.Precheck(ctx, "proj-1", "item-1", parsed)
	if err != nil {
		t.Fatalf("Precheck failed: %v", err)
	}
	cmt, summary = orch.Execute(ctx, pre, user, "")
	if summary.OK {
		t.Fatalf("expected DispatchSummary.OK == false for missing explicit agent, got %+v", summary)
	}
	if len(hub.createdReqs) != 0 {
		t.Fatalf("expected 0 CreateAgent calls for missing explicit agent, got %d", len(hub.createdReqs))
	}
	if cmt == nil || !cmt.IsSystem || !strings.Contains(cmt.Body, "not found") {
		t.Fatalf("expected failure system dispatch comment, got %+v", cmt)
	}

	// 3. Previously bound item.ImplAgentID missing on Hub WITHOUT --agent (StrictTarget == false) still recreates
	orch, itemRepo, hub = newScionOrchestratorTestFixture()
	itemRepo.items["item-1"].ImplAgentID = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	parsed = scion.ParseSlashCommand("/implement continue work")
	pre, err = orch.Precheck(ctx, "proj-1", "item-1", parsed)
	if err != nil {
		t.Fatalf("Precheck failed: %v", err)
	}
	_, summary = orch.Execute(ctx, pre, user, "")
	if !summary.OK || summary.Action != "created" {
		t.Fatalf("expected fallback CreateAgent when bound ImplAgentID is missing, got %+v", summary)
	}
	if len(hub.createdReqs) != 1 {
		t.Fatalf("expected 1 CreateAgent fallback call, got %d", len(hub.createdReqs))
	}
}
