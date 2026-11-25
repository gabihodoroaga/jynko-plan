// Package testutil provides in-memory fakes shared by unit tests. It must only be imported from tests.
package testutil

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// ProjectRepo is an in-memory repository.ProjectRepository with soft-delete support (PLAN-13).
type ProjectRepo struct {
	mu       sync.Mutex
	Projects map[string]*model.Project
	// Members is keyed by projectID then userID.
	Members map[string]map[string]*model.ProjectMember
	// DeleteErr, when set for a project ID, makes the hard Delete fail for that project.
	DeleteErr map[string]error
	// HardDeleted records project IDs removed by Delete.
	HardDeleted []string
}

// NewProjectRepo returns an empty in-memory project repository.
func NewProjectRepo() *ProjectRepo {
	return &ProjectRepo{
		Projects:  map[string]*model.Project{},
		Members:   map[string]map[string]*model.ProjectMember{},
		DeleteErr: map[string]error{},
	}
}

// AddProject stores a project with the given members (all users are added to MemberIDs).
func (r *ProjectRepo) AddProject(p model.Project, members ...model.ProjectMember) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := p
	if r.Members[p.ID] == nil {
		r.Members[p.ID] = map[string]*model.ProjectMember{}
	}
	for i := range members {
		m := members[i]
		r.Members[p.ID][m.UserID] = &m
		cp.MemberIDs = append(cp.MemberIDs, m.UserID)
	}
	r.Projects[p.ID] = &cp
}

func (r *ProjectRepo) Create(_ context.Context, project *model.Project, initialAdmin *model.ProjectMember) error {
	if initialAdmin != nil {
		r.AddProject(*project, *initialAdmin)
	} else {
		r.AddProject(*project)
	}
	return nil
}

func (r *ProjectRepo) GetByID(_ context.Context, projectID string) (*model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.Projects[projectID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *ProjectRepo) ListByUser(_ context.Context, userID, userEmail string) ([]model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []model.Project{}
	email := strings.ToLower(strings.TrimSpace(userEmail))
	for _, p := range r.Projects {
		for _, id := range p.MemberIDs {
			if id == userID || (email != "" && id == email) {
				out = append(out, *p)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *ProjectRepo) Update(_ context.Context, project *model.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.Projects[project.ID]; !ok {
		return repository.ErrNotFound
	}
	cp := *project
	r.Projects[project.ID] = &cp
	return nil
}

func (r *ProjectRepo) Delete(_ context.Context, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.DeleteErr[projectID]; err != nil {
		return err
	}
	if _, ok := r.Projects[projectID]; !ok {
		return repository.ErrNotFound
	}
	delete(r.Projects, projectID)
	delete(r.Members, projectID)
	r.HardDeleted = append(r.HardDeleted, projectID)
	return nil
}

func (r *ProjectRepo) SetDeleted(_ context.Context, projectID string, deletedAt *time.Time, deletedBy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.Projects[projectID]
	if !ok {
		return repository.ErrNotFound
	}
	if deletedAt != nil {
		at := deletedAt.UTC()
		p.DeletedAt = &at
		p.DeletedBy = deletedBy
	} else {
		p.DeletedAt = nil
		p.DeletedBy = ""
	}
	return nil
}

func (r *ProjectRepo) ListDeletedBefore(_ context.Context, cutoff time.Time) ([]model.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []model.Project{}
	for _, p := range r.Projects {
		if p.DeletedAt != nil && !p.DeletedAt.After(cutoff) {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *ProjectRepo) GetMember(_ context.Context, projectID, userID string) (*model.ProjectMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.Members[projectID][userID]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}

func (r *ProjectRepo) GetMemberByEmail(_ context.Context, projectID, email string) (*model.ProjectMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	normalized := strings.ToLower(strings.TrimSpace(email))
	for _, m := range r.Members[projectID] {
		if normalized != "" && strings.ToLower(m.Email) == normalized {
			cp := *m
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *ProjectRepo) ListMembers(_ context.Context, projectID string) ([]model.ProjectMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []model.ProjectMember{}
	for _, m := range r.Members[projectID] {
		out = append(out, *m)
	}
	return out, nil
}

func (r *ProjectRepo) UpsertMember(_ context.Context, projectID string, member *model.ProjectMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Members[projectID] == nil {
		r.Members[projectID] = map[string]*model.ProjectMember{}
	}
	cp := *member
	r.Members[projectID][member.UserID] = &cp
	return nil
}

func (r *ProjectRepo) RemoveMember(_ context.Context, projectID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.Members[projectID][userID]; !ok {
		return repository.ErrNotFound
	}
	delete(r.Members[projectID], userID)
	return nil
}

// ErrInjected is a convenience error for simulating repository failures.
var ErrInjected = errors.New("injected failure")

var _ repository.ProjectRepository = (*ProjectRepo)(nil)
