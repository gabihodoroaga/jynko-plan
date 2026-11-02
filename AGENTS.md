# AGENTS.md — Jynko AI Coding Assistant Guide

This file is the canonical reference for AI coding agents working on **Jynko**, an open-source project management and knowledge base platform. Read and follow these instructions in every session.

---

## 1. Core Architectural Invariants (Non-Negotiable)

1. **Projects**:
   - `Project` (`projects/{projectId}`) is the top-level tenant, permission boundary, and Firestore root collection.
2. **Authentication — Google Cloud IAP / OIDC Only**:
   - Production authentication relies strictly on Google Cloud Identity-Aware Proxy (IAP) / OIDC headers (`X-Goog-Authenticated-User-Id`, `X-Goog-Authenticated-User-Email`, `X-Goog-IAP-JWT-Assertion`).
   - Do **not** build password login forms, Firebase Auth, or custom JWT token issuers.
   - For local development (`ENV=development`), the backend auth middleware falls back to `X-Dev-User-Id`, `X-Dev-User-Email`, and `X-Dev-User-Name` headers (or `.env` defaults `DEV_USER_ID`, `DEV_USER_EMAIL`, `DEV_USER_NAME`).
   - Any authenticated user can create a new project (`POST /api/v1/projects`) and automatically becomes its initial `ADMIN`.
3. **Database — Live GCP Firestore Connection**:
   - Backend persistence connects directly to a live Google Cloud Firestore project (`GOOGLE_CLOUD_PROJECT`) using `cloud.google.com/go/firestore`.
   - Work item creation (`POST /api/v1/projects/:projectId/items`) must atomically increment `Project.ItemSeq` inside a Firestore transaction to generate sequential human-readable keys (`<KEY_PREFIX>-<SEQ>`, e.g., `JYNKO-1`).
4. **Per-Project RBAC (`ADMIN`, `MEMBER`, `VIEWER`)**:
   - Every project-scoped route (`/api/v1/projects/:projectId/...`) must enforce the user's `ProjectRole` stored in `projects/{projectId}/members/{userId}`.
   - Wiki pages (`Page`) do **not** have a separate `AccessLevel` field—page visibility and editing strictly follow the parent project's RBAC (`VIEWER` reads, `MEMBER`/`ADMIN` edits, `IsLocked` prevents edits while locked).
5. **UI/UX — Light Mode Design System (`slate-50`) & Two-Section Shell (`Work` & `Wiki`)**:
   - Strictly light-mode palette: `#f8fafc` (`bg-slate-50`) / `#f4f5f7` recessed canvas, `#ffffff` (`bg-white`) elevated cards/headers, `1px` hairline borders (`#e2e8f0` / `border-slate-200`), compact `11px` pill badges, and `#0284c7` (`bg-sky-600`) primary accents.
   - **Two Primary Sections (`Work` and `Wiki`)**:
     - Left Icon Rail (`68px`) has 3 buttons: **Work**, **Wiki**, and **Settings**.
     - **Work section**: Secondary panel (`w-64`) shows `Work` header, `+ New work item`, `Home`, `My Work`, `Stickies`, and the `Projects` tree (`+` button at root of `Projects` to create a project; `hodo` -> `Overview`, `Work items`, `Cycles`, `Epics`, `Modules`, `Pages`). Right side shows the active Work view (**List**, **Board**, **Calendar**, **Timeline**, **Spreadsheet**, **Epics**, **Cycles**, **Modules**, **Stickies**).
     - **Wiki section**: Secondary panel (`w-64`) switches to `Wiki` header, `+ New page`, `Home`, and `Collections` tree (each collection expands to show its nested **Pages tree**, plus `Shared`, `Private`, `Archives`). Right side shows ONLY the **Page View & Editor** (breadcrumb header, formatting toolbar, and centered Markdown block editor/viewer).
6. **End-to-End UI Testing (`github.com/go-rod/rod`) — Always Update E2E Tests**:
   - Jynko's automated E2E browser test suite is stored in the repository under `backend/internal/e2e/` (`validator.go` and `ui_e2e_test.go`), exposed via `go test` (`./scripts/run-e2e.sh`), CLI (`backend/cmd/ui-validator/main.go`), and GitHub Actions CI (`.github/workflows/e2e.yml`).
   - **Mandatory Requirement**: Whenever you add, modify, or remove any UI route, component, layout, or interactive control, you **must** update the corresponding E2E checks in `backend/internal/e2e/validator.go` (and `backend/internal/e2e/ui_e2e_test.go`) and run `./scripts/run-e2e.sh` to verify that all E2E tests and screenshots pass before completing your work.

---

## 2. Tech Stack & Directory Structure

### Backend (`/backend`)
- **Language & Runtime**: Go 1.22+
- **HTTP Framework**: Gin (`github.com/gin-gonic/gin`)
- **Database SDK**: Google Cloud Firestore (`cloud.google.com/go/firestore`)
- **Structure**:
  - `cmd/server/main.go`: HTTP server entrypoint and dependency wiring
  - `internal/config/`: Environment configuration loader
  - `internal/model/`: Domain structs (`domain.go`, `page.go`, `rbac.go`)
  - `internal/middleware/`: `auth.go` (IAP + dev fallback), `rbac.go` (`RequireProjectRole`), `cors.go`
  - `internal/repository/`: Firestore repositories (`project`, `item`, `cycle`, `epic`, `module`, `label`, `page`, `sticky`)
  - `internal/service/`: Business logic and validation layer
  - `internal/handler/`: Gin HTTP handlers under `/api/v1`
  - `internal/e2e/`: Go-Rod end-to-end browser test suite (`validator.go`, `ui_e2e_test.go`)
  - `cmd/ui-validator/main.go`: Standalone CLI wrapper for running the E2E UI validator and generating `report.json` + screenshots

### Frontend (`/frontend`)
- **Framework**: Angular 18+ (Standalone Components, Angular Signals, Angular CDK `@angular/cdk/drag-drop`)
- **Styling**: Tailwind CSS (`slate-50` light theme tokens defined in `tailwind.config.js`)
- **Structure**:
  - `src/app/core/`: `guards/`, `interceptors/`, `models/`, `services/` (`api`, `auth`, `project`, `work-item`, `page`, `sticky`, `settings`)
  - `src/app/features/work-items/`: `work-items-shell`, `work-item-list`, `work-item-board`, `work-item-calendar`, `work-item-timeline`, `work-item-spreadsheet`, `work-item-detail-drawer`
  - `src/app/features/wiki/`: `page-tree`, `collection-list`, `page-detail`, `page-editor` (Markdown block editor with `/` slash menu), `version-history`
  - `src/app/features/{projects,cycles,epics,modules,stickies,settings}/`
  - `src/app/shared/`: Reusable badges, avatars, modals, and pipes

---

## 3. Firestore Collection Hierarchy

All project data lives under `projects/{projectId}`:
- `projects/{projectId}` (`model.Project` — includes `member_ids []string` and `item_seq int`)
- `projects/{projectId}/members/{userId}` (`model.ProjectMember`)
- `projects/{projectId}/modules/{moduleId}` (`model.Module`)
- `projects/{projectId}/epics/{epicId}` (`model.Epic`)
- `projects/{projectId}/cycles/{cycleId}` (`model.Cycle`)
- `projects/{projectId}/labels/{labelId}` (`model.Label`)
- `projects/{projectId}/work_items/{itemId}` (`model.WorkItem`)
  - `projects/{projectId}/work_items/{itemId}/sub_items/{subItemId}` (`model.SubWorkItem`)
  - `projects/{projectId}/work_items/{itemId}/comments/{commentId}` (`model.Comment`)
- `projects/{projectId}/collections/{collectionId}` (`model.PageCollection`)
- `projects/{projectId}/pages/{pageId}` (`model.Page`)
  - `projects/{projectId}/pages/{pageId}/versions/{versionId}` (`model.PageVersion`)
- `projects/{projectId}/stickies/{stickyId}` (`model.Sticky`)

---

## 4. Coding Conventions & Best Practices

### Go Backend
- Pass `context.Context` (`c.Request.Context()`) as the first parameter to all repository and service methods.
- Return structured JSON errors (`{"error": "message"}`) with appropriate HTTP status codes (`400`, `401`, `403`, `404`, `500`).
- Ensure slices initialized for JSON responses default to `[]T{}` rather than `nil` so the API returns `[]` instead of `null`.
- Whenever a Wiki `Page` is updated (`PUT /api/v1/projects/:projectId/pages/:pageId`), reject edits if `page.IsLocked == true`, and automatically write a `PageVersion` snapshot.
- Keep `Project.MemberIDs` synchronized whenever members are added or removed in `projects/{projectId}/members/{userId}` so `GET /api/v1/projects` can query `Where("member_ids", "array-contains", userID)`.

### Angular Frontend
- Use **Standalone Components** (`standalone: true`) and **Angular Signals** (`signal`, `computed`) for reactive state in services and components.
- Follow the light-mode design tokens from [docs/ARCHITECTURE.md](file:///Users/gabrielhodoroaga/Projects/github/gabihodoroaga/jynko/docs/ARCHITECTURE.md):
  - Status values: `BACKLOG`, `TODO`, `IN_PROGRESS`, `IN_REVIEW`, `DONE`, `CANCELLED`
  - Priority values: `NONE`, `LOW`, `MEDIUM`, `HIGH`, `URGENT`
- Respect RBAC in the UI: disable or hide mutation controls when the current user's project role is `VIEWER`, and restrict project settings / member management to `ADMIN`.

### End-to-End (E2E) Testing (`backend/internal/e2e/`)
- **Always Keep E2E Tests Updated**: Every UI change or new feature must be accompanied by an updated or new `CheckSpec` in `backend/internal/e2e/validator.go` (`DefaultChecks()`).
- Attach clear `aria-label` attributes to interactive triggers (buttons, dropdowns, modals) in Angular templates and assert them in `backend/internal/e2e/validator.go` (also document new selectors in `.agents/skills/jynko-ui-validator/references/selectors-and-routes.md`).
- Run `./scripts/run-e2e.sh` after making changes to verify that `go test ./internal/e2e/...` passes and captures updated screenshots.

---

## 5. Build, Run & Verification Commands

### Backend
```bash
cd backend
go mod tidy
go build ./...
go test ./...
go run ./cmd/server/main.go
```

### Frontend
```bash
cd frontend
npm install
npx ng build
npx ng test --watch=false
npx ng serve
```

### Automated E2E UI Tests & Screenshots (`github.com/go-rod/rod` & CI/CD)
Read `.agents/skills/jynko-ui-validator/SKILL.md` for full instructions on running and extending Jynko's Go-Rod E2E test suite (`backend/internal/e2e/`).
```bash
# Run full E2E test suite (auto-starts backend on :8082 and frontend on :4202 if not already listening)
./scripts/run-e2e.sh /tmp/jynko-screenshots

# Or run directly via go test when frontend is already running on :4202
JYNKO_E2E=1 JYNKO_E2E_OUT_DIR=/tmp/jynko-screenshots go -C backend test -v -count=1 ./internal/e2e/...
```

### Required Environment Variables (`backend/.env`)
```env
PORT=8082
ENV=development
GOOGLE_CLOUD_PROJECT=your-gcp-project-id
FIRESTORE_DATABASE_ID=(default)
IAP_AUDIENCE=
DEV_USER_ID=dev-user-1
DEV_USER_EMAIL=dev@jynko.local
DEV_USER_NAME=Local Developer
CORS_ALLOWED_ORIGINS=http://localhost:4202
```

---

## 6. Reference Documentation

Always consult [docs/ARCHITECTURE.md](file:///Users/gabrielhodoroaga/Projects/github/gabihodoroaga/jynko/docs/ARCHITECTURE.md) for full domain struct definitions, API endpoint specifications, and UI color matrices.
