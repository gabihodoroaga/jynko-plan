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
			var rawReq map[string]any
			_ = json.NewDecoder(r.Body).Decode(&rawReq)
			pid, _ := rawReq["projectId"].(string)
			if strings.TrimSpace(pid) == "" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"code":"validation_error","message":"projectId is required"}}`))
				return
			}
			if _, hasLegacy := rawReq["project_slug"]; hasLegacy {
				t.Errorf("did not expect legacy project_slug key in CreateAgent payload: %+v", rawReq)
			}
			if rawReq["acceptAsyncLaunch"] != true {
				t.Errorf("expected acceptAsyncLaunch=true")
			}
			if rawReq["harnessConfig"] != "antigravity" {
				t.Errorf("expected top-level harnessConfig=antigravity, got %v", rawReq["harnessConfig"])
			}
			cfgMap, _ := rawReq["config"].(map[string]any)
			if cfgMap["harness"] != "antigravity" || cfgMap["harness_config"] != "antigravity" {
				t.Errorf("expected config.harness and config.harness_config = antigravity, got %+v", cfgMap)
			}
			envMap, _ := cfgMap["env"].(map[string]any)
			if envMap["DEFAULT_PROJECT_ID"] != "proj-1" {
				t.Errorf("expected config.env.DEFAULT_PROJECT_ID=proj-1, got %+v", envMap)
			}
			name, _ := rawReq["name"].(string)
			if name == "dup-name" {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"slug exists"}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Agent{ID: "agent-new", Slug: name, Name: name, ProjectID: pid, HarnessConfig: "antigravity", Phase: "running"})
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

	// 2. 409 Conflict on CreateAgent retries with unique name suffix and sends wire contract fields
	created, err := client.CreateAgent(context.Background(), CreateAgentRequest{
		Name:          "dup-name",
		ProjectID:     "global",
		Template:      "jynko-dev",
		HarnessConfig: "antigravity",
		Config: &AgentConfig{
			Harness:       "antigravity",
			HarnessConfig: "antigravity",
			Env:           map[string]string{"DEFAULT_PROJECT_ID": "proj-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateAgent retry failed: %v", err)
	}
	if created.ID != "agent-new" || created.Name == "dup-name" || created.Harness != "antigravity" {
		t.Errorf("expected retried unique name and normalized harness, got %+v", created)
	}

	// 3. OIDC token must be cached across all requests (only 1 metadata call)
	if metaCalls.Load() != 1 {
		t.Errorf("expected OIDC token cached (1 metadata call), got %d", metaCalls.Load())
	}
}

func TestClient_ListAgentsAndStartingPhase(t *testing.T) {
	var startCalls atomic.Int32
	var getCalls atomic.Int32
	var msgCalls atomic.Int32

	hubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agents":
			if got := r.URL.Query().Get("projectId"); got != "global" {
				t.Errorf("expected query param projectId=global, got %q (raw=%q)", got, r.URL.RawQuery)
			}
			if r.URL.Query().Has("project_slug") {
				t.Errorf("did not expect legacy project_slug query param")
			}
			_, _ = w.Write([]byte(`{"agents":[{"id":"uuid-1","slug":"rev-jyn-1","name":"rev-jyn-1","projectId":"global","harnessConfig":"antigravity","phase":"running"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agents/uuid-2":
			n := getCalls.Add(1)
			phase := "starting"
			if n == 2 {
				phase = "provisioning"
			} else if n >= 3 {
				phase = "running"
			}
			_ = json.NewEncoder(w).Encode(Agent{ID: "uuid-2", Name: "impl-jyn-2", Phase: phase})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agents/uuid-2/start":
			startCalls.Add(1)
			w.WriteHeader(http.StatusConflict)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agents/uuid-2/message":
			msgCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer hubSrv.Close()

	client := NewClient(ClientConfig{
		BaseURL:      hubSrv.URL,
		Token:        "scion_pat_test",
		PollInterval: 5 * time.Millisecond,
		StartTimeout: 500 * time.Millisecond,
	})

	agents, err := client.ListAgents(context.Background(), "global")
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != "uuid-1" || agents[0].Harness != "antigravity" {
		t.Fatalf("unexpected ListAgents result: %+v", agents)
	}

	// Agent already in starting/provisioning must NOT call /start; must poll until running then /message
	if err := client.EnsureRunningAndSendMessage(context.Background(), "uuid-2", "Continue work"); err != nil {
		t.Fatalf("EnsureRunningAndSendMessage on starting agent failed: %v", err)
	}
	if startCalls.Load() != 0 {
		t.Errorf("expected 0 StartAgent calls for starting/provisioning agent, got %d", startCalls.Load())
	}
	if msgCalls.Load() != 1 {
		t.Errorf("expected 1 SendMessage call, got %d", msgCalls.Load())
	}
}
