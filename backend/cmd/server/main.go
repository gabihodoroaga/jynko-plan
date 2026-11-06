package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/handler"
	"github.com/gabihodoroaga/jynko-plan/internal/logger"
	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
	"github.com/gabihodoroaga/jynko-plan/internal/telemetry"
)

func main() {
	cfg := config.Load()

	// Initialize structured logger configured for GCP Cloud Logging
	appLogger := logger.New(logger.Config{
		Format:    cfg.LogFormat,
		Level:     logger.ParseLevel(cfg.LogLevel),
		ProjectID: cfg.GoogleCloudProject,
		AddSource: cfg.IsProduction(),
	})
	slog.SetDefault(appLogger)

	ctx := context.Background()

	// Initialize OpenTelemetry Tracing and Metrics pipelines
	shutdownTel, err := telemetry.Init(ctx, cfg)
	if err != nil {
		slog.Warn("failed to initialize OpenTelemetry", slog.String("error", err.Error()))
	} else {
		defer func() {
			telCtx, telCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer telCancel()
			if err := shutdownTel(telCtx); err != nil {
				slog.Warn("error shutting down OpenTelemetry", slog.String("error", err.Error()))
			}
		}()
	}

	firestoreClient, err := repository.NewFirestoreClient(ctx, cfg.GoogleCloudProject, cfg.FirestoreDatabaseID)
	if err != nil {
		slog.Error("failed to initialize Firestore client", slog.String("error", err.Error()), slog.String("gcp_project", cfg.GoogleCloudProject))
		os.Exit(1)
	}
	defer func() {
		_ = firestoreClient.Close()
	}()

	if err := repository.EnsureSeedData(ctx, firestoreClient); err != nil {
		slog.Warn("warning: failed to ensure Firestore seed data", slog.String("error", err.Error()))
	}

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
	labelSvc := service.NewLabelService(labelRepo)
	itemSvc := service.NewItemService(itemRepo)
	pageSvc := service.NewPageService(pageRepo)
	stickySvc := service.NewStickyService(stickyRepo)

	// Handlers
	authHandler := handler.NewAuthHandler()
	projectHandler := handler.NewProjectHandler(projectSvc)
	moduleHandler := handler.NewModuleHandler(moduleSvc)
	epicHandler := handler.NewEpicHandler(epicSvc)
	cycleHandler := handler.NewCycleHandler(cycleSvc)
	labelHandler := handler.NewLabelHandler(labelSvc)
	itemHandler := handler.NewItemHandler(itemSvc)
	commentHandler := handler.NewCommentHandler(itemSvc)
	pageHandler := handler.NewPageHandler(pageSvc)
	stickyHandler := handler.NewStickyHandler(stickySvc)

	router := SetupRouter(
		cfg,
		projectRepo,
		authHandler,
		projectHandler,
		moduleHandler,
		epicHandler,
		cycleHandler,
		labelHandler,
		itemHandler,
		commentHandler,
		pageHandler,
		stickyHandler,
	)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		slog.Info("Jynko server listening",
			slog.String("port", cfg.Port),
			slog.String("env", cfg.Env),
			slog.String("gcp_project", cfg.GoogleCloudProject),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// SetupRouter configures all Gin routes and middleware under /api/v1.
func SetupRouter(
	cfg *config.Config,
	projectRepo repository.ProjectRepository,
	authHandler *handler.AuthHandler,
	projectHandler *handler.ProjectHandler,
	moduleHandler *handler.ModuleHandler,
	epicHandler *handler.EpicHandler,
	cycleHandler *handler.CycleHandler,
	labelHandler *handler.LabelHandler,
	itemHandler *handler.ItemHandler,
	commentHandler *handler.CommentHandler,
	pageHandler *handler.PageHandler,
	stickyHandler *handler.StickyHandler,
) *gin.Engine {
	r := gin.New()
	r.Use(
		telemetry.GinTracingMiddleware("jynko"),
		telemetry.HTTPMetricsMiddleware(),
		middleware.StructuredLogger(cfg.GoogleCloudProject),
		middleware.StructuredRecovery(cfg.GoogleCloudProject),
		middleware.CORSMiddleware(cfg.CORSAllowedOrigins),
	)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if cfg.EnablePrometheusMetrics {
		r.GET("/metrics", gin.WrapH(telemetry.PrometheusHandler()))
	}

	api := r.Group("/api/v1")
	api.Use(middleware.IAPAuthMiddleware(cfg))

	// Auth identity & profile endpoints
	api.GET("/auth/me", authHandler.GetMe)
	api.PATCH("/auth/me", authHandler.UpdateMe)

	// Top-level Projects
	api.GET("/projects", projectHandler.ListProjects)
	api.POST("/projects", projectHandler.CreateProject)

	// Project-scoped routes with RBAC middleware
	projGroup := api.Group("/projects/:projectId")
	{
		viewer := middleware.RequireProjectRole(projectRepo, model.RoleViewer)
		member := middleware.RequireProjectRole(projectRepo, model.RoleMember)
		admin := middleware.RequireProjectRole(projectRepo, model.RoleAdmin)

		// Project Details & Settings
		projGroup.GET("", viewer, projectHandler.GetProject)
		projGroup.PATCH("", admin, projectHandler.UpdateProject)
		projGroup.DELETE("", admin, projectHandler.DeleteProject)

		// RBAC Members
		projGroup.GET("/members", viewer, projectHandler.ListMembers)
		projGroup.POST("/members", admin, projectHandler.UpsertMember)
		projGroup.DELETE("/members/:userId", admin, projectHandler.RemoveMember)

		// Modules
		projGroup.GET("/modules", viewer, moduleHandler.ListModules)
		projGroup.POST("/modules", member, moduleHandler.CreateModule)
		projGroup.PATCH("/modules/:moduleId", member, moduleHandler.UpdateModule)
		projGroup.DELETE("/modules/:moduleId", member, moduleHandler.DeleteModule)

		// Epics
		projGroup.GET("/epics", viewer, epicHandler.ListEpics)
		projGroup.POST("/epics", member, epicHandler.CreateEpic)
		projGroup.PATCH("/epics/:epicId", member, epicHandler.UpdateEpic)
		projGroup.DELETE("/epics/:epicId", member, epicHandler.DeleteEpic)

		// Cycles
		projGroup.GET("/cycles", viewer, cycleHandler.ListCycles)
		projGroup.POST("/cycles", member, cycleHandler.CreateCycle)
		projGroup.PATCH("/cycles/:cycleId", member, cycleHandler.UpdateCycle)
		projGroup.DELETE("/cycles/:cycleId", member, cycleHandler.DeleteCycle)

		// Labels
		projGroup.GET("/labels", viewer, labelHandler.ListLabels)
		projGroup.POST("/labels", member, labelHandler.CreateLabel)
		projGroup.PATCH("/labels/:labelId", member, labelHandler.UpdateLabel)
		projGroup.DELETE("/labels/:labelId", member, labelHandler.DeleteLabel)

		// Work Items
		projGroup.GET("/items", viewer, itemHandler.ListItems)
		projGroup.POST("/items", member, itemHandler.CreateItem)
		projGroup.GET("/items/:itemId", viewer, itemHandler.GetItem)
		projGroup.PATCH("/items/:itemId", member, itemHandler.UpdateItem)
		projGroup.DELETE("/items/:itemId", member, itemHandler.DeleteItem)

		// Sub-Items
		projGroup.GET("/items/:itemId/sub-items", viewer, itemHandler.ListSubItems)
		projGroup.POST("/items/:itemId/sub-items", member, itemHandler.CreateSubItem)
		projGroup.PATCH("/items/:itemId/sub-items/:subItemId", member, itemHandler.UpdateSubItem)
		projGroup.DELETE("/items/:itemId/sub-items/:subItemId", member, itemHandler.DeleteSubItem)

		// Comments
		projGroup.GET("/items/:itemId/comments", viewer, commentHandler.ListComments)
		projGroup.POST("/items/:itemId/comments", member, commentHandler.CreateComment)
		projGroup.DELETE("/items/:itemId/comments/:commentId", member, commentHandler.DeleteComment)

		// Wiki Pages & Versions
		projGroup.GET("/pages", viewer, pageHandler.ListPages)
		projGroup.POST("/pages", member, pageHandler.CreatePage)
		projGroup.GET("/pages/:pageId", viewer, pageHandler.GetPage)
		projGroup.PUT("/pages/:pageId", member, pageHandler.UpdatePage)
		projGroup.PATCH("/pages/:pageId/lock", member, pageHandler.SetPageLock)
		projGroup.DELETE("/pages/:pageId", member, pageHandler.DeletePage)
		projGroup.GET("/pages/:pageId/versions", viewer, pageHandler.ListVersions)
		projGroup.POST("/pages/:pageId/versions/:versionId/restore", member, pageHandler.RestoreVersion)

		// Page Collections
		projGroup.GET("/collections", viewer, pageHandler.ListCollections)
		projGroup.POST("/collections", member, pageHandler.CreateCollection)
		projGroup.PATCH("/collections/:collectionId", member, pageHandler.UpdateCollection)
		projGroup.DELETE("/collections/:collectionId", member, pageHandler.DeleteCollection)

		// Stickies
		projGroup.GET("/stickies", viewer, stickyHandler.ListStickies)
		projGroup.POST("/stickies", member, stickyHandler.CreateSticky)
		projGroup.PATCH("/stickies/:stickyId", member, stickyHandler.UpdateSticky)
		projGroup.DELETE("/stickies/:stickyId", member, stickyHandler.DeleteSticky)
	}

	// Static SPA asset serving & HTML5 history routing fallback (when STATIC_DIR is configured)
	if cfg.StaticDir != "" {
		cleanStaticDir := filepath.Clean(cfg.StaticDir)
		indexFile := filepath.Join(cleanStaticDir, "index.html")

		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// Don't intercept unmatched API routes; return standard 404 JSON
			if strings.HasPrefix(path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
				return
			}

			// If the requested static file exists on disk, serve it directly
			candidate := filepath.Join(cleanStaticDir, filepath.Clean(path))
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				c.File(candidate)
				return
			}

			// Otherwise, fall back to index.html for Angular client-side push-state routing
			if _, err := os.Stat(indexFile); err == nil {
				c.File(indexFile)
				return
			}

			c.JSON(http.StatusNotFound, gin.H{"error": "static frontend not found"})
		})
	}

	return r
}
