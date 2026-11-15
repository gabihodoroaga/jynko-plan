package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// ServerDeps groups all domain services and repositories required by the MCP server.
type ServerDeps struct {
	Config       Config
	AppConfig    *config.Config
	JWTValidator middleware.JWTValidatorFunc
	ProjectRepo  repository.ProjectRepository
	ProjectSvc   *service.ProjectService
	ItemSvc      *service.ItemService
	PageSvc      *service.PageService
	CycleSvc     *service.CycleService
	EpicSvc      *service.EpicService
	ModuleSvc    *service.ModuleService
	StickySvc    *service.StickyService
}

// Server wraps the mark3labs MCP Server with Jynko domain tool, resource, and prompt registrations.
type Server struct {
	mcpServer *server.MCPServer
	deps      ServerDeps
	resolver  *Resolver
}

// NewServer initializes and configures a Jynko MCP Server instance.
func NewServer(deps ServerDeps) *Server {
	mcpServer := server.NewMCPServer(
		"jynko",
		"0.1.0",
		server.WithTitle("Jynko Project Management & Wiki MCP Server"),
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false),
		server.WithPromptCapabilities(false),
	)

	resolver := NewResolver(deps.Config, deps.ProjectRepo)

	registerWorkItemTools(mcpServer, resolver, deps.ItemSvc)
	registerWikiTools(mcpServer, resolver, deps.PageSvc)
	registerProjectTools(mcpServer, resolver, deps.ProjectSvc)
	registerPlanningTools(mcpServer, resolver, deps.CycleSvc, deps.EpicSvc, deps.ModuleSvc, deps.StickySvc)
	registerResources(mcpServer, resolver, deps.ItemSvc, deps.PageSvc)
	registerPrompts(mcpServer, resolver, deps.ItemSvc, deps.CycleSvc)

	return &Server{
		mcpServer: mcpServer,
		deps:      deps,
		resolver:  resolver,
	}
}

// MCPServer returns the underlying mark3labs MCPServer.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// ServeStdio starts the MCP server over standard input/output (ideal for CLI and desktop clients).
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// HTTPHandler returns an authenticated http.Handler for Streamable HTTP / SSE MCP traffic.
func (s *Server) HTTPHandler() http.Handler {
	streamable := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithMaxRequestBodyBytes(2<<20),
	)

	appCfg := s.deps.AppConfig
	if appCfg == nil {
		// Default to fail-closed prd config when AppConfig is omitted unless fallback User is explicitly configured for local testing
		appCfg = &config.Config{Env: config.EnvPrd, RawEnv: config.EnvPrd}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}

		// Validate Origin header if present to prevent cross-origin browser / DNS-rebinding attacks
		if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
			if !isAllowedOrigin(origin, appCfg.CORSAllowedOrigins) {
				writeJSONError(w, http.StatusForbidden, "origin not allowed")
				return
			}
		}

		userCtx, status, errMsg := middleware.AuthenticateHTTPRequest(r.Context(), r.Header, appCfg, s.deps.JWTValidator)
		if status != http.StatusOK {
			writeJSONError(w, status, errMsg)
			return
		}

		ctx := WithUserContext(r.Context(), userCtx)
		streamable.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ServeSSE starts the MCP server over streamable HTTP / SSE with authentication and safe timeouts.
func (s *Server) ServeSSE(addrOrPort string) error {
	addr := strings.TrimSpace(addrOrPort)
	if addr == "" {
		addr = "8085"
	}
	if !strings.Contains(addr, ":") {
		if s.deps.AppConfig == nil || s.deps.AppConfig.AllowDevAuth() {
			addr = fmt.Sprintf("127.0.0.1:%s", addr)
		} else {
			addr = fmt.Sprintf(":%s", addr)
		}
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           s.HTTPHandler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return httpSrv.ListenAndServe()
}

func isAllowedOrigin(origin string, allowed []string) bool {
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if a == "*" || strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
