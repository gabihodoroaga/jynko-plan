# Jynko Routes, DOM Selectors, and UI Invariants Reference

Use this reference when writing or extending `github.com/go-rod/rod` checks in `backend/cmd/ui-validator/main.go`.

---

## 1. Application Routes (`frontend/src/app/app.routes.ts`)

| Route | Component | Section | Purpose & Key Assertions |
| :--- | :--- | :--- | :--- |
| `/home` | `HomeComponent` | `Work` | Cross-project overview of **all user projects** (`hodo`, `cloud-infra`), aggregate metrics (`Active Projects`, `Total Work Items`), project progress cards, and `My Assigned Work Across All Projects`. Must **not** contain a `+ New work item` button. |
| `/my-work` | `MyWorkComponent` | `Work` | Cross-project list of work items assigned to the current user (`HODO-1..4`, `CLD-1..3`). Supports toggling between `Group by Project` and `Group by Status`, plus styled `<app-dropdown>` filters for Status, Priority, and Project. |
| `/projects/:projectId` (and bare `/projects` redirect) | `ProjectsComponent` | `Work` | Active project (`hodo` / `proj-jynko-core`, `cloud-infra` / `proj-cloud-infra`) overview, stats, and project members table. Must **not** contain a `+ New Project` button in the page header (project creation lives in the left sidebar `+` button). |
| `/projects/:projectId/work-items` (and bare `/work-items` redirect) | `WorkItemsShellComponent` | `Work` | Work items shell with 5 view tabs (`List`, `Board`, `Calendar`, `Timeline`, `Spreadsheet`), search input, styled Status & Priority `<app-dropdown>` filters, and `+ New Work Item` button. |
| `/projects/:projectId/work-items/:itemId` | `WorkItemDetailPageComponent` | `Work` | Full-page single-column Work Item detail view with editable Title, Markdown Description, Status, Priority, Assignee, Module, Epic, Cycle, Start Date, Due Date, Estimate, Sub-Work Items checklist, and Discussion Comments. |
| `/projects/:projectId/cycles` (and bare `/cycles` redirect) | `CyclesComponent` | `Work` | Sprint cycles with progress bars and status badges. |
| `/projects/:projectId/epics` (and bare `/epics` redirect) | `EpicsComponent` | `Work` | High-level Epics tracker with completion progress bars. |
| `/projects/:projectId/modules` (and bare `/modules` redirect) | `ModulesComponent` | `Work` | Feature/system Modules grid. |
| `/stickies` | `StickiesComponent` | `Work` | Quick colorful sticky notes grid (`BANANA`, `FLAMINGO`, `MINT`, `LAGOON`, `LAVENDER`, `PEACH`) with color picker and instant filtering. |
| `/wiki` | `PageDetailComponent` | `Wiki` | Dedicated Wiki workspace. Left secondary panel shows `+ New page`, `Home`, `Collections -> General / <project> -> Pages tree`, `Shared`, `Private`, `Archives`. Right main pane shows only the Page View & Editor (`Welcome to Company's Wiki`, formatting toolbar). |
| `/settings` | `SettingsComponent` | `Settings` | Dedicated Settings section with **General App Settings** (Appearance & Defaults, Notifications, IAP Identity) and **Per-Project Settings** (`hodo`, `cloud-infra`, etc. — Metadata, Work Item & Cycle Defaults, Wiki Collection Policy, Per-Project RBAC Members). |
| `/profile` | `ProfileComponent` | `Work` | User Profile page with Photo/Avatar uploader & URL/color picker, editable Display Name, and read-only Google Cloud IAP Email (`input[aria-label="Profile email"][readonly]`). |
| `/signed-out` | `SignedOutComponent` | Standalone | Dedicated signed-out & account-chooser screen (`[data-testid="signed-out-card"]`), hiding the 3-column workspace shell while signed out and exposing IAP re-login / Google account chooser links (`!devMode`) or local `X-Dev-User-*` persona switchers (`devMode`). |

---

## 2. Stable DOM Selectors & `aria-label` Hooks

### App Shell & Navigation (`frontend/src/app/app.component.ts`)
- **Top Header Brand & Global Actions**:
  - Brand title text: `Jynko` (must **not** contain a dropdown caret `⌄` or workspace switcher).
  - Non-production environment badge (`ENV=dev` or `ENV=stg`, omitted in `ENV=prd`): `header [data-testid="env-badge"]` (`DEV` or `STG`, with browser tab `document.title` prefixed as `[DEV] Jynko` or `[STG] Jynko`).
  - Global search input: `header input[type="search"]`.
  - Top-bar page refresh button (checks SW updates & reloads app): `header button[data-testid="app-refresh-btn"][aria-label="Refresh page"]` (with update indicator dot `[data-testid="app-refresh-update-dot"]` when a new version is ready).
  - Pull / click-and-drag-down refresh indicator pill: `[data-testid="pull-to-refresh-indicator"]` (`data-pull-ready="true"` once threshold is reached).
  - Mobile hamburger button (`< 768px`): `button[aria-label="Toggle mobile navigation"]`.
- **Mobile Navigation (`< 768px`)**:
  - Off-canvas navigation drawer container: `[data-testid="app-navigation-drawer"]`
  - Off-canvas backdrop overlay: `[data-testid="mobile-nav-backdrop"]`
  - Work Items mobile filter toggle: `button[aria-label="Toggle mobile filters"]`
  - Wiki mobile collections drawer trigger: `button[aria-label="Browse wiki collections"]`
  - Settings mobile scope switcher: `[data-testid="mobile-settings-scope-switcher"]`
- **Left Icon Rail (`68px`)**:
  - `Work` section button: `aside button` with text `Work`.
  - `Wiki` section button: `aside button` with text `Wiki`.
  - `Settings` button: `aside button` with text `Settings`.
- **Work Secondary Panel (`w-64`)**:
  - Collapse secondary panel button: `button[aria-label="Collapse secondary panel"]`
  - Expand secondary panel button (shown when collapsed): `button[aria-label="Expand secondary panel"]`
  - Top-left project breadcrumb (Work Items, Cycles, Epics, Modules):
    ```css
    main nav[aria-label="Breadcrumbs"] a[data-testid="page-project-link"]
    ```
  - New work item button: `[data-testid="open-create-work-item-btn"]` (`button` containing `New work item`).
  - Create work item modal (opened when clicking `[data-testid="open-create-work-item-btn"]` or `+ Add work item`):
    - Modal title: `Create work item`
    - Inputs: `input#quick-create-title`, `textarea#quick-create-description`, `input#quick-create-estimate` (defaults to `1`), `input#quick-create-start-date`, `input#quick-create-due-date`
    - Dropdowns: `button[aria-label="New work item status"]`, `button[aria-label="New work item priority"]`, `button[aria-label="New work item cycle"]` (`No cycle` + project cycles)
  - Navigation links: `a[href="/home"]`, `a[href="/my-work"]`, `a[href="/stickies"]`.
  - Create new project `+` icon button at root of `Projects`:
    ```css
    button[aria-label="Create new project"]
    ```
  - Create New Project modal (opened when clicking `button[aria-label="Create new project"]`):
    - Modal title: `Create New Project`
    - Inputs: `input[aria-label="New project name"]`, `input[aria-label="New project key prefix"]`, `textarea[aria-label="New project description"]`
    - Submit button: `button[aria-label="Submit create project"]`
  - Project `Overview`, `Work items`, and `Pages` links in left `Projects` tree:
    ```css
    a[aria-label="Open <projectName> overview"]
    a[aria-label="Open <projectName> work items"]
    a[aria-label="Open <projectName> pages in Wiki"]
    ```
  - Wiki Collection row in left `Wiki` panel:
    ```css
    button[aria-label="Collection <collectionName>"]
    ```
  - Stickies controls (`/stickies`):
    - Random sticky button: `button[aria-label="Surprise me random sticky"]`
    - New sticky composer toggle: `button[aria-label="Create new sticky"]`
    - Filter by text input: `input[aria-label="Filter stickies by text"]`
  - Settings controls (`/settings`):
    - Left rail Settings button: `a[aria-label="Open Settings"]`
    - Secondary panel General Settings button: `button[aria-label="Open General App Settings"]`
    - Secondary panel Per-Project Settings buttons: `button[aria-label="Open project settings for <projectName>"]`
    - Scope switcher tabs: `button[aria-label="General App Settings tab"]`, `button[aria-label="Per-Project Settings tab"]`
    - Progressive Web App (PWA) card: `[data-testid="pwa-settings-card"]`, `button[aria-label="Install Jynko PWA from settings"]`, `button[aria-label="Check for PWA updates"]`, `button[aria-label="Clear PWA cache"]`, `input[aria-label="Toggle PWA auto update"]`, `input[aria-label="Toggle PWA install menu shortcut"]`, `input[aria-label="Toggle PWA offline indicator"]`
    - Project Appearance / background colour picker (PLAN-4, Per-Project Settings): `[data-testid="project-appearance-card"] [role="radiogroup"][aria-label="Project background color"]`, options `button[role="radio"][aria-label="Set project background color <Label>"]` (e.g. `Default (Slate)`, `Sky`, `Rose`) with `aria-checked` and `data-color-key` (`DEFAULT`, `SKY`, `ROSE`, ...).
    - Scion AI Agents & Slash Commands card (Per-Project Settings): `[data-testid="scion-settings-card"]`, `input[aria-label="Toggle Scion AI Agents"]`, `input[aria-label="Default Git Repo URL"]`, `input[aria-label="Default Base Branch"]`, `input[aria-label="Scion Project Slug"]`, `input[aria-label="Scion Template"]`, `textarea[aria-label="Implement prompt template"]`, `textarea[aria-label="Review prompt template"]`, `textarea[aria-label="Follow-up prompt template"]`, `button[aria-label="Reset implement prompt template"]`, `button[aria-label="Reset review prompt template"]`, `button[aria-label="Reset follow-up prompt template"]`.
    - Project top bar (Overview, Work items, Cycles, Epics, Modules — PLAN-7): `main [data-testid="project-top-bar"]`. The only element tinted with the project background colour (`.project-bg`, from the `--project-bg` CSS variable set on `main`; `main[data-project-bg="<KEY>"]`). Shared layout: `px-3 sm:px-4 py-2` with a 1px `border-b border-slate-100` hairline.
    - Project page content (below the top bar): `main [data-testid="project-page-content"]` — default `bg-slate-50` canvas with `p-2.5 sm:p-4` padding on every project page.
    - Work Items view/filter toolbar (List/Board/... switcher, Columns, Filters): `main [data-testid="work-items-toolbar"]` — white (`bg-white`), not tinted.
    - User menu PWA install / refresh shortcuts: `button[aria-label="Install Jynko PWA"]` (browser tab), `button[aria-label="Refresh Jynko app"]` (standalone PWA mode)
  - Kanban Board Column Configurator (`/work-items` in `BOARD` mode):
    - Toolbar columns button: `button[aria-label="Configure Kanban board columns"]`
    - Popover dialog: `[role="dialog"][aria-label="Kanban Board Columns Configuration"]`
    - Per-status switch: `input[aria-label="Toggle board column <StatusLabel>"]`
    - Board column container: `[data-board-column="<STATUS>"]`
  - Calendar & Timeline View Navigation (`/work-items` in `CALENDAR` / `TIMELINE` modes):
    - View switcher buttons: `button[aria-label="Switch to <List|Board|Calendar|Timeline|Spreadsheet> view"]`
    - Calendar jump to current month: `button[aria-label="Calendar jump to current month"]`
    - Timeline window controls: `button[aria-label="Timeline previous window"]`, `button[aria-label="Timeline jump to today"]`, `button[aria-label="Timeline next window"]`
- **Styled Custom Dropdowns (`frontend/src/app/shared/components/dropdown/dropdown.component.ts`)**:
  - Component selector: `app-dropdown`
  - Trigger button: `button[aria-haspopup="listbox"]`
  - Accessible labels by view:
    - My Work status filter: `button[aria-label="Filter My Work by Status"]`
    - My Work priority filter: `button[aria-label="Filter My Work by Priority"]`
    - My Work project filter: `button[aria-label="Filter My Work by Project"]`
    - Work Items status filter: `button[aria-label="Filter by Status"]`
    - Work Items priority filter: `button[aria-label="Filter by Priority"]`
    - New Work Item modal selectors: `button[aria-label="New work item status"]`, `button[aria-label="New work item priority"]`, `textarea[aria-label="New work item description"]`, `input[aria-label="New work item start date"]`, `input[aria-label="New work item due date"]`
    - Work Item detail drawer & full page selectors: `button[aria-label="Open work item full page"]`, `a[aria-label="Back to Work Items"]`, `input[aria-label="Work item start date"]`, `input[aria-label="Work item due date"]`, `input[aria-label="Work item estimate"]`, `[data-testid="scion-agent-bar"]`, `button[aria-label="Refresh Scion agent status"]`, `[data-testid="scion-slash-menu"]`, `textarea[aria-label="Work item comment"]`, `button[aria-label="Toggle comment markdown preview"]`, `button[aria-label="Post comment"]`, `button[aria-label="Edit comment"]`, `textarea[aria-label="Edit comment markdown"]`, `button[aria-label="Toggle editing comment preview"]`, `button[aria-label="Delete comment"]`, `[data-testid="work-item-description-container"]`, `button[aria-label="Copy description"]` (icon-only, top-right of description), `button[aria-label="Copy comment"]` (icon-only, top-right of each comment)
    - Project member role selector: `button[aria-label="Select member role"]`
    - User account card, switcher & profile selectors: `button[aria-label="Open user menu"]`, `[data-testid="user-dropdown-card"]`, `button[aria-label="Open user profile"]`, `button[aria-label="Switch user account"]`, `[data-testid="dev-account-switcher-list"]`, `button[aria-label^="Switch to account "]`, `button[aria-label="Use another account"]`, `button[aria-label="Sign out"]`, `[data-testid="signed-out-card"]`, `button[aria-label="Sign in with Google Cloud IAP"]`, `button[aria-label^="Sign in as "]`, `input[aria-label="Profile display name"]`, `input[aria-label="Profile email"][readonly]`, `input[aria-label="Profile avatar URL"]`
    - Project Overview edit selector: `button[aria-label="Edit project"]`
    - Project Labels CRUD selectors: `input[aria-label^="Edit label name "]`, `input[aria-label^="Edit label color "]`, `button[aria-label^="Remove label "]`, `input[aria-label="New label name"]`
    - Wiki toolbar selectors: `button[aria-label="Text block style"]`, `button[aria-label="Text color"]`, `button[aria-label="Save wiki page"]`
    - Cycles CRUD selectors (`/projects/:projectId/cycles`): `button[aria-label="Create new cycle"]`, `button[aria-label^="Edit cycle "]`, `button[aria-label^="Delete cycle "]`, `button[aria-label^="Toggle active status for "]`, `input[aria-label="Edit cycle name"]`
    - Epics CRUD selectors (`/projects/:projectId/epics`): `button[aria-label="Create new epic"]`, `button[aria-label^="Edit epic "]`, `button[aria-label^="Delete epic "]`, `button[aria-label^="Epic status for "]`, `input[aria-label="Edit epic name"]`
    - Modules CRUD selectors (`/projects/:projectId/modules`): `button[aria-label="Create new module"]`, `button[aria-label^="Edit module "]`, `button[aria-label^="Delete module "]`, `button[aria-label^="Module lead for "]`, `input[aria-label="Edit module name"]`, `input[aria-label="Edit module git repo URL"]`, `input[aria-label="Edit module base branch"]`
  - Open floating menu container: `[role="listbox"]`
  - Menu options: `[role="option"]`

---

## 3. Non-Negotiable UI Invariants to Assert

1. **No `Drafts` Anywhere**:
   - The `Work` secondary panel (`aside.w-64`) must never contain `"Drafts"`.
2. **No Native `<select>` Elements**:
   - `page.MustElements("select")` should return `0` across all views; all option lists must use `<app-dropdown>` (`[role="listbox"]`).
3. **Top-Left Application Icon Only**:
   - Top-left brand header shows the custom geometric `Jynko` SVG icon + `Jynko` title, with no dropdown caret (`⌄`).
4. **Two-Section Shell Separation**:
   - In `Work` mode (`/home`, `/my-work`, `/work-items`, `/projects`, etc.), the secondary panel shows `Work` navigation and the `Projects` tree.
   - In `Wiki` mode (`/wiki`), the secondary panel switches to `Wiki` (`+ New page`, `Collections`, nested `Pages` tree) and the right pane displays only the Page View & Editor.

    - Per-Project RBAC Members inline display name edit (PLAN-10, ADMIN only, saves on blur/Enter, email stays read-only): `input[aria-label="Edit member display name"][data-member-email="<email>"]`