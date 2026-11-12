package mcp

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"

	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// ServerDeps groups all domain services and repositories required by the MCP server.
type ServerDeps struct {
	Config      Config
	ProjectRepo repository.ProjectRepository
	ProjectSvc  *service.ProjectService
	ItemSvc     *service.ItemService
	PageSvc     *service.PageService
	CycleSvc    *service.CycleService
	EpicSvc     *service.EpicService
	ModuleSvc   *service.ModuleService
	StickySvc   *service.StickyService
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

// ServeSSE starts the MCP server over streamable HTTP / SSE.
func (s *Server) ServeSSE(port string) error {
	if port == "" {
		port = "8085"
	}
	httpServer := server.NewStreamableHTTPServer(s.mcpServer)
	return httpServer.Start(fmt.Sprintf(":%s", port))
}
