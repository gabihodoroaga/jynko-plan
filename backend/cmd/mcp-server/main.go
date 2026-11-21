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
	"github.com/gabihodoroaga/jynko-plan/internal/storage"
)

func main() {
	// Crucial: Route all logs to Stderr to prevent corrupting stdout JSON-RPC 2.0 messages in STDIO mode
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var (
		transportFlag = flag.String("transport", getEnv("MCP_TRANSPORT", "stdio"), "Transport protocol: 'stdio' or 'sse'")
		hostFlag      = flag.String("host", getEnv("MCP_HOST", ""), "Bind host for SSE transport (defaults to 127.0.0.1 in dev, 0.0.0.0 in prd/stg)")
		portFlag      = flag.String("port", getEnv("MCP_PORT", "8085"), "Port for SSE transport")
		projectFlag   = flag.String("project", getEnv("DEFAULT_PROJECT_ID", "proj-jynko-core"), "Default project ID fallback")
	)
	flag.Parse()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid server configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}
	ctx := context.Background()

	slog.Info("initializing Jynko MCP Server",
		slog.String("transport", *transportFlag),
		slog.String("env", cfg.Env),
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

	// Storage Provider (local filesystem for dev/e2e, GCS for prd/stg)
	storageProvider, err := storage.NewStorageProvider(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize storage provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer storageProvider.Close()

	// Repositories
	projectRepo := repository.NewProjectRepository(firestoreClient)
	moduleRepo := repository.NewModuleRepository(firestoreClient)
	epicRepo := repository.NewEpicRepository(firestoreClient)
	cycleRepo := repository.NewCycleRepository(firestoreClient)
	labelRepo := repository.NewLabelRepository(firestoreClient)
	itemRepo := repository.NewItemRepository(firestoreClient)
	pageRepo := repository.NewPageRepository(firestoreClient)
	stickyRepo := repository.NewStickyRepository(firestoreClient)
	attachmentRepo := repository.NewAttachmentRepository(firestoreClient)

	// Services
	projectSvc := service.NewProjectService(projectRepo)
	moduleSvc := service.NewModuleService(moduleRepo)
	epicSvc := service.NewEpicService(epicRepo)
	cycleSvc := service.NewCycleService(cycleRepo)
	_ = service.NewLabelService(labelRepo)
	itemSvc := service.NewItemService(itemRepo)
	pageSvc := service.NewPageService(pageRepo)
	stickySvc := service.NewStickyService(stickyRepo)
	attachmentSvc := service.NewAttachmentService(attachmentRepo, storageProvider)

	// Static fallback user is ONLY enabled for local STDIO transport or dev environment.
	// In prd/stg SSE mode, every incoming HTTP request must carry authenticated IAP/OIDC credentials.
	var mcpUser model.UserContext
	if *transportFlag == "stdio" || cfg.AllowDevAuth() {
		mcpUser = model.UserContext{
			UserID: cfg.DevUserID,
			Email:  cfg.DevUserEmail,
			Name:   cfg.DevUserName,
		}
		if mcpUser.UserID == "" && *transportFlag == "stdio" {
			mcpUser.UserID = "mcp-agent"
			mcpUser.Email = "agent@jynko.local"
			mcpUser.Name = "Jynko AI Agent"
		}
	}

	mcpCfg := mcp.Config{
		DefaultProjectID: *projectFlag,
		User:             mcpUser,
	}

	serverDeps := mcp.ServerDeps{
		Config:        mcpCfg,
		AppConfig:     cfg,
		ProjectRepo:   projectRepo,
		ProjectSvc:    projectSvc,
		ItemSvc:       itemSvc,
		PageSvc:       pageSvc,
		CycleSvc:      cycleSvc,
		EpicSvc:       epicSvc,
		ModuleSvc:     moduleSvc,
		StickySvc:     stickySvc,
		AttachmentSvc: attachmentSvc,
	}

	srv := mcp.NewServer(serverDeps)

	switch *transportFlag {
	case "sse":
		addr := *portFlag
		if *hostFlag != "" {
			addr = fmt.Sprintf("%s:%s", *hostFlag, *portFlag)
		}
		slog.Info(fmt.Sprintf("starting MCP Server on SSE %s ...", addr))
		if err := srv.ServeSSE(addr); err != nil {
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
