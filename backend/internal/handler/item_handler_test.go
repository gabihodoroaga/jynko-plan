package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// keyedFakeItemRepo adds human-key lookups on top of fakeItemRepo.
type keyedFakeItemRepo struct {
	fakeItemRepo
}

func (f *keyedFakeItemRepo) GetByKey(_ context.Context, _, key string) (*model.WorkItem, error) {
	for _, it := range f.items {
		if it.Key == key {
			cp := *it
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}

// TestGetItem_IDKeyBlankAndMissing covers PLAN-12: GET /items/:itemId accepts an internal ID or a
// human key, returns 404 for unknown items, and 400 (not 500) for a blank/whitespace-only ID.
func TestGetItem_IDKeyBlankAndMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &keyedFakeItemRepo{fakeItemRepo{items: map[string]*model.WorkItem{
		"item-1": {ID: "item-1", ProjectID: "proj-1", Key: "PLAN-12", Title: "Improve refresh"},
	}}}
	h := NewItemHandler(service.NewItemService(repo))
	r := gin.New()
	r.GET("/projects/:projectId/items/:itemId", h.GetItem)

	cases := []struct {
		name   string
		path   string
		status int
		wantID string
	}{
		{"by internal id", "/projects/proj-1/items/item-1", http.StatusOK, "item-1"},
		{"by human key", "/projects/proj-1/items/PLAN-12", http.StatusOK, "item-1"},
		{"unknown item", "/projects/proj-1/items/PLAN-999", http.StatusNotFound, ""},
		{"whitespace-only id", "/projects/proj-1/items/%20%20", http.StatusBadRequest, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("GET %s status = %d, want %d (body=%s)", tc.path, w.Code, tc.status, w.Body.String())
			}
			if tc.wantID != "" {
				var got model.WorkItem
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.ID != tc.wantID {
					t.Fatalf("GET %s returned item %+v (err=%v), want id %q", tc.path, got, err, tc.wantID)
				}
			}
		})
	}
}
