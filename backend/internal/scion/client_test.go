package scion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_AuthOIDCCachingAndLifecycle(t *testing.T) {
	// Fake GCE metadata server returning an OIDC JWT expiring in 1 hour
	var metaCalls atomic.Int32
	exp := time.Now().Add(1 * time.Hour).Unix()
	claimsJSON, _ := json.Marshal(map[string]any{"exp": exp})
	fakeJWT := "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(claimsJSON) + ".sig"

	metaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		metaCalls.Add(1)
		_, _ = w.Write([]byte(fakeJWT))
	}))
	defer metaSrv.Close()

	metaHost := strings.TrimPrefix(metaSrv.URL, "http://")
	_ = os.Setenv("GCE_METADATA_HOST", metaHost)
	defer func() { _ = os.Unsetenv("GCE_METADATA_HOST") }()

	var phaseCalls atomic.Int32
	var startCalls atomic.Int32
	var messageCalls atomic.Int32

	hubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer scion_pat_test" ||
			r.Header.Get("X-Scion-Authorization") != "Bearer scion_pat_test" ||
			r.Header.Get("Proxy-Authorization") != "Bearer "+fakeJWT {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agents/agent-1":
			n := phaseCalls.Add(1)
			phase := "stopped"
			if n >= 2 {
				phase = "running"
			}
			_ = json.NewEncoder(w).Encode(Agent{ID: "agent-1", Name: "impl-hodo-1", Phase: phase})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agents/agent-1/start":
			startCalls.Add(1)
			_ = json.NewEncoder(w).Encode(Agent{ID: "agent-1", Name: "impl-hodo-1", Phase: "starting"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agents/agent-1/message":
			messageCalls.Add(1)
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["wake"] != true {
				t.Errorf("expected wake=true in message payload")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"sent"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agents":
			var req CreateAgentRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if !req.AcceptAsyncLaunch {
				t.Errorf("expected acceptAsyncLaunch=true")
			}
			if req.Slug == "dup-slug" {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"slug exists"}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Agent{ID: "agent-new", Slug: req.Slug, Name: req.Name, Phase: "running"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer hubSrv.Close()

	client := NewClient(ClientConfig{
		BaseURL:      hubSrv.URL,
		Token:        "scion_pat_test",
		IapAudience:  "aud-123",
		PollInterval: 10 * time.Millisecond,
		StartTimeout: 500 * time.Millisecond,
	})

	// 1. Stopped agent must be started and polled until running before SendMessage
	if err := client.EnsureRunningAndSendMessage(context.Background(), "agent-1", "Please fix tests"); err != nil {
		t.Fatalf("EnsureRunningAndSendMessage failed: %v", err)
	}
	if startCalls.Load() != 1 {
		t.Errorf("expected 1 StartAgent call, got %d", startCalls.Load())
	}
	if messageCalls.Load() != 1 {
		t.Errorf("expected 1 SendMessage call, got %d", messageCalls.Load())
	}

	// 2. 409 Conflict on CreateAgent retries with unique slug suffix
	created, err := client.CreateAgent(context.Background(), CreateAgentRequest{
		Name: "dup-slug",
		Slug: "dup-slug",
	})
	if err != nil {
		t.Fatalf("CreateAgent retry failed: %v", err)
	}
	if created.ID != "agent-new" || created.Slug == "dup-slug" {
		t.Errorf("expected retried unique slug, got %+v", created)
	}

	// 3. OIDC token must be cached across all requests (only 1 metadata call)
	if metaCalls.Load() != 1 {
		t.Errorf("expected OIDC token cached (1 metadata call), got %d", metaCalls.Load())
	}
}
