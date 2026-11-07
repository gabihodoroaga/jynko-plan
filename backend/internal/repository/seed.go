package repository

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

func parseSeedTime(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Now().UTC()
	}
	return t
}

func ptrTime(ts string) *time.Time {
	t := parseSeedTime(ts)
	return &t
}

func ptrInt(v int) *int {
	return &v
}

// EnsureSeedData idempotently populates the Firestore database with Jynko's initial
// projects, RBAC members, modules, epics, cycles, labels, work items, sub-items,
// comments, wiki collections/pages/versions, and stickies if "proj-jynko-core" does not exist yet.
func EnsureSeedData(ctx context.Context, fc *FirestoreClient) error {
	coreRef := fc.ProjectDoc("proj-jynko-core")
	snap, err := coreRef.Get(ctx)
	if err == nil && snap.Exists() {
		slog.InfoContext(ctx, "Firestore seed check: project already exists",
			slog.String("collection", fc.ProjectsCollection),
			slog.String("project_id", "proj-jynko-core"),
			slog.String("gcp_project", fc.ProjectID),
		)
		return nil
	}
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	slog.InfoContext(ctx, "Seeding initial Jynko data into Firestore",
		slog.String("collection", fc.ProjectsCollection),
		slog.String("gcp_project", fc.ProjectID),
	)
	batch := fc.Client.Batch()

	// =========================================================================
	// 1. Project 1: proj-jynko-core ("hodo")
	// =========================================================================
	projHodo := &model.Project{
		ID:          "proj-jynko-core",
		Name:        "hodo",
		KeyPrefix:   "HODO",
		Description: "Core engineering project for hodo project management and knowledge base.",
		ItemSeq:     6,
		MemberIDs:   []string{"dev-user-1", "dev@jynko.local", "user-2", "alex.rivera@jynko.local", "user-3", "sam.chen@jynko.local"},
		Settings:    model.DefaultProjectSettings(),
		CreatedBy:   "dev-user-1",
		CreatedAt:   parseSeedTime("2026-10-01T09:00:00Z"),
		UpdatedAt:   parseSeedTime("2026-10-28T18:00:00Z"),
	}
	batch.Set(coreRef, projHodo)

	// Members for proj-jynko-core
	hodoMembers := []model.ProjectMember{
		{
			UserID:   "dev-user-1",
			Email:    "dev@jynko.local",
			Name:     "Local Developer",
			Role:     model.RoleAdmin,
			JoinedAt: parseSeedTime("2026-10-01T09:00:00Z"),
		},
		{
			UserID:   "user-2",
			Email:    "alex.rivera@jynko.local",
			Name:     "Alex Rivera",
			Role:     model.RoleMember,
			JoinedAt: parseSeedTime("2026-10-02T10:00:00Z"),
		},
		{
			UserID:   "user-3",
			Email:    "sam.chen@jynko.local",
			Name:     "Sam Chen",
			Role:     model.RoleViewer,
			JoinedAt: parseSeedTime("2026-10-05T14:00:00Z"),
		},
	}
	for _, m := range hodoMembers {
		memberCopy := m
		batch.Set(coreRef.Collection("members").Doc(m.UserID), &memberCopy)
	}

	// Modules for proj-jynko-core
	hodoModules := []model.Module{
		{
			ID:          "mod-auth",
			ProjectID:   "proj-jynko-core",
			Name:        "IAP Security & RBAC",
			Description: "Google Cloud IAP header authentication and per-project role enforcement.",
			LeadUserID:  "dev-user-1",
			CreatedAt:   parseSeedTime("2026-10-02T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-20T09:00:00Z"),
		},
		{
			ID:          "mod-wiki",
			ProjectID:   "proj-jynko-core",
			Name:        "Wiki Knowledge Engine",
			Description: "Hierarchical page tree, Markdown block editor with slash commands, and version history.",
			LeadUserID:  "user-2",
			CreatedAt:   parseSeedTime("2026-10-03T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-22T09:00:00Z"),
		},
		{
			ID:          "mod-agent",
			ProjectID:   "proj-jynko-core",
			Name:        "AI Agent Everywhere",
			Description: "Google Gemini powered summarization, search, and natural language task mutations.",
			LeadUserID:  "dev-user-1",
			CreatedAt:   parseSeedTime("2026-10-04T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-25T09:00:00Z"),
		},
	}
	for _, mod := range hodoModules {
		modCopy := mod
		batch.Set(coreRef.Collection("modules").Doc(mod.ID), &modCopy)
	}

	// Epics for proj-jynko-core
	hodoEpics := []model.Epic{
		{
			ID:          "epic-q4-launch",
			ProjectID:   "proj-jynko-core",
			Name:        "Q4 Open-Source v1.0 Launch",
			Description: "Deliver all 5 Work Item views, Firestore persistence, and Gemini Agent integration.",
			Status:      model.EpicStatusInProgress,
			StartDate:   ptrTime("2026-10-01T00:00:00Z"),
			TargetDate:  ptrTime("2026-11-30T00:00:00Z"),
			CreatedAt:   parseSeedTime("2026-10-01T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-28T09:00:00Z"),
		},
		{
			ID:          "epic-wiki-v2",
			ProjectID:   "proj-jynko-core",
			Name:        "Knowledge Base",
			Description: "Nested page trees, collections, Markdown slash-command block editing, and audit snapshots.",
			Status:      model.EpicStatusInProgress,
			StartDate:   ptrTime("2026-10-10T00:00:00Z"),
			TargetDate:  ptrTime("2026-11-15T00:00:00Z"),
			CreatedAt:   parseSeedTime("2026-10-10T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-28T09:00:00Z"),
		},
	}
	for _, ep := range hodoEpics {
		epCopy := ep
		batch.Set(coreRef.Collection("epics").Doc(ep.ID), &epCopy)
	}

	// Cycles for proj-jynko-core
	hodoCycles := []model.Cycle{
		{
			ID:          "cycle-sprint-4",
			ProjectID:   "proj-jynko-core",
			Name:        "Cycle 4 — Core Views & AI Agent",
			Description: "Active two-week iteration focused on Timeline, Calendar, Spreadsheet, and Gemini Agent.",
			StartDate:   parseSeedTime("2026-10-20T00:00:00Z"),
			EndDate:     parseSeedTime("2026-11-03T00:00:00Z"),
			IsActive:    true,
			CreatedAt:   parseSeedTime("2026-10-20T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-28T09:00:00Z"),
		},
		{
			ID:          "cycle-sprint-3",
			ProjectID:   "proj-jynko-core",
			Name:        "Cycle 3 — Foundation & Firestore Schema",
			Description: "Completed sprint establishing Go Gin server and Firestore repositories.",
			StartDate:   parseSeedTime("2026-10-05T00:00:00Z"),
			EndDate:     parseSeedTime("2026-10-19T00:00:00Z"),
			IsActive:    false,
			CreatedAt:   parseSeedTime("2026-10-05T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-19T18:00:00Z"),
		},
	}
	for _, cy := range hodoCycles {
		cyCopy := cy
		batch.Set(coreRef.Collection("cycles").Doc(cy.ID), &cyCopy)
	}

	// Labels for proj-jynko-core
	hodoLabels := []model.Label{
		{ID: "lbl-arch", ProjectID: "proj-jynko-core", Name: "Architecture", Color: "#0284c7", CreatedAt: parseSeedTime("2026-10-01T00:00:00Z")},
		{ID: "lbl-security", ProjectID: "proj-jynko-core", Name: "Security / IAP", Color: "#be123c", CreatedAt: parseSeedTime("2026-10-01T00:00:00Z")},
		{ID: "lbl-ui", ProjectID: "proj-jynko-core", Name: " UI", Color: "#15803d", CreatedAt: parseSeedTime("2026-10-01T00:00:00Z")},
		{ID: "lbl-ai", ProjectID: "proj-jynko-core", Name: "Gemini Agent", Color: "#b45309", CreatedAt: parseSeedTime("2026-10-01T00:00:00Z")},
	}
	for _, lbl := range hodoLabels {
		lblCopy := lbl
		batch.Set(coreRef.Collection("labels").Doc(lbl.ID), &lblCopy)
	}

	// WorkItems for proj-jynko-core
	hodoItems := []model.WorkItem{
		{
			ID:                "item-1",
			ProjectID:         "proj-jynko-core",
			Key:               "HODO-1",
			SequenceID:        1,
			ModuleID:          "mod-auth",
			EpicID:            "epic-q4-launch",
			CycleID:           "cycle-sprint-4",
			Title:             "Implement Google Cloud IAP Header Auth & RBAC Middleware",
			Description:       "Validate `X-Goog-Authenticated-User-Id` and `X-Goog-Authenticated-User-Email` headers in Gin and enforce per-project `ADMIN`, `MEMBER`, and `VIEWER` roles.",
			Status:            model.StatusDone,
			Priority:          model.PriorityUrgent,
			AssigneeID:        "dev-user-1",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-security", "lbl-arch"},
			Estimate:          ptrInt(5),
			StartDate:         ptrTime("2026-10-20T00:00:00Z"),
			DueDate:           ptrTime("2026-10-24T00:00:00Z"),
			SubItemCount:      2,
			CompletedSubCount: 2,
			CreatedAt:         parseSeedTime("2026-10-20T09:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-24T17:00:00Z"),
		},
		{
			ID:                "item-2",
			ProjectID:         "proj-jynko-core",
			Key:               "HODO-2",
			SequenceID:        2,
			ModuleID:          "mod-wiki",
			EpicID:            "epic-wiki-v2",
			CycleID:           "cycle-sprint-4",
			Title:             "Build Markdown Block Wiki Editor with Slash Commands (/)",
			Description:       "Implement block-based Markdown editing supporting `/h1`, `/h2`, `/todo`, `/code`, `/quote`, live preview, page locking, and immutable `PageVersion` revision history.",
			Status:            model.StatusInProgress,
			Priority:          model.PriorityHigh,
			AssigneeID:        "user-2",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-ui"},
			Estimate:          ptrInt(8),
			StartDate:         ptrTime("2026-10-23T00:00:00Z"),
			DueDate:           ptrTime("2026-10-30T00:00:00Z"),
			SubItemCount:      3,
			CompletedSubCount: 2,
			CreatedAt:         parseSeedTime("2026-10-22T10:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-28T11:00:00Z"),
		},
		{
			ID:                "item-3",
			ProjectID:         "proj-jynko-core",
			Key:               "HODO-3",
			SequenceID:        3,
			ModuleID:          "mod-agent",
			EpicID:            "epic-q4-launch",
			CycleID:           "cycle-sprint-4",
			Title:             "Integrate Google Gemini SDK in AI Agent Dispatcher",
			Description:       "Connect `google.golang.org/genai` behind `LLMProvider` for work item thread summarization, unified Firestore search ranking, and automated task creation.",
			Status:            model.StatusInReview,
			Priority:          model.PriorityHigh,
			AssigneeID:        "dev-user-1",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-ai", "lbl-arch"},
			Estimate:          ptrInt(5),
			StartDate:         ptrTime("2026-10-25T00:00:00Z"),
			DueDate:           ptrTime("2026-10-29T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-25T09:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-28T14:00:00Z"),
		},
		{
			ID:                "item-4",
			ProjectID:         "proj-jynko-core",
			Key:               "HODO-4",
			SequenceID:        4,
			ModuleID:          "mod-wiki",
			EpicID:            "epic-q4-launch",
			CycleID:           "cycle-sprint-4",
			Title:             "Deliver Calendar, Timeline (Gantt) & Spreadsheet Work Item Views",
			Description:       "Ensure all 5 views in the light-mode toolbar share reactive filters and allow inline status, priority, estimate, and schedule updates.",
			Status:            model.StatusInProgress,
			Priority:          model.PriorityMedium,
			AssigneeID:        "dev-user-1",
			ReporterID:        "user-2",
			LabelIDs:          []string{"lbl-ui"},
			Estimate:          ptrInt(8),
			StartDate:         ptrTime("2026-10-26T00:00:00Z"),
			DueDate:           ptrTime("2026-11-02T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-26T08:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-28T15:00:00Z"),
		},
		{
			ID:                "item-5",
			ProjectID:         "proj-jynko-core",
			Key:               "HODO-5",
			SequenceID:        5,
			ModuleID:          "mod-auth",
			EpicID:            "epic-q4-launch",
			CycleID:           "cycle-sprint-4",
			Title:             "Atomic Firestore Transaction for Sequential Item Keys",
			Description:       "Increment `Project.ItemSeq` inside a Firestore transaction when creating new work items so keys (`HODO-1`, `HODO-2`) never collide under concurrency.",
			Status:            model.StatusTodo,
			Priority:          model.PriorityHigh,
			AssigneeID:        "user-2",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-arch"},
			Estimate:          ptrInt(3),
			StartDate:         ptrTime("2026-10-28T00:00:00Z"),
			DueDate:           ptrTime("2026-11-01T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-27T12:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-27T12:00:00Z"),
		},
		{
			ID:                "item-6",
			ProjectID:         "proj-jynko-core",
			Key:               "HODO-6",
			SequenceID:        6,
			ModuleID:          "mod-agent",
			EpicID:            "epic-wiki-v2",
			Title:             "One-Click Convert Discussion Thread into Structured Wiki Spec",
			Description:       "Allow engineers to convert any work item discussion into a formatted Wiki page inside the project knowledge base.",
			Status:            model.StatusBacklog,
			Priority:          model.PriorityLow,
			AssigneeID:        "user-3",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-ai"},
			Estimate:          ptrInt(3),
			StartDate:         ptrTime("2026-11-01T00:00:00Z"),
			DueDate:           ptrTime("2026-11-06T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-28T10:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-28T10:00:00Z"),
		},
	}
	for _, it := range hodoItems {
		itCopy := it
		batch.Set(coreRef.Collection("work_items").Doc(it.ID), &itCopy)
	}

	// Sub-items for item-1 and item-2
	subItems := []model.SubWorkItem{
		{
			ID:           "sub-1",
			ParentItemID: "item-1",
			Title:        "Parse X-Goog-Authenticated-User-Id and Email headers",
			IsCompleted:  true,
			AssigneeID:   "dev-user-1",
			CreatedAt:    parseSeedTime("2026-10-20T10:00:00Z"),
			UpdatedAt:    parseSeedTime("2026-10-21T10:00:00Z"),
		},
		{
			ID:           "sub-2",
			ParentItemID: "item-1",
			Title:        "Add unit tests for Viewer, Member, and Admin RBAC hierarchy",
			IsCompleted:  true,
			AssigneeID:   "dev-user-1",
			CreatedAt:    parseSeedTime("2026-10-21T10:00:00Z"),
			UpdatedAt:    parseSeedTime("2026-10-22T10:00:00Z"),
		},
		{
			ID:           "sub-3",
			ParentItemID: "item-2",
			Title:        "Implement slash command menu (/h1, /h2, /todo, /code, /quote)",
			IsCompleted:  true,
			AssigneeID:   "user-2",
			CreatedAt:    parseSeedTime("2026-10-23T10:00:00Z"),
			UpdatedAt:    parseSeedTime("2026-10-25T10:00:00Z"),
		},
		{
			ID:           "sub-4",
			ParentItemID: "item-2",
			Title:        "Enforce page lock indicator and read-only mode",
			IsCompleted:  true,
			AssigneeID:   "user-2",
			CreatedAt:    parseSeedTime("2026-10-24T10:00:00Z"),
			UpdatedAt:    parseSeedTime("2026-10-26T10:00:00Z"),
		},
		{
			ID:           "sub-5",
			ParentItemID: "item-2",
			Title:        "Add side-by-side live Markdown preview mode",
			IsCompleted:  false,
			AssigneeID:   "user-2",
			CreatedAt:    parseSeedTime("2026-10-26T10:00:00Z"),
			UpdatedAt:    parseSeedTime("2026-10-26T10:00:00Z"),
		},
	}
	for _, sub := range subItems {
		subCopy := sub
		batch.Set(coreRef.Collection("work_items").Doc(sub.ParentItemID).Collection("sub_items").Doc(sub.ID), &subCopy)
	}

	// Comments for item-2
	comments := []model.Comment{
		{
			ID:         "cmt-1",
			WorkItemID: "item-2",
			AuthorID:   "dev-user-1",
			AuthorName: "Local Developer",
			Body:       "Make sure that updating a Wiki page automatically snapshots a `PageVersion` record in Firestore and rejects edits if `is_locked == true`.",
			CreatedAt:  parseSeedTime("2026-10-26T14:20:00Z"),
			UpdatedAt:  parseSeedTime("2026-10-26T14:20:00Z"),
		},
		{
			ID:         "cmt-2",
			WorkItemID: "item-2",
			AuthorID:   "user-2",
			AuthorName: "Alex Rivera",
			Body:       "Implemented! Also removed `AccessLevel` so all pages cleanly inherit the parent project RBAC roles.",
			CreatedAt:  parseSeedTime("2026-10-27T09:15:00Z"),
			UpdatedAt:  parseSeedTime("2026-10-27T09:15:00Z"),
		},
	}
	for _, cmt := range comments {
		cmtCopy := cmt
		batch.Set(coreRef.Collection("work_items").Doc(cmt.WorkItemID).Collection("comments").Doc(cmt.ID), &cmtCopy)
	}

	// Wiki Collections for proj-jynko-core
	hodoCollections := []model.PageCollection{
		{
			ID:          "col-general",
			ProjectID:   "proj-jynko-core",
			Name:        "General",
			Description: "General company & workspace knowledge base.",
			CreatedAt:   parseSeedTime("2026-10-01T08:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-28T09:00:00Z"),
		},
		{
			ID:          "col-proj-jynko-core",
			ProjectID:   "proj-jynko-core",
			Name:        "hodo",
			Description: "Core engineering project for hodo project management and knowledge base.",
			CreatedAt:   parseSeedTime("2026-10-01T09:00:00Z"),
			UpdatedAt:   parseSeedTime("2026-10-28T18:00:00Z"),
		},
	}
	for _, col := range hodoCollections {
		colCopy := col
		batch.Set(coreRef.Collection("collections").Doc(col.ID), &colCopy)
	}

	// Wiki Pages for proj-jynko-core
	hodoPages := []model.Page{
		{
			ID:           "page-welcome",
			ProjectID:    "proj-jynko-core",
			CollectionID: "col-general",
			Title:        "Welcome to Company's Wiki",
			Content: `This is **your Wiki Home** — the heart of your project where **knowledge**, **documents**, and **processes live**.

Use **this page** to document your company, your projects, and your people.

Each section below shows how you can use **different block types** to organize content beautifully.

---

## 🏢 About Your Company

### 🌍 Mission & Vision

> "Every great company starts with a clear why."
>
> *✏️ Edit this block to describe your mission.*
> Example: "Our mission is to simplify how teams plan, collaborate, and deliver work — together."

### 💎 Core Values

- **Transparency by Default**: Every project decision is documented and searchable.
- **Velocity with Craft**: Ship fast while keeping clean, maintainable architecture.
- **Customer Obsession**: Solve real workflow bottlenecks with clarity.`,
			Order:     1,
			IsLocked:  false,
			LabelIDs:  []string{},
			CreatedBy: "dev-user-1",
			UpdatedBy: "dev-user-1",
			CreatedAt: parseSeedTime("2026-10-01T08:30:00Z"),
			UpdatedAt: parseSeedTime("2026-10-29T08:00:00Z"),
		},
		{
			ID:           "page-arch-overview",
			ProjectID:    "proj-jynko-core",
			CollectionID: "col-proj-jynko-core",
			Title:        "Jynko System Architecture Overview",
			Content: `# Jynko System Architecture Overview

Jynko is an open-source project management and knowledge base platform built with **Go (Gin)**, **Google Cloud Firestore**, **Google Cloud IAP**, **Google Gemini API**, and **Angular 18+ & Tailwind CSS**.

## Core Architectural Invariants

- **No Workspaces**: ` + "`Project`" + ` (` + "`projects/{projectId}`" + `) is the top-level tenant and RBAC isolation boundary.
- **Google Cloud IAP Authentication**: Identity is extracted directly from ` + "`X-Goog-Authenticated-User-Id`" + ` and ` + "`X-Goog-Authenticated-User-Email`" + ` headers.
- **Live GCP Firestore Persistence**: Atomic sequence numbering generates sequential keys (` + "`HODO-1`" + `, ` + "`HODO-2`" + `).

## Slash Commands in the Editor

Type ` + "`/`" + ` on any new line or use the block toolbar to insert headings, task lists, code blocks, quotes, or dividers.`,
			Order:     1,
			IsLocked:  false,
			LabelIDs:  []string{"lbl-arch"},
			CreatedBy: "dev-user-1",
			UpdatedBy: "dev-user-1",
			CreatedAt: parseSeedTime("2026-10-01T10:00:00Z"),
			UpdatedAt: parseSeedTime("2026-10-28T16:00:00Z"),
		},
		{
			ID:           "page-iap-security",
			ProjectID:    "proj-jynko-core",
			CollectionID: "col-proj-jynko-core",
			ParentPageID: "page-arch-overview",
			Title:        "Authentication & Per-Project RBAC Specification",
			Content: `# Authentication & Per-Project RBAC Specification

All requests to ` + "`/api/v1/*`" + ` are authenticated via **Google Cloud Identity-Aware Proxy (IAP)** headers.

## Role Hierarchy

1. **ADMIN**: Full control over project settings, member role assignments, work items, and wiki pages.
2. **MEMBER**: Can create, edit, and delete work items, sub-items, cycles, epics, modules, wiki pages, and run AI Agent mutation tools.
3. **VIEWER**: Read-only access across all views and wiki documentation; can run AI Agent search and summarization queries.`,
			Order:     1,
			IsLocked:  false,
			LabelIDs:  []string{"lbl-security"},
			CreatedBy: "dev-user-1",
			UpdatedBy: "user-2",
			CreatedAt: parseSeedTime("2026-10-05T11:00:00Z"),
			UpdatedAt: parseSeedTime("2026-10-27T14:00:00Z"),
		},
		{
			ID:           "page-release-checklist",
			ProjectID:    "proj-jynko-core",
			CollectionID: "col-proj-jynko-core",
			Title:        "v1.0 Production Release Runbook",
			Content: `# v1.0 Production Release Runbook

Follow this checklist prior to deploying a new Jynko release to GCP Cloud Run / GKE behind IAP:

- [x] Verify ` + "`GOOGLE_CLOUD_PROJECT`" + ` points to the live Firestore instance
- [x] Confirm ` + "`GEMINI_API_KEY`" + ` secret is mounted
- [x] Run ` + "`go test ./...`" + ` in ` + "`/backend`" + `
- [ ] Verify IAP OAuth2 audience header enforcement`,
			Order:     2,
			IsLocked:  true,
			LabelIDs:  []string{"lbl-arch"},
			CreatedBy: "user-2",
			UpdatedBy: "dev-user-1",
			CreatedAt: parseSeedTime("2026-10-12T09:00:00Z"),
			UpdatedAt: parseSeedTime("2026-10-28T09:00:00Z"),
		},
	}
	for _, pg := range hodoPages {
		pgCopy := pg
		pageRef := coreRef.Collection("pages").Doc(pg.ID)
		batch.Set(pageRef, &pgCopy)
		ver := &model.PageVersion{
			ID:        "ver-init-" + pg.ID,
			PageID:    pg.ID,
			Title:     pg.Title,
			Content:   pg.Content,
			CreatedBy: pg.CreatedBy,
			CreatedAt: pg.CreatedAt,
		}
		batch.Set(pageRef.Collection("versions").Doc(ver.ID), ver)
	}

	// Stickies for proj-jynko-core
	hodoStickies := []model.Sticky{
		{
			ID:        "sticky-1",
			ProjectID: "proj-jynko-core",
			Content:   "⚡ Verify Firestore transaction increments Project.ItemSeq atomically for HODO-7 and beyond.",
			Color:     model.StickyColorBanana,
			IsPinned:  true,
			AuthorID:  "dev-user-1",
			CreatedAt: parseSeedTime("2026-10-28T09:15:00Z"),
			UpdatedAt: parseSeedTime("2026-10-29T10:00:00Z"),
		},
		{
			ID:        "sticky-2",
			ProjectID: "proj-jynko-core",
			Content:   "🦩 Idea: Add slash command /callout in the Markdown Wiki editor for architectural decision highlights.",
			Color:     model.StickyColorFlamingo,
			IsPinned:  true,
			AuthorID:  "dev-user-1",
			CreatedAt: parseSeedTime("2026-10-28T11:20:00Z"),
			UpdatedAt: parseSeedTime("2026-10-29T09:30:00Z"),
		},
		{
			ID:        "sticky-3",
			ProjectID: "proj-jynko-core",
			Content:   "🌿 Every new project automatically gets its own Wiki Collection so specs stay organized per project.",
			Color:     model.StickyColorMint,
			IsPinned:  false,
			AuthorID:  "user-2",
			CreatedAt: parseSeedTime("2026-10-28T14:00:00Z"),
			UpdatedAt: parseSeedTime("2026-10-29T08:45:00Z"),
		},
		{
			ID:        "sticky-5",
			ProjectID: "proj-jynko-core",
			Content:   "🔮 Gemini Agent prompt tuning: summarize blockers and overdue items in <150 words.",
			Color:     model.StickyColorLavender,
			IsPinned:  false,
			AuthorID:  "dev-user-1",
			CreatedAt: parseSeedTime("2026-10-28T17:00:00Z"),
			UpdatedAt: parseSeedTime("2026-10-29T07:50:00Z"),
		},
	}
	for _, st := range hodoStickies {
		stCopy := st
		batch.Set(coreRef.Collection("stickies").Doc(st.ID), &stCopy)
	}

	// =========================================================================
	// 2. Project 2: proj-cloud-infra ("cloud-infra")
	// =========================================================================
	cldRef := fc.ProjectDoc("proj-cloud-infra")
	projCld := &model.Project{
		ID:          "proj-cloud-infra",
		Name:        "cloud-infra",
		KeyPrefix:   "CLD",
		Description: "GCP Cloud Run, Firestore multi-region provisioning, and IAP zero-trust ingress.",
		ItemSeq:     4,
		MemberIDs:   []string{"dev-user-1", "dev@jynko.local", "user-2", "alex.rivera@jynko.local"},
		Settings:    model.DefaultProjectSettings(),
		CreatedBy:   "dev-user-1",
		CreatedAt:   parseSeedTime("2026-10-08T10:00:00Z"),
		UpdatedAt:   parseSeedTime("2026-10-29T08:30:00Z"),
	}
	batch.Set(cldRef, projCld)

	cldMembers := []model.ProjectMember{
		{
			UserID:   "dev-user-1",
			Email:    "dev@jynko.local",
			Name:     "Local Developer",
			Role:     model.RoleAdmin,
			JoinedAt: parseSeedTime("2026-10-08T10:00:00Z"),
		},
		{
			UserID:   "user-2",
			Email:    "alex.rivera@jynko.local",
			Name:     "Alex Rivera",
			Role:     model.RoleMember,
			JoinedAt: parseSeedTime("2026-10-08T11:00:00Z"),
		},
	}
	for _, m := range cldMembers {
		memberCopy := m
		batch.Set(cldRef.Collection("members").Doc(m.UserID), &memberCopy)
	}

	cldItems := []model.WorkItem{
		{
			ID:                "item-cld-1",
			ProjectID:         "proj-cloud-infra",
			Key:               "CLD-1",
			SequenceID:        1,
			Title:             "Configure Cloud Load Balancer IAP OAuth2 Brand & Audience Verification",
			Description:       "Provision Google Cloud Load Balancer backend service with Identity-Aware Proxy enabled and verify JWT audience headers in Go middleware.",
			Status:            model.StatusInProgress,
			Priority:          model.PriorityUrgent,
			AssigneeID:        "dev-user-1",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-security"},
			Estimate:          ptrInt(5),
			StartDate:         ptrTime("2026-10-24T00:00:00Z"),
			DueDate:           ptrTime("2026-10-31T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-24T10:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-29T08:15:00Z"),
		},
		{
			ID:                "item-cld-2",
			ProjectID:         "proj-cloud-infra",
			Key:               "CLD-2",
			SequenceID:        2,
			Title:             "Terraform Module for Firestore Composite Indexes & Daily Backups",
			Description:       "Automate Firestore database creation in `nam5` with PITR (Point-in-Time Recovery) enabled and composite index definitions.",
			Status:            model.StatusTodo,
			Priority:          model.PriorityHigh,
			AssigneeID:        "dev-user-1",
			ReporterID:        "user-2",
			LabelIDs:          []string{"lbl-arch"},
			Estimate:          ptrInt(3),
			StartDate:         ptrTime("2026-10-28T00:00:00Z"),
			DueDate:           ptrTime("2026-11-03T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-27T11:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-28T16:00:00Z"),
		},
		{
			ID:                "item-cld-3",
			ProjectID:         "proj-cloud-infra",
			Key:               "CLD-3",
			SequenceID:        3,
			Title:             "Secret Manager Rotation Policy for Gemini API Keys",
			Description:       "Mount `GEMINI_API_KEY` from Google Cloud Secret Manager into Cloud Run revisions with least-privilege IAM service account bindings.",
			Status:            model.StatusDone,
			Priority:          model.PriorityMedium,
			AssigneeID:        "dev-user-1",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-security", "lbl-ai"},
			Estimate:          ptrInt(2),
			StartDate:         ptrTime("2026-10-18T00:00:00Z"),
			DueDate:           ptrTime("2026-10-22T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-18T09:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-22T15:00:00Z"),
		},
		{
			ID:                "item-cld-4",
			ProjectID:         "proj-cloud-infra",
			Key:               "CLD-4",
			SequenceID:        4,
			Title:             "Cloud Run Min-Instance Warmup & CPU Throttling Benchmark",
			Description:       "Benchmark cold-start latency for Go Gin server + Firestore client initialization under burst traffic.",
			Status:            model.StatusInReview,
			Priority:          model.PriorityMedium,
			AssigneeID:        "user-2",
			ReporterID:        "dev-user-1",
			LabelIDs:          []string{"lbl-arch"},
			Estimate:          ptrInt(3),
			StartDate:         ptrTime("2026-10-26T00:00:00Z"),
			DueDate:           ptrTime("2026-11-01T00:00:00Z"),
			SubItemCount:      0,
			CompletedSubCount: 0,
			CreatedAt:         parseSeedTime("2026-10-26T14:00:00Z"),
			UpdatedAt:         parseSeedTime("2026-10-28T17:30:00Z"),
		},
	}
	for _, it := range cldItems {
		itCopy := it
		batch.Set(cldRef.Collection("work_items").Doc(it.ID), &itCopy)
	}

	cldCol := &model.PageCollection{
		ID:          "col-proj-cloud-infra",
		ProjectID:   "proj-cloud-infra",
		Name:        "cloud-infra",
		Description: "GCP Cloud Run, Firestore multi-region provisioning, and IAP zero-trust ingress.",
		CreatedAt:   parseSeedTime("2026-10-08T10:00:00Z"),
		UpdatedAt:   parseSeedTime("2026-10-29T08:30:00Z"),
	}
	batch.Set(cldRef.Collection("collections").Doc(cldCol.ID), cldCol)

	cldPage := &model.Page{
		ID:           "page-cloud-infra-home",
		ProjectID:    "proj-cloud-infra",
		CollectionID: "col-proj-cloud-infra",
		Title:        "cloud-infra — GCP & Terraform Runbook",
		Content: `# cloud-infra — GCP & Terraform Runbook

This collection contains all infrastructure documentation and operational playbooks for **cloud-infra** (` + "`CLD`" + `).

## Architecture Highlights

- **Multi-Region Firestore (` + "`nam5`" + `)**: Zero-downtime document storage with atomic transactions.
- **Cloud Run & Identity-Aware Proxy (IAP)**: Enforces Google Workspace OIDC identity headers on every ingress request.`,
		Order:     1,
		IsLocked:  false,
		LabelIDs:  []string{},
		CreatedBy: "dev-user-1",
		UpdatedBy: "dev-user-1",
		CreatedAt: parseSeedTime("2026-10-08T10:00:00Z"),
		UpdatedAt: parseSeedTime("2026-10-29T08:30:00Z"),
	}
	cldPageRef := cldRef.Collection("pages").Doc(cldPage.ID)
	batch.Set(cldPageRef, cldPage)
	batch.Set(cldPageRef.Collection("versions").Doc("ver-init-page-cloud-infra-home"), &model.PageVersion{
		ID:        "ver-init-page-cloud-infra-home",
		PageID:    cldPage.ID,
		Title:     cldPage.Title,
		Content:   cldPage.Content,
		CreatedBy: cldPage.CreatedBy,
		CreatedAt: cldPage.CreatedAt,
	})

	cldStickies := []model.Sticky{
		{
			ID:        "sticky-4",
			ProjectID: "proj-cloud-infra",
			Content:   "🌊 Cloud Run zero-trust IAP ingress: confirm OAuth2 audience header on staging cluster.",
			Color:     model.StickyColorLagoon,
			IsPinned:  false,
			AuthorID:  "dev-user-1",
			CreatedAt: parseSeedTime("2026-10-28T16:10:00Z"),
			UpdatedAt: parseSeedTime("2026-10-29T08:15:00Z"),
		},
		{
			ID:        "sticky-6",
			ProjectID: "proj-cloud-infra",
			Content:   "🍊 Remember to snapshot Firestore indexes before the v1.0 production rollout.",
			Color:     model.StickyColorPeach,
			IsPinned:  false,
			AuthorID:  "user-2",
			CreatedAt: parseSeedTime("2026-10-28T18:00:00Z"),
			UpdatedAt: parseSeedTime("2026-10-29T07:30:00Z"),
		},
	}
	for _, st := range cldStickies {
		stCopy := st
		batch.Set(cldRef.Collection("stickies").Doc(st.ID), &stCopy)
	}

	_, err = batch.Commit(ctx)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "Successfully seeded initial Jynko data into Firestore",
		slog.String("collection", fc.ProjectsCollection),
		slog.String("gcp_project", fc.ProjectID),
	)
	return nil
}
