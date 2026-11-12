package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/mcp"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

func main() {
	// Crucial: Route all logs to Stderr to prevent corrupting stdout JSON-RPC 2.0 messages in STDIO mode
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var (
		transportFlag = flag.String("transport", getEnv("MCP_TRANSPORT", "stdio"), "Transport protocol: 'stdio' or 'sse'")
		portFlag      = flag.String("port", getEnv("MCP_PORT", "8085"), "Port for SSE transport")
		projectFlag   = flag.String("project", getEnv("DEFAULT_PROJECT_ID", "proj-jynko-core"), "Default project ID fallback")
	)
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()

	slog.Info("initializing Jynko MCP Server",
		slog.String("transport", *transportFlag),
		slog.String("default_project", *projectFlag),
		slog.String("gcp_project", cfg.GoogleCloudProject),
	)

	// Live Firestore connection
	firestoreClient, err := repository.NewFirestoreClient(
		ctx,
		cfg.GoogleCloudProject,
		cfg.FirestoreDatabaseID,
		cfg.ProjectsCollection,
	)
	if err != nil {
		slog.Error("failed to connect to Firestore", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer firestoreClient.Close()

	// Repositories
	projectRepo := repository.NewProjectRepository(firestoreClient)
	moduleRepo := repository.NewModuleRepository(firestoreClient)
	epicRepo := repository.NewEpicRepository(firestoreClient)
	cycleRepo := repository.NewCycleRepository(firestoreClient)
	labelRepo := repository.NewLabelRepository(firestoreClient)
	itemRepo := repository.NewItemRepository(firestoreClient)
	pageRepo := repository.NewPageRepository(firestoreClient)
	stickyRepo := repository.NewStickyRepository(firestoreClient)

	// Services
	projectSvc := service.NewProjectService(projectRepo)
	moduleSvc := service.NewModuleService(moduleRepo)
	epicSvc := service.NewEpicService(epicRepo)
	cycleSvc := service.NewCycleService(cycleRepo)
	_ = service.NewLabelService(labelRepo)
	itemSvc := service.NewItemService(itemRepo)
	pageSvc := service.NewPageService(pageRepo)
	stickySvc := service.NewStickyService(stickyRepo)

	mcpUser := model.UserContext{
		UserID: cfg.DevUserID,
		Email:  cfg.DevUserEmail,
		Name:   cfg.DevUserName,
	}
	if mcpUser.UserID == "" {
		mcpUser.UserID = "mcp-agent"
		mcpUser.Email = "agent@jynko.local"
		mcpUser.Name = "Jynko AI Agent"
	}

	mcpCfg := mcp.Config{
		DefaultProjectID: *projectFlag,
		User:             mcpUser,
	}

	serverDeps := mcp.ServerDeps{
		Config:      mcpCfg,
		ProjectRepo: projectRepo,
		ProjectSvc:  projectSvc,
		ItemSvc:     itemSvc,
		PageSvc:     pageSvc,
		CycleSvc:    cycleSvc,
		EpicSvc:     epicSvc,
		ModuleSvc:   moduleSvc,
		StickySvc:   stickySvc,
	}

	srv := mcp.NewServer(serverDeps)

	switch *transportFlag {
	case "sse":
		slog.Info(fmt.Sprintf("starting MCP Server on SSE port :%s ...", *portFlag))
		if err := srv.ServeSSE(*portFlag); err != nil {
			slog.Error("SSE server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case "stdio":
		slog.Info("starting MCP Server in STDIO mode...")
		if err := srv.ServeStdio(); err != nil {
			slog.Error("STDIO server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	default:
		slog.Error(fmt.Sprintf("unknown transport: %s (expected 'stdio' or 'sse')", *transportFlag))
		os.Exit(1)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
