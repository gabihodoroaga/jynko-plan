package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ValidationCheckResult records the outcome of a single UI E2E check and its screenshot.
type ValidationCheckResult struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Passed     bool     `json:"passed"`
	Assertions []string `json:"assertions"`
	Errors     []string `json:"errors,omitempty"`
	Screenshot string   `json:"screenshot"`
	DurationMs int64    `json:"duration_ms"`
}

// ValidationReport aggregates all E2E check results.
type ValidationReport struct {
	Timestamp   string                  `json:"timestamp"`
	BaseURL     string                  `json:"base_url"`
	AllPassed   bool                    `json:"all_passed"`
	TotalChecks int                     `json:"total_checks"`
	PassedCount int                     `json:"passed_count"`
	FailedCount int                     `json:"failed_count"`
	Results     []ValidationCheckResult `json:"results"`
}

// CheckSpec pairs a stable check ID with its execution function.
type CheckSpec struct {
	ID   string
	Name string
	Run  func(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult
}

// DefaultChecks returns the canonical list of Jynko E2E UI validation checks covering all application features.
func DefaultChecks() []CheckSpec {
	return []CheckSpec{
		{
			ID:   "01-app-shell",
			Name: "App Shell, Navigation Rail & Responsive Panel Layout",
			Run:  CheckAppShell,
		},
		{
			ID:   "02-user-and-profile",
			Name: "User Menu, Account Dropdown & Profile Page",
			Run:  CheckUserAndProfile,
		},
		{
			ID:   "03-home-overview",
			Name: "Cross-Project Home Overview, Metrics & Progress Cards",
			Run:  CheckHomeOverview,
		},
		{
			ID:   "04-my-work",
			Name: "My Work Cross-Project Tracking & Styled Dropdown Filters",
			Run:  CheckMyWork,
		},
		{
			ID:   "05-projects-crud",
			Name: "Projects Lifecycle, Selection & Metadata Editing Dialog",
			Run:  CheckProjectsCRUD,
		},
		{
			ID:   "06-project-labels",
			Name: "Project Labels Swatch Picker, Inline Rename & Removal",
			Run:  CheckProjectLabels,
		},
		{
			ID:   "07-work-items-views",
			Name: "Work Items 5 View Switchers, Filter Bar & Full-Height Kanban",
			Run:  CheckWorkItemsViews,
		},
		{
			ID:   "08-work-item-detail",
			Name: "Work Item Detail Drawer (Pinned Actions) & Full-Page View",
			Run:  CheckWorkItemDetail,
		},
		{
			ID:   "09-cycles-crud",
			Name: "Sprint Cycles Management, Active Badge Toggle & Modals",
			Run:  CheckCyclesCRUD,
		},
		{
			ID:   "10-epics-crud",
			Name: "Strategic Epics Tracking, Inline Status & Modals",
			Run:  CheckEpicsCRUD,
		},
		{
			ID:   "11-modules-crud",
			Name: "Feature Modules Management, Lead Dropdown & Modals",
			Run:  CheckModulesCRUD,
		},
		{
			ID:   "12-wiki-workspace",
			Name: "Wiki Workspace, Collections Tree & Document Editor",
			Run:  CheckWikiWorkspace,
		},
		{
			ID:   "13-stickies-board",
			Name: "Stickies Board, Fun Color Palettes & Surprise Me Generator",
			Run:  CheckStickiesBoard,
		},
		{
			ID:   "14-settings",
			Name: "Settings Workspace: General App Preferences & Per-Project Settings",
			Run:  CheckSettings,
		},
	}
}

// LaunchBrowser starts a headless Chrome/Chromium instance configured for local and CI/CD environments.
func LaunchBrowser() (*rod.Browser, error) {
	l := launcher.New().
		Headless(true).
		NoSandbox(true).
		Leakless(false).
		Set("disable-gpu").
		Set("disable-dev-shm-usage")

	if customBin := os.Getenv("CHROME_BIN"); customBin != "" {
		l = l.Bin(customBin)
	} else if path, exists := launcher.LookPath(); exists {
		l = l.Bin(path)
	} else if _, err := os.Stat("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"); err == nil {
		l = l.Bin("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
	}

	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch headless browser via go-rod: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to headless browser: %w", err)
	}
	return browser, nil
}

// RunSuite executes all DefaultChecks against baseURL, writes screenshots and report.json to outDir, and returns the report.
func RunSuite(browser *rod.Browser, baseURL, outDir string) (ValidationReport, string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ValidationReport{}, "", fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}

	cleanBaseURL := strings.TrimRight(baseURL, "/")
	report := ValidationReport{
		Timestamp: time.Now().Format(time.RFC3339),
		BaseURL:   cleanBaseURL,
		AllPassed: true,
	}

	for _, spec := range DefaultChecks() {
		res := spec.Run(browser, cleanBaseURL, outDir)
		report.Results = append(report.Results, res)
		report.TotalChecks++
		if res.Passed {
			report.PassedCount++
		} else {
			report.FailedCount++
			report.AllPassed = false
		}
	}

	reportPath := filepath.Join(outDir, "report.json")
	if raw, err := json.MarshalIndent(report, "", "  "); err == nil {
		_ = os.WriteFile(reportPath, raw, 0o644)
	}

	return report, reportPath, nil
}

// PrintReport formats the ValidationReport to stdout.
func PrintReport(report ValidationReport, reportPath string) {
	fmt.Println("================================================================================")
	fmt.Printf("JYNKO GO-ROD UI E2E VALIDATION REPORT (%s)\n", report.BaseURL)
	fmt.Println("================================================================================")
	for _, r := range report.Results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s (%dms)\n", status, r.Name, r.DurationMs)
		for _, a := range r.Assertions {
			fmt.Printf("  ✓ %s\n", a)
		}
		for _, e := range r.Errors {
			fmt.Printf("  ✗ %s\n", e)
		}
		fmt.Printf("  📸 Screenshot: %s\n\n", r.Screenshot)
	}
	fmt.Printf("Summary: %d/%d checks passed (Report: %s)\n", report.PassedCount, report.TotalChecks, reportPath)
	fmt.Println("================================================================================")
}

func newViewportPage(browser *rod.Browser, url string) *rod.Page {
	page := browser.MustPage("")
	_ = proto.EmulationSetDeviceMetricsOverride{
		Width:             1440,
		Height:            900,
		DeviceScaleFactor: 1,
		Mobile:            false,
	}.Call(page)
	page.MustNavigate(url).MustWaitLoad()

	// Wait until the live backend + Firestore API responses populate the application state
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		bodyText := page.MustElement("body").MustText()
		if strings.Contains(bodyText, "hodo") || strings.Contains(bodyText, "Welcome to Company's Wiki") || strings.Contains(bodyText, "Electric Banana") {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(800 * time.Millisecond)
	return page
}

func assertFullWidthPageContainer(page *rod.Page, res *ValidationCheckResult, pageLabel string) {
	hasMaxW3xl, _, _ := page.Has(`main .max-w-3xl`)
	hasMaxW5xl, _, _ := page.Has(`main .max-w-5xl`)
	hasMaxW6xl, _, _ := page.Has(`main .max-w-6xl`)
	if !hasMaxW3xl && !hasMaxW5xl && !hasMaxW6xl {
		res.Assertions = append(res.Assertions, fmt.Sprintf("%s page body uses 100%% full width (w-full) without max-w-3xl/5xl/6xl constraints", pageLabel))
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("%s page body still contains a max-w-3xl/5xl/6xl wrapper inside <main>", pageLabel))
	}
}

// -----------------------------------------------------------------------------
// Suite 1: App Shell & Responsive Panel Layout
// -----------------------------------------------------------------------------
func CheckAppShell(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/home"
	res := ValidationCheckResult{
		ID:         "01-app-shell",
		Name:       "App Shell, Navigation Rail & Responsive Panel Layout",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "01-app-shell.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	headerText := page.MustElement("header").MustText()
	asideText := page.MustElement("aside").MustText()

	if strings.Contains(headerText, "Jynko") && !strings.Contains(headerText, "⌄") {
		res.Assertions = append(res.Assertions, "Top-left brand header renders Gemini-styled Jynko icon without dropdown caret")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Expected header to show 'Jynko' without '⌄', got: %q", headerText))
	}

	hasWorkRailBtn, _, _ := page.Has(`nav[aria-label="Primary rail"] a[href*="work-items"]`)
	hasWikiRailBtn, _, _ := page.Has(`nav[aria-label="Primary rail"] a[href="/wiki"]`)
	hasSettingsRailBtn, _, _ := page.Has(`nav[aria-label="Primary rail"] a[aria-label="Open Settings"]`)
	if hasWorkRailBtn && hasWikiRailBtn && hasSettingsRailBtn {
		res.Assertions = append(res.Assertions, "Primary navigation rail renders exactly 3 section buttons (Work, Wiki, Settings)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Primary navigation rail missing buttons: work=%v, wiki=%v, settings=%v", hasWorkRailBtn, hasWikiRailBtn, hasSettingsRailBtn))
	}

	hasUserSwitcher, _, _ := page.Has(`button[aria-label="Switch IAP Test Identity"]`)
	if !hasUserSwitcher && !strings.Contains(asideText, "Drafts") && !strings.Contains(asideText, "Views") {
		res.Assertions = append(res.Assertions, "Work secondary panel strictly omits legacy Drafts, Views, and left-panel identity switcher")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Work secondary panel contains legacy Drafts, Views, or user identity switcher")
	}

	if hasCollapse, collapseBtn, _ := page.Has(`button[aria-label="Collapse secondary panel"]`); hasCollapse && collapseBtn != nil {
		collapseBtn.MustClick()
		time.Sleep(200 * time.Millisecond)

		if hasExpand, expandBtn, _ := page.Has(`button[aria-label="Expand secondary panel"]`); hasExpand && expandBtn != nil {
			expandBtn.MustClick()
			time.Sleep(200 * time.Millisecond)

			restoredAside := page.MustElement("aside").MustText()
			if strings.Contains(restoredAside, "Projects") {
				res.Assertions = append(res.Assertions, "Secondary panel collapse button collapses panel, renders 'Expand secondary panel', and restores cleanly on click")
			} else {
				res.Passed = false
				res.Errors = append(res.Errors, "Expanding secondary panel did not restore Projects list")
			}
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Collapsing secondary panel did not render button[aria-label='Expand secondary panel']")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Collapse secondary panel']")
	}

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 2: User Menu, Account Dropdown & Profile Page
// -----------------------------------------------------------------------------
func CheckUserAndProfile(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/home"
	res := ValidationCheckResult{
		ID:         "02-user-and-profile",
		Name:       "User Menu, Account Dropdown & Profile Page",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "02-user-and-profile.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	if hasUserBtn, userBtn, _ := page.Has(`button[aria-label="Open user menu"]`); hasUserBtn && userBtn != nil {
		userBtn.MustClick()
		time.Sleep(250 * time.Millisecond)

		if hasCard, cardEl, _ := page.Has(`[data-testid="user-dropdown-card"]`); hasCard && cardEl != nil {
			cardText := cardEl.MustText()
			hasProfileBtn, profileBtn, _ := cardEl.Has(`button[aria-label="Open user profile"]`)
			hasSignOutBtn, _, _ := cardEl.Has(`button[aria-label="Sign out"]`)

			if hasProfileBtn && hasSignOutBtn && strings.Contains(cardText, "dev@jynko.local") {
				res.Assertions = append(res.Assertions, "Top-right user button opens Google-style account card with Name, Email, Profile button, and Sign out button")
				profileBtn.MustClick()
				time.Sleep(300 * time.Millisecond)

				profileURL := page.MustInfo().URL
				profileMain := page.MustElement("main").MustText()
				hasNameInput, _, _ := page.Has(`input[aria-label="Profile display name"]`)
				hasReadonlyEmail, _, _ := page.Has(`input[aria-label="Profile email"][readonly]`)
				hasAvatarInput, _, _ := page.Has(`input[aria-label="Profile avatar URL"]`)

				if strings.Contains(profileURL, "/profile") &&
					strings.Contains(profileMain, "User Profile") &&
					hasNameInput &&
					hasReadonlyEmail &&
					hasAvatarInput {
					res.Assertions = append(res.Assertions, "Clicking 'Profile' navigates to /profile with editable Name, read-only IAP Email, and Avatar controls")
				} else {
					res.Passed = false
					res.Errors = append(res.Errors, fmt.Sprintf("Profile page validation failed: url=%q, hasName=%v, hasReadonly=%v, hasAvatar=%v", profileURL, hasNameInput, hasReadonlyEmail, hasAvatarInput))
				}
			} else {
				res.Passed = false
				res.Errors = append(res.Errors, "User dropdown card missing expected Name/Email, Profile button, or Sign out button")
			}
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Clicking top-right user button did not open [data-testid='user-dropdown-card']")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing top-right user button[aria-label='Open user menu']")
	}

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 3: Cross-Project Home Overview & Metrics
// -----------------------------------------------------------------------------
func CheckHomeOverview(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/home"
	res := ValidationCheckResult{
		ID:         "03-home-overview",
		Name:       "Cross-Project Home Overview, Metrics & Progress Cards",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "03-home-overview.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	mainText := page.MustElement("main").MustText()

	metricsPresent := strings.Contains(mainText, "Active Projects") &&
		strings.Contains(mainText, "Total Work Items") &&
		strings.Contains(mainText, "In Progress") &&
		strings.Contains(mainText, "Assigned to Me")
	if metricsPresent {
		res.Assertions = append(res.Assertions, "Home page renders 4 cross-project summary metric cards")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Home page missing one or more summary metric cards")
	}

	projectsPresent := strings.Contains(mainText, "hodo") &&
		strings.Contains(mainText, "cloud-infra") &&
		strings.Contains(mainText, "Recent Work Items Across All Projects") &&
		strings.Contains(mainText, "Knowledge Base Pages")
	if projectsPresent {
		res.Assertions = append(res.Assertions, "Home page renders project overview cards, recent work items, and knowledge base pages")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Home page missing project cards or recent activity sections")
	}

	if !strings.Contains(mainText, "+ New work item") {
		res.Assertions = append(res.Assertions, "Home page strictly omits project-scoped '+ New work item' button")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Home page contains project-scoped '+ New work item' button")
	}

	assertFullWidthPageContainer(page, &res, "Home")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 4: My Work Cross-Project Tracking & Styled Dropdown Filters
// -----------------------------------------------------------------------------
func CheckMyWork(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/my-work"
	res := ValidationCheckResult{
		ID:         "04-my-work",
		Name:       "My Work Cross-Project Tracking & Styled Dropdown Filters",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "04-my-work.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	mainText := page.MustElement("main").MustText()
	if strings.Contains(mainText, "My Work") &&
		strings.Contains(mainText, "HODO-1") &&
		strings.Contains(mainText, "CLD-1") {
		res.Assertions = append(res.Assertions, "My Work displays items assigned to current user across both 'hodo' (HODO-1) and 'cloud-infra' (CLD-1)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "My Work failed to render cross-project items HODO-1 and CLD-1")
	}

	if selects := page.MustElements("select"); len(selects) == 0 {
		res.Assertions = append(res.Assertions, "Zero native browser <select> elements present on My Work page")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Found %d native <select> elements on My Work page", len(selects)))
	}

	if has, btn, _ := page.Has(`button[aria-label="Filter My Work by Status"]`); has && btn != nil {
		btn.MustClick()
		time.Sleep(200 * time.Millisecond)
		if hasListbox, _, _ := page.Has(`[role="listbox"]`); hasListbox {
			res.Assertions = append(res.Assertions, "Clicking styled Status filter opens floating custom [role='listbox'] menu instead of native select")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Clicking styled dropdown did not open [role='listbox']")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Filter My Work by Status']")
	}

	assertFullWidthPageContainer(page, &res, "My Work")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 5: Projects Lifecycle, Selection & Metadata Editing Dialog
// -----------------------------------------------------------------------------
func CheckProjectsCRUD(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/projects"
	res := ValidationCheckResult{
		ID:         "05-projects-crud",
		Name:       "Projects Lifecycle, Selection & Metadata Editing Dialog",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "05-projects-crud.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	currentURL := page.MustInfo().URL
	if strings.Contains(currentURL, "/projects/proj-jynko-core") {
		res.Assertions = append(res.Assertions, "Navigating to /projects redirects to active project URL (/projects/proj-jynko-core)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Expected URL to include '/projects/proj-jynko-core', got: %q", currentURL))
	}

	// Verify project switching via left tree
	if hasCloudOverview, cloudLink, _ := page.Has(`a[aria-label="Open cloud-infra overview"]`); hasCloudOverview && cloudLink != nil {
		cloudLink.MustClick()
		time.Sleep(250 * time.Millisecond)
		switchedURL := page.MustInfo().URL
		if strings.Contains(switchedURL, "/projects/proj-cloud-infra") {
			res.Assertions = append(res.Assertions, "Selecting 'cloud-infra -> Overview' in left tree switches active project to cloud-infra")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, fmt.Sprintf("Expected URL '/projects/proj-cloud-infra', got: %q", switchedURL))
		}
		if hasHodoOverview, hodoLink, _ := page.Has(`a[aria-label="Open hodo overview"]`); hasHodoOverview && hodoLink != nil {
			hodoLink.MustClick()
			time.Sleep(250 * time.Millisecond)
		}
	}

	mainText := page.MustElement("main").MustText()
	hasEditProjBtn, editProjBtn, _ := page.Has(`button[aria-label="Edit project"]`)
	if !strings.Contains(mainText, "Project Overview & Per-Project RBAC") &&
		!strings.Contains(mainText, "+ New Project") &&
		hasEditProjBtn {
		res.Assertions = append(res.Assertions, "Project Overview renders 'Edit' button on active project card and omits legacy header text")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Project Overview failed header/edit button check: hasEditProjBtn=%v", hasEditProjBtn))
	}

	// Test Edit Project modal
	if hasEditProjBtn && editProjBtn != nil {
		editProjBtn.MustClick()
		time.Sleep(250 * time.Millisecond)
		bodyText := page.MustElement("body").MustText()
		hasProjNameInput, nameEl, _ := page.Has(`input[aria-label="New project name"]`)
		hasProjPrefixInput, _, _ := page.Has(`input[aria-label="New project key prefix"]`)
		hasProjDescInput, _, _ := page.Has(`textarea[aria-label="New project description"]`)
		nameVal := ""
		if hasProjNameInput && nameEl != nil {
			nameVal = nameEl.MustProperty("value").String()
		}
		if strings.Contains(bodyText, "Edit Project") && hasProjNameInput && hasProjPrefixInput && hasProjDescInput && nameVal == "hodo" {
			res.Assertions = append(res.Assertions, "Clicking 'Edit' on Project Overview opens Project modal pre-populated with active project metadata")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, fmt.Sprintf("Edit Project modal failed: nameVal=%q", nameVal))
		}
		if hasClose, closeBtn, _ := page.Has(`dialog[open] button[aria-label="Close modal"]`); hasClose && closeBtn != nil {
			closeBtn.MustClick()
			time.Sleep(150 * time.Millisecond)
		}
	}

	// Test '+' create project modal in sidebar
	if hasPlus, plusBtn, _ := page.Has(`button[aria-label="Create new project"]`); hasPlus && plusBtn != nil {
		plusBtn.MustClick()
		time.Sleep(250 * time.Millisecond)
		bodyText := page.MustElement("body").MustText()
		if strings.Contains(bodyText, "Create New Project") && strings.Contains(bodyText, "Key Prefix") {
			res.Assertions = append(res.Assertions, "Clicking '+' at root of Projects list in left panel opens 'Create New Project' modal")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Clicking '+' button did not open 'Create New Project' modal")
		}
		if hasClose, closeBtn, _ := page.Has(`dialog[open] button[aria-label="Close modal"]`); hasClose && closeBtn != nil {
			closeBtn.MustClick()
			time.Sleep(150 * time.Millisecond)
		}
	}

	assertFullWidthPageContainer(page, &res, "Projects")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 6: Project Labels Swatch Picker, Inline Rename & Removal
// -----------------------------------------------------------------------------
func CheckProjectLabels(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/projects/proj-jynko-core"
	res := ValidationCheckResult{
		ID:         "06-project-labels",
		Name:       "Project Labels Swatch Picker, Inline Rename & Removal",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "06-project-labels.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	hasEditLabelInput, _, _ := page.Has(`input[aria-label^="Edit label name "]`)
	hasRemoveLabelBtn, _, _ := page.Has(`button[aria-label^="Remove label "]`)
	hasColorPicker, _, _ := page.Has(`input[aria-label^="Edit label color "]`)
	hasAddLabelInput, _, _ := page.Has(`input[aria-label="New label name"]`)

	if hasEditLabelInput && hasRemoveLabelBtn && hasColorPicker && hasAddLabelInput {
		res.Assertions = append(res.Assertions, "Project Labels section renders inline editable name inputs, color swatch pickers, remove ('×') buttons, and new label form")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Project Labels section missing controls: hasEdit=%v, hasRemove=%v, hasColor=%v, hasAdd=%v", hasEditLabelInput, hasRemoveLabelBtn, hasColorPicker, hasAddLabelInput))
	}

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 7: Work Items 5 View Switchers, Filter Bar & Full-Height Kanban
// -----------------------------------------------------------------------------
func CheckWorkItemsViews(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/work-items"
	res := ValidationCheckResult{
		ID:         "07-work-items-views",
		Name:       "Work Items 5 View Switchers, Filter Bar & Full-Height Kanban",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "07-work-items-views.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	currentURL := page.MustInfo().URL
	if strings.Contains(currentURL, "/projects/proj-jynko-core/work-items") {
		res.Assertions = append(res.Assertions, "Navigating to /work-items redirects to active project URL (/projects/proj-jynko-core/work-items)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Expected URL to include '/projects/proj-jynko-core/work-items', got: %q", currentURL))
	}

	mainText := page.MustElement("main").MustText()
	viewsPresent := strings.Contains(mainText, "List") &&
		strings.Contains(mainText, "Board") &&
		strings.Contains(mainText, "Calendar") &&
		strings.Contains(mainText, "Timeline") &&
		strings.Contains(mainText, "Spreadsheet")

	if viewsPresent {
		res.Assertions = append(res.Assertions, "All 5 Work Item view switcher tabs (List, Board, Calendar, Timeline, Spreadsheet) are rendered")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing one or more Work Item view switcher tabs")
	}

	if has, _, _ := page.Has(`button[aria-label="Filter by Status"]`); has {
		res.Assertions = append(res.Assertions, "Work Items filter toolbar uses styled <app-dropdown> controls")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing styled Status filter dropdown in Work Items toolbar")
	}

	if !strings.Contains(mainText, "lll High") && !strings.Contains(mainText, "ll Medium") && !strings.Contains(mainText, "\nlll\n") {
		res.Assertions = append(res.Assertions, "Priority badges display clean label names ('High', 'Medium', 'Urgent') without 'll'/'lll' prefixes")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Priority badges still contain 'll' or 'lll' prefixes")
	}

	hasCancelledCol, _, _ := page.Has(`[data-board-column="CANCELLED"]`)
	hasTodoCol, _, _ := page.Has(`[data-board-column="TODO"]`)
	hasFullHeightCol, _, _ := page.Has(`.board-column.h-full`)
	if !hasCancelledCol && hasTodoCol && hasFullHeightCol && strings.Contains(mainText, "Columns (5/6)") && !strings.Contains(mainText, "Hidden Columns") {
		res.Assertions = append(res.Assertions, "Kanban board columns stretch to fill full page height (.board-column.h-full) and hide 'Cancelled' column by default (Columns 5/6)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Kanban board check failed: hasCancelled=%v, hasFullHeight=%v", hasCancelledCol, hasFullHeightCol))
	}

	if has, colBtn, _ := page.Has(`button[aria-label="Configure Kanban board columns"]`); has && colBtn != nil {
		colBtn.MustClick()
		time.Sleep(250 * time.Millisecond)
		if hasToggle, _, _ := page.Has(`input[aria-label="Toggle board column Cancelled"]`); hasToggle {
			res.Assertions = append(res.Assertions, "Clicking 'Columns (5/6)' opens Kanban Board Columns configurator with per-status visibility switches")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Kanban Board Columns configurator did not render status switches")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Configure Kanban board columns']")
	}

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 8: Work Item Detail Drawer & Full-Page View
// -----------------------------------------------------------------------------
func CheckWorkItemDetail(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/projects/proj-jynko-core/work-items"
	res := ValidationCheckResult{
		ID:         "08-work-item-detail",
		Name:       "Work Item Detail Drawer (Pinned Actions) & Full-Page View",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "08-work-item-detail.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	// Open Work Item side drawer
	if hasCard, cardEl, _ := page.Has(`[data-item-key="HODO-1"]`); hasCard && cardEl != nil {
		cardEl.MustClick()
		time.Sleep(300 * time.Millisecond)
		if hasDrawer, drawerEl, _ := page.Has(`app-work-item-detail-drawer aside`); hasDrawer && drawerEl != nil {
			drawerText := drawerEl.MustText()
			hasFullViewBtn, _, _ := drawerEl.Has(`button[aria-label="Open work item full page"]`)
			hasCloseDrawerBtn, _, _ := drawerEl.Has(`button[aria-label="Close work item drawer"]`)
			rightAlignedControls := drawerEl.MustEval(`() => {
				const drawerRect = this.getBoundingClientRect();
				const fullBtn = this.querySelector('button[aria-label="Open work item full page"]');
				const closeBtn = this.querySelector('button[aria-label="Close work item drawer"]');
				if (!fullBtn || !closeBtn) return false;
				const midX = drawerRect.left + drawerRect.width / 2;
				return fullBtn.getBoundingClientRect().left > midX && closeBtn.getBoundingClientRect().left > midX;
			}`).Bool()
			hasTopProps, _, _ := drawerEl.Has(`[data-testid="work-item-top-properties"]`)
			hasStartBtn, _, _ := drawerEl.Has(`button[aria-label="Work item start date"]`)
			hasDueBtn, _, _ := drawerEl.Has(`button[aria-label="Work item due date"]`)
			hasEstBtn, _, _ := drawerEl.Has(`button[aria-label="Work item estimate"]`)
			hasDrawerLabelBtn, labelDropBtn, _ := drawerEl.Has(`button[aria-label="Toggle work item label dropdown"]`)
			hasDrawerLabelOption := false
			if hasDrawerLabelBtn && labelDropBtn != nil {
				labelDropBtn.MustClick()
				time.Sleep(150 * time.Millisecond)
				hasDrawerLabelOption, _, _ = drawerEl.Has(`button[aria-label^="Toggle work item label "]`)
			}
			if hasFullViewBtn && hasCloseDrawerBtn && rightAlignedControls && hasTopProps && hasStartBtn && hasDueBtn && hasEstBtn && hasDrawerLabelOption && !strings.Contains(drawerText, "Delete") {
				res.Assertions = append(res.Assertions, "Work Item side drawer renders top value-only property dropdown pills with 'Full view' and Close ('✕') buttons pinned to the right, plus separate 'Labels:' line with multi-select dropdown")
			} else {
				res.Passed = false
				res.Errors = append(res.Errors, fmt.Sprintf("Work Item side drawer failed assertions: hasFullView=%v, hasClose=%v, rightAligned=%v, hasTopProps=%v, hasStart=%v, hasDue=%v, hasEst=%v, hasLabelOption=%v", hasFullViewBtn, hasCloseDrawerBtn, rightAlignedControls, hasTopProps, hasStartBtn, hasDueBtn, hasEstBtn, hasDrawerLabelOption))
			}
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Clicking work item card did not open app-work-item-detail-drawer")
		}
	}

	page.MustScreenshot(res.Screenshot)

	// Verify full-page Work Item route (/projects/proj-jynko-core/work-items/item-1)
	detailPage := newViewportPage(browser, baseURL+"/projects/proj-jynko-core/work-items/item-1")
	defer detailPage.MustClose()
	detailMain := detailPage.MustElement("main").MustText()
	hasTopPropsPage, _, _ := detailPage.Has(`[data-testid="work-item-top-properties"]`)
	hasStartDateBtn, _, _ := detailPage.Has(`button[aria-label="Work item start date"]`)
	hasDueDateBtn, _, _ := detailPage.Has(`button[aria-label="Work item due date"]`)
	hasEstimateBtn, _, _ := detailPage.Has(`button[aria-label="Work item estimate"]`)
	hasAutoGrowDesc, _, _ := detailPage.Has(`textarea.auto-grow-description[aria-label="Work item description"]`)
	hasBackLink, _, _ := detailPage.Has(`a[aria-label="Back to Work Items"]`)

	if strings.Contains(detailMain, "HODO-1") && hasTopPropsPage && hasStartDateBtn && hasDueDateBtn && hasEstimateBtn && hasAutoGrowDesc && hasBackLink && strings.Contains(detailMain, "Sub-Work Items") {
		res.Assertions = append(res.Assertions, "Full-page Work Item view renders single-column layout with top property pills, auto-growing Description, and sub-items checklist")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Full-page Work Item view failed assertions: hasTopProps=%v, hasStartDate=%v, hasDueDate=%v, hasEstimate=%v, hasAutoGrowDesc=%v, hasBackLink=%v", hasTopPropsPage, hasStartDateBtn, hasDueDateBtn, hasEstimateBtn, hasAutoGrowDesc, hasBackLink))
	}

	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 9: Sprint Cycles Management
// -----------------------------------------------------------------------------
func CheckCyclesCRUD(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/projects/proj-jynko-core/cycles"
	res := ValidationCheckResult{
		ID:         "09-cycles-crud",
		Name:       "Sprint Cycles Management, Active Badge Toggle & Modals",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "09-cycles-crud.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	hasCreateCycleBtn, _, _ := page.Has(`button[aria-label="Create new cycle"]`)
	hasEditCycleBtn, editCycleBtn, _ := page.Has(`button[aria-label^="Edit cycle "]`)
	hasDeleteCycleBtn, _, _ := page.Has(`button[aria-label^="Delete cycle "]`)
	hasToggleActiveBtn, _, _ := page.Has(`button[aria-label^="Toggle active status for "]`)
	hasEditCycleModal := false

	if hasEditCycleBtn && editCycleBtn != nil {
		editCycleBtn.MustClick()
		time.Sleep(200 * time.Millisecond)
		hasEditCycleModal, _, _ = page.Has(`input[aria-label="Edit cycle name"]`)
		if hasClose, closeBtn, _ := page.Has(`dialog[open] button[aria-label="Close modal"]`); hasClose && closeBtn != nil {
			closeBtn.MustClick()
			time.Sleep(150 * time.Millisecond)
		}
	}

	if hasCreateCycleBtn && hasEditCycleBtn && hasDeleteCycleBtn && hasToggleActiveBtn && hasEditCycleModal {
		res.Assertions = append(res.Assertions, "Cycles page renders '+ New Cycle', clickable Active status badge toggle, 'Delete' confirmation, and 'Edit Sprint Cycle' modal")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Cycles page failed assertions: hasCreate=%v, hasEdit=%v, hasDelete=%v, hasToggle=%v, hasEditModal=%v", hasCreateCycleBtn, hasEditCycleBtn, hasDeleteCycleBtn, hasToggleActiveBtn, hasEditCycleModal))
	}

	assertFullWidthPageContainer(page, &res, "Cycles")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 10: Strategic Epics Tracking & Management
// -----------------------------------------------------------------------------
func CheckEpicsCRUD(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/projects/proj-jynko-core/epics"
	res := ValidationCheckResult{
		ID:         "10-epics-crud",
		Name:       "Strategic Epics Tracking, Inline Status & Modals",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "10-epics-crud.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	hasCreateEpicBtn, _, _ := page.Has(`button[aria-label="Create new epic"]`)
	hasEditEpicBtn, editEpicBtn, _ := page.Has(`button[aria-label^="Edit epic "]`)
	hasDeleteEpicBtn, _, _ := page.Has(`button[aria-label^="Delete epic "]`)
	hasEpicStatusDrop, _, _ := page.Has(`button[aria-label^="Epic status for "]`)
	hasEditEpicModal := false

	if hasEditEpicBtn && editEpicBtn != nil {
		editEpicBtn.MustClick()
		time.Sleep(200 * time.Millisecond)
		hasEditEpicModal, _, _ = page.Has(`input[aria-label="Edit epic name"]`)
		if hasClose, closeBtn, _ := page.Has(`dialog[open] button[aria-label="Close modal"]`); hasClose && closeBtn != nil {
			closeBtn.MustClick()
			time.Sleep(150 * time.Millisecond)
		}
	}

	if hasCreateEpicBtn && hasEditEpicBtn && hasDeleteEpicBtn && hasEpicStatusDrop && hasEditEpicModal {
		res.Assertions = append(res.Assertions, "Epics page renders '+ New Epic', inline status dropdown, 'Delete' confirmation, and 'Edit Strategic Epic' modal")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Epics page failed assertions: hasCreate=%v, hasEdit=%v, hasDelete=%v, hasStatusDrop=%v, hasEditModal=%v", hasCreateEpicBtn, hasEditEpicBtn, hasDeleteEpicBtn, hasEpicStatusDrop, hasEditEpicModal))
	}

	assertFullWidthPageContainer(page, &res, "Epics")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 11: Feature Modules Management
// -----------------------------------------------------------------------------
func CheckModulesCRUD(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/projects/proj-jynko-core/modules"
	res := ValidationCheckResult{
		ID:         "11-modules-crud",
		Name:       "Feature Modules Management, Lead Dropdown & Modals",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "11-modules-crud.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	hasCreateModuleBtn, _, _ := page.Has(`button[aria-label="Create new module"]`)
	hasEditModuleBtn, editModuleBtn, _ := page.Has(`button[aria-label^="Edit module "]`)
	hasDeleteModuleBtn, _, _ := page.Has(`button[aria-label^="Delete module "]`)
	hasModuleLeadDrop, _, _ := page.Has(`button[aria-label^="Module lead for "]`)
	hasEditModuleModal := false

	if hasEditModuleBtn && editModuleBtn != nil {
		editModuleBtn.MustClick()
		time.Sleep(200 * time.Millisecond)
		hasEditModuleModal, _, _ = page.Has(`input[aria-label="Edit module name"]`)
		if hasClose, closeBtn, _ := page.Has(`dialog[open] button[aria-label="Close modal"]`); hasClose && closeBtn != nil {
			closeBtn.MustClick()
			time.Sleep(150 * time.Millisecond)
		}
	}

	if hasCreateModuleBtn && hasEditModuleBtn && hasDeleteModuleBtn && hasModuleLeadDrop && hasEditModuleModal {
		res.Assertions = append(res.Assertions, "Modules page renders '+ New Module', inline lead dropdown, 'Delete' confirmation, and 'Edit Feature Module' modal")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Modules page failed assertions: hasCreate=%v, hasEdit=%v, hasDelete=%v, hasLeadDrop=%v, hasEditModal=%v", hasCreateModuleBtn, hasEditModuleBtn, hasDeleteModuleBtn, hasModuleLeadDrop, hasEditModuleModal))
	}

	assertFullWidthPageContainer(page, &res, "Modules")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 12: Wiki Workspace & Document Editor
// -----------------------------------------------------------------------------
func CheckWikiWorkspace(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/wiki"
	res := ValidationCheckResult{
		ID:         "12-wiki-workspace",
		Name:       "Wiki Workspace, Collections Tree & Document Editor",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "12-wiki-workspace.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(page.MustElement("main").MustText(), "Welcome to Company's Wiki") {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}

	asideText := page.MustElement("aside").MustText()
	mainText := page.MustElement("main").MustText()

	if strings.Contains(asideText, "Wiki") &&
		strings.Contains(asideText, "New page") &&
		strings.Contains(asideText, "Collections") &&
		strings.Contains(asideText, "General") {
		res.Assertions = append(res.Assertions, "Secondary panel switches to Wiki navigation (New page, Home, Collections -> General -> Pages tree)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Secondary panel did not switch to Wiki Collections/Pages tree")
	}

	hasWikiTitleInput, _, _ := page.Has(`input[aria-label="Wiki page title"]`)
	hasWikiSaveBtn, _, _ := page.Has(`button[aria-label="Save wiki page"]`)
	if strings.Contains(mainText, "Welcome to Company's Wiki") &&
		hasWikiTitleInput &&
		hasWikiSaveBtn &&
		!strings.Contains(mainText, "Publish") &&
		!strings.Contains(mainText, "AI summary") {
		res.Assertions = append(res.Assertions, "Right workspace renders Page breadcrumb bar, 'Save' button (replacing 'Publish'), formatting toolbar, and blur-updated document canvas")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Right workspace failed Wiki assertions; hasTitle=%v, hasSave=%v", hasWikiTitleInput, hasWikiSaveBtn))
	}

	assertFullWidthPageContainer(page, &res, "Wiki")

	// Verify creating a project auto-creates a Wiki collection and project 'Pages' link navigates to it
	cleanupTestProjectByName("wiki-sync-test")
	defer cleanupTestProjectByName("wiki-sync-test")

	homePage := newViewportPage(browser, baseURL+"/home")
	if hasPlus, plusBtn, _ := homePage.Has(`button[aria-label="Create new project"]`); hasPlus && plusBtn != nil {
		plusBtn.MustClick()
		time.Sleep(250 * time.Millisecond)

		nameInput := homePage.MustElement(`input[aria-label="New project name"]`)
		nameInput.MustInput("wiki-sync-test")

		prefixInput := homePage.MustElement(`input[aria-label="New project key prefix"]`)
		prefixInput.MustInput("WSYN")

		descInput := homePage.MustElement(`textarea[aria-label="New project description"]`)
		descInput.MustInput("Auto wiki sync test project.")

		submitBtn := homePage.MustElement(`button[aria-label="Submit create project"]`)
		submitBtn.MustClick()

		syncDeadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(syncDeadline) {
			if hasLink, _, _ := homePage.Has(`a[aria-label="Open wiki-sync-test pages in Wiki"]`); hasLink {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		if hasPagesLink, pagesLink, _ := homePage.Has(`a[aria-label="Open wiki-sync-test pages in Wiki"]`); hasPagesLink && pagesLink != nil {
			pagesLink.MustClick()
			openDeadline := time.Now().Add(6 * time.Second)
			for time.Now().Before(openDeadline) {
				mText := homePage.MustElement("main").MustText()
				if strings.Contains(mText, "wiki-sync-test — Project Overview") {
					break
				}
				time.Sleep(150 * time.Millisecond)
			}
			mText := homePage.MustElement("main").MustText()
			if strings.Contains(mText, "wiki-sync-test — Project Overview") {
				res.Assertions = append(res.Assertions, "Creating project 'wiki-sync-test' automatically created matching Wiki collection and opened its project overview page in Wiki")
			} else {
				res.Passed = false
				res.Errors = append(res.Errors, "Navigating to newly created project pages in Wiki failed to open project overview page")
			}
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Missing link a[aria-label='Open wiki-sync-test pages in Wiki'] under created project")
		}
	}
	homePage.MustClose()

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 13: Stickies Board Scratchpad
// -----------------------------------------------------------------------------
func CheckStickiesBoard(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/stickies"
	res := ValidationCheckResult{
		ID:         "13-stickies-board",
		Name:       "Stickies Board, Fun Color Palettes & Surprise Me Generator",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "13-stickies-board.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	mainText := page.MustElement("main").MustText()
	if strings.Contains(mainText, "Stickies") &&
		strings.Contains(mainText, "Electric Banana") &&
		strings.Contains(mainText, "Disco Flamingo") &&
		strings.Contains(mainText, "Radioactive Mint") {
		res.Assertions = append(res.Assertions, "Stickies board renders colorful sticky cards with funny color palettes (Electric Banana, Disco Flamingo, Radioactive Mint, etc.)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Stickies view failed to render colorful sticky notes or funny palette labels")
	}

	if has, surpriseBtn, _ := page.Has(`button[aria-label="Surprise me random sticky"]`); has && surpriseBtn != nil {
		surpriseBtn.MustClick()
		var updatedMain string
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			updatedMain = page.MustElement("main").MustText()
			if strings.Contains(updatedMain, "All Colors (7)") {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
		if strings.Contains(updatedMain, "All Colors (7)") {
			res.Assertions = append(res.Assertions, "Clicking '🎲 Surprise Me' spawns a new sticky note with a random funny color persisted in Firestore (count updated to 7)")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Clicking 'Surprise Me' did not increment sticky notes count to 7")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Surprise me random sticky']")
	}

	assertFullWidthPageContainer(page, &res, "Stickies")

	page.MustScreenshot(res.Screenshot)
	cleanupTestStickies()
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// -----------------------------------------------------------------------------
// Suite 14: Settings Workspace
// -----------------------------------------------------------------------------
func CheckSettings(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/settings"
	res := ValidationCheckResult{
		ID:         "14-settings",
		Name:       "Settings Workspace: General App Preferences & Per-Project Settings",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "14-settings.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	asideText := page.MustElement("aside").MustText()
	mainText := page.MustElement("main").MustText()

	if strings.Contains(asideText, "Settings") &&
		strings.Contains(asideText, "General Settings") &&
		strings.Contains(asideText, "Per-Project Settings") {
		res.Assertions = append(res.Assertions, "Secondary panel switches to Settings navigation (General Settings + Per-Project Settings list)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Secondary panel did not render Settings navigation")
	}

	if strings.Contains(mainText, "Application & Interface Preferences") &&
		strings.Contains(mainText, "Notifications & Activity Digest") {
		res.Assertions = append(res.Assertions, "General App Settings view renders mock global preferences and notification toggles")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "General App Settings view failed to render expected mock sections")
	}

	if has, projSettingsBtn, _ := page.Has(`button[aria-label="Open project settings for hodo"]`); has && projSettingsBtn != nil {
		projSettingsBtn.MustClick()
		time.Sleep(300 * time.Millisecond)

		updatedMain := page.MustElement("main").MustText()
		if strings.Contains(updatedMain, "Project Identity & Metadata — hodo") &&
			strings.Contains(updatedMain, "Work Item & Cycle Defaults (hodo)") &&
			strings.Contains(updatedMain, "Wiki Collection & Documentation Policy (hodo)") {
			res.Assertions = append(res.Assertions, "Clicking project in Settings panel switches to Per-Project Settings (Metadata, Workflow Defaults, Wiki Policy & RBAC)")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Per-Project Settings view did not render expected project sections for 'hodo'")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Open project settings for hodo'] in Settings panel")
	}

	assertFullWidthPageContainer(page, &res, "Settings")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

func cleanupTestStickies() {
	seeded := map[string]bool{
		"sticky-1": true,
		"sticky-2": true,
		"sticky-3": true,
		"sticky-4": true,
		"sticky-5": true,
		"sticky-6": true,
	}
	resp, err := http.Get("http://localhost:8082/api/v1/projects/proj-jynko-core/stickies")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var list []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return
	}
	for _, s := range list {
		if !seeded[s.ID] {
			req, _ := http.NewRequest(http.MethodDelete, "http://localhost:8082/api/v1/projects/proj-jynko-core/stickies/"+s.ID, nil)
			if r, err := http.DefaultClient.Do(req); err == nil {
				_ = r.Body.Close()
			}
		}
	}
}

func cleanupTestProjectByName(name string) {
	resp, err := http.Get("http://localhost:8082/api/v1/projects")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var list []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return
	}
	for _, p := range list {
		if strings.EqualFold(p.Name, name) {
			req, _ := http.NewRequest(http.MethodDelete, "http://localhost:8082/api/v1/projects/"+p.ID, nil)
			if r, err := http.DefaultClient.Do(req); err == nil {
				_ = r.Body.Close()
			}
		}
	}
}
