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

---

## 2. Stable DOM Selectors & `aria-label` Hooks

### App Shell & Navigation (`frontend/src/app/app.component.ts`)
- **Top Header Brand**:
  - Brand title text: `Jynko` (must **not** contain a dropdown caret `⌄` or workspace switcher).
  - Global search input: `header input[type="search"]`.
- **Left Icon Rail (`68px`)**:
  - `Work` section button: `aside button` with text `Work`.
  - `Wiki` section button: `aside button` with text `Wiki`.
  - `Settings` button: `aside button` with text `Settings`.
- **Work Secondary Panel (`w-64`)**:
  - Collapse secondary panel button: `button[aria-label="Collapse secondary panel"]`
  - Expand secondary panel button (shown when collapsed): `button[aria-label="Expand secondary panel"]`
  - New work item button: `button` containing `New work item`.
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
  - Kanban Board Column Configurator (`/work-items` in `BOARD` mode):
    - Toolbar columns button: `button[aria-label="Configure Kanban board columns"]`
    - Popover dialog: `[role="dialog"][aria-label="Kanban Board Columns Configuration"]`
    - Per-status switch: `input[aria-label="Toggle board column <StatusLabel>"]`
    - Board column container: `[data-board-column="<STATUS>"]`
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
    - Work Item detail drawer & full page selectors: `button[aria-label="Open work item full page"]`, `a[aria-label="Back to Work Items"]`, `input[aria-label="Work item start date"]`, `input[aria-label="Work item due date"]`, `input[aria-label="Work item estimate"]`
    - Project member role selector: `button[aria-label="Select member role"]`
    - Wiki toolbar selectors: `button[aria-label="Text block style"]`, `button[aria-label="Text color"]`
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
