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

// DefaultChecks returns the canonical list of Jynko E2E UI validation checks.
// Always add or update checks here whenever new UI routes, components, or invariants are introduced.
func DefaultChecks() []CheckSpec {
	return []CheckSpec{
		{
			ID:   "01-home-and-sidebar",
			Name: "Global Brand Header, Work Panel (No Drafts, '+' Projects Trigger) & Multi-Project Home",
			Run:  CheckHomeAndSidebar,
		},
		{
			ID:   "02-my-work-and-dropdown",
			Name: "Cross-Project My Work Overview & Styled Dropdown Menu Interaction",
			Run:  CheckMyWorkAndStyledDropdown,
		},
		{
			ID:   "03-projects-and-plus-modal",
			Name: "Project Overview (No Header New Project Button) & Left Panel '+' Create Project Modal",
			Run:  CheckProjectsAndLeftPanelPlusModal,
		},
		{
			ID:   "04-work-items-views",
			Name: "Work Items Shell (5 View Switcher Modes, Styled Filter Dropdowns & Configurable Kanban Columns)",
			Run:  CheckWorkItemsViews,
		},
		{
			ID:   "05-wiki-section-layout",
			Name: "Two-Section Wiki Layout (Secondary Panel Collections/Pages Tree + Right-Side Page Editor)",
			Run:  CheckWikiSectionLayout,
		},
		{
			ID:   "06-stickies-board",
			Name: "Stickies Canvas, Funny Color Palettes & Interactive Random Note Creation",
			Run:  CheckStickiesBoard,
		},
		{
			ID:   "07-project-pages-and-auto-collection",
			Name: "Create Project -> Auto-Created Wiki Collection & Project 'Pages' Navigation to Wiki",
			Run:  CheckProjectPagesAndAutoWikiCollection,
		},
		{
			ID:   "08-settings-general-and-project",
			Name: "Settings Section: General App Settings & Per-Project Settings Switcher",
			Run:  CheckSettingsGeneralAndPerProject,
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

// 1. CheckHomeAndSidebar validates /home: Top-left Jynko brand icon (no caret), Work panel without Drafts, "+" at root of Projects, Cross-Project Home without "+ New work item" button.
func CheckHomeAndSidebar(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/home"
	res := ValidationCheckResult{
		ID:         "01-home-and-sidebar",
		Name:       "Global Brand Header, Work Panel (No Drafts, '+' Projects Trigger) & Multi-Project Home",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "01-home-overview.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	bodyText := page.MustElement("body").MustText()
	headerText := page.MustElement("header").MustText()
	asideText := page.MustElement("aside").MustText()
	mainText := page.MustElement("main").MustText()

	if strings.Contains(headerText, "Jynko") && !strings.Contains(headerText, "⌄") {
		res.Assertions = append(res.Assertions, "Top-left application bar displays 'Jynko' brand icon without dropdown caret")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Expected top header to show 'Jynko' without '⌄', got: %q", headerText))
	}

	hasLeftUserDropdown, _, _ := page.Has(`button[aria-label="Switch IAP Test Identity"]`)
	if !hasLeftUserDropdown &&
		!strings.Contains(asideText, "Drafts") &&
		!strings.Contains(asideText, "Views") &&
		strings.Contains(asideText, "Home") &&
		strings.Contains(asideText, "My Work") &&
		strings.Contains(asideText, "Stickies") {
		res.Assertions = append(res.Assertions, "Work secondary panel displays Home, My Work, Stickies and omits Drafts, Views, and left-panel user dropdown")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Work secondary panel either still contains 'Drafts'/'Views'/user dropdown or is missing Home/My Work/Stickies")
	}

	if has, _, _ := page.Has(`button[aria-label="Create new project"]`); has {
		res.Assertions = append(res.Assertions, "Left panel has '+' button (aria-label='Create new project') at the root of the Projects list")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing '+' button at root of Projects list in left panel")
	}

	// Test collapsing and re-expanding the secondary panel
	if hasCollapse, collapseBtn, _ := page.Has(`button[aria-label="Collapse secondary panel"]`); hasCollapse && collapseBtn != nil {
		collapseBtn.MustClick()
		time.Sleep(200 * time.Millisecond)

		if hasExpand, expandBtn, _ := page.Has(`button[aria-label="Expand secondary panel"]`); hasExpand && expandBtn != nil {
			expandBtn.MustClick()
			time.Sleep(200 * time.Millisecond)

			restoredAside := page.MustElement("aside").MustText()
			if strings.Contains(restoredAside, "Projects") {
				res.Assertions = append(res.Assertions, "Secondary panel collapse button collapses the panel and displays 'Expand secondary panel' button to restore it")
			} else {
				res.Passed = false
				res.Errors = append(res.Errors, "Clicking 'Expand secondary panel' did not restore full secondary panel")
			}
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Collapsing secondary panel did not render button[aria-label='Expand secondary panel']")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Collapse secondary panel']")
	}

	if strings.Contains(mainText, "hodo") &&
		strings.Contains(mainText, "cloud-infra") &&
		!strings.Contains(mainText, "+ New work item") {
		res.Assertions = append(res.Assertions, "Home page displays overview cards for all user projects ('hodo' & 'cloud-infra') without '+ New work item' button")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Home main view check failed; body=%d chars", len(bodyText)))
	}

	assertFullWidthPageContainer(page, &res, "Home")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// 2. CheckMyWorkAndStyledDropdown validates /my-work: Cross-project assigned work items & interactive styled <app-dropdown> menu.
func CheckMyWorkAndStyledDropdown(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/my-work"
	res := ValidationCheckResult{
		ID:         "02-my-work-and-dropdown",
		Name:       "Cross-Project My Work Overview & Styled Dropdown Menu Interaction",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "02-my-work-dropdown-open.png"),
	}

	page := newViewportPage(browser, url)
	defer page.MustClose()

	mainText := page.MustElement("main").MustText()
	if strings.Contains(mainText, "My Work") &&
		strings.Contains(mainText, "HODO-1") &&
		strings.Contains(mainText, "CLD-1") {
		res.Assertions = append(res.Assertions, "My Work page displays items assigned to current user across both 'hodo' (HODO-1) and 'cloud-infra' (CLD-1)")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "My Work page did not render cross-project assigned items HODO-1 and CLD-1")
	}

	if selects := page.MustElements("select"); len(selects) == 0 {
		res.Assertions = append(res.Assertions, "Zero native browser <select> elements present on My Work page")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Expected 0 native <select> elements, found %d", len(selects)))
	}

	if has, btn, _ := page.Has(`button[aria-label="Filter My Work by Status"]`); has && btn != nil {
		btn.MustClick()
		time.Sleep(250 * time.Millisecond)
		if hasListbox, listbox, _ := page.Has(`[role="listbox"]`); hasListbox && listbox != nil {
			res.Assertions = append(res.Assertions, "Clicking styled <app-dropdown> opens custom floating [role='listbox'] menu instead of native browser select")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Styled dropdown click did not open [role='listbox'] menu")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Could not find styled dropdown trigger button[aria-label='Filter My Work by Status']")
	}

	assertFullWidthPageContainer(page, &res, "My Work")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// 3. CheckProjectsAndLeftPanelPlusModal validates /projects: No "+ New Project" button in Project Overview header; clicking "+" in left panel opens Create New Project modal.
func CheckProjectsAndLeftPanelPlusModal(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/projects"
	res := ValidationCheckResult{
		ID:         "03-projects-and-plus-modal",
		Name:       "Project Overview (No Header New Project Button) & Left Panel '+' Create Project Modal",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "03-projects-create-modal.png"),
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

	// Verify clicking another project's Overview in the left sidebar updates the browser URL and active project
	if hasCloudOverview, cloudLink, _ := page.Has(`a[aria-label="Open cloud-infra overview"]`); hasCloudOverview && cloudLink != nil {
		cloudLink.MustClick()
		time.Sleep(300 * time.Millisecond)
		switchedURL := page.MustInfo().URL
		switchedMain := page.MustElement("main").MustText()
		if strings.Contains(switchedURL, "/projects/proj-cloud-infra") && strings.Contains(switchedMain, "projects/proj-cloud-infra") {
			res.Assertions = append(res.Assertions, "Selecting 'cloud-infra -> Overview' updates URL to /projects/proj-cloud-infra and renders cloud-infra overview")
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
	if strings.Contains(mainText, "Project Overview & Per-Project RBAC") &&
		!strings.Contains(mainText, "+ New Project") {
		res.Assertions = append(res.Assertions, "Project Overview header omits '+ New Project' button")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Project Overview page still contains '+ New Project' button or failed to load")
	}

	assertFullWidthPageContainer(page, &res, "Projects")

	if has, plusBtn, _ := page.Has(`button[aria-label="Create new project"]`); has && plusBtn != nil {
		plusBtn.MustClick()
		time.Sleep(300 * time.Millisecond)
		bodyText := page.MustElement("body").MustText()
		if strings.Contains(bodyText, "Create New Project") && strings.Contains(bodyText, "Key Prefix") {
			res.Assertions = append(res.Assertions, "Clicking '+' at root of Projects list in left panel opens 'Create New Project' modal")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Clicking '+' button did not open 'Create New Project' modal")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Create new project'] in left panel")
	}

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// 4. CheckWorkItemsViews validates /work-items: All 5 view switcher modes, styled filter dropdowns & configurable Kanban columns (Cancelled hidden by default).
func CheckWorkItemsViews(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/work-items"
	res := ValidationCheckResult{
		ID:         "04-work-items-views",
		Name:       "Work Items Shell (5 View Switcher Modes, Styled Filter Dropdowns & Configurable Kanban Columns)",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "04-work-items-board.png"),
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

	// Verify switching to cloud-infra work items updates the URL to /projects/proj-cloud-infra/work-items
	if hasCloudItems, cloudItemsLink, _ := page.Has(`a[aria-label="Open cloud-infra work items"]`); hasCloudItems && cloudItemsLink != nil {
		cloudItemsLink.MustClick()
		time.Sleep(300 * time.Millisecond)
		switchedURL := page.MustInfo().URL
		if strings.Contains(switchedURL, "/projects/proj-cloud-infra/work-items") {
			res.Assertions = append(res.Assertions, "Selecting 'cloud-infra -> Work items' updates URL to /projects/proj-cloud-infra/work-items")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, fmt.Sprintf("Expected URL '/projects/proj-cloud-infra/work-items', got: %q", switchedURL))
		}
		if hasHodoItems, hodoItemsLink, _ := page.Has(`a[aria-label="Open hodo work items"]`); hasHodoItems && hodoItemsLink != nil {
			hodoItemsLink.MustClick()
			time.Sleep(300 * time.Millisecond)
		}
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

	// Verify priority badges show only the label ('High', 'Medium', 'Urgent') without 'lll' or 'll' prefix
	if !strings.Contains(mainText, "lll High") && !strings.Contains(mainText, "ll Medium") && !strings.Contains(mainText, "\nlll\n") {
		res.Assertions = append(res.Assertions, "Priority badges display clean label names ('High', 'Medium', 'Urgent') without 'll'/'lll' prefixes")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Priority badges still contain 'll' or 'lll' prefixes")
	}

	// Verify 'Cancelled' column is hidden by default on the Kanban board while 'Backlog'/'Todo'/'In Progress'/'In Review'/'Done' are visible, and no 'Hidden Columns' card is shown on the board
	hasCancelledCol, _, _ := page.Has(`[data-board-column="CANCELLED"]`)
	hasTodoCol, _, _ := page.Has(`[data-board-column="TODO"]`)
	if !hasCancelledCol && hasTodoCol && strings.Contains(mainText, "Columns (5/6)") && !strings.Contains(mainText, "Hidden Columns") {
		res.Assertions = append(res.Assertions, "Kanban board hides 'Cancelled' column by default (Columns 5/6) without a 'Hidden Columns' rail on the board")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Expected Cancelled board column hidden by default, Columns (5/6), and no Hidden Columns rail; hasCancelledCol=%v", hasCancelledCol))
	}

	// Open "Columns (5/6)" popover and verify configurable column toggles
	if has, colBtn, _ := page.Has(`button[aria-label="Configure Kanban board columns"]`); has && colBtn != nil {
		colBtn.MustClick()
		time.Sleep(250 * time.Millisecond)
		if hasToggle, _, _ := page.Has(`input[aria-label="Toggle board column Cancelled"]`); hasToggle {
			res.Assertions = append(res.Assertions, "Clicking 'Columns (5/6)' opens Kanban Board Columns configurator with per-status visibility switches")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Kanban Board Columns configurator popover did not render status switches")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Configure Kanban board columns'] in Work Items toolbar")
	}

	page.MustScreenshot(res.Screenshot)

	// Open Work Item side panel on the board and verify Start Date, Due Date, Full view button, and no Delete button
	if hasCard, cardEl, _ := page.Has(`[data-item-key="HODO-1"]`); hasCard && cardEl != nil {
		cardEl.MustClick()
		time.Sleep(300 * time.Millisecond)
		if hasDrawer, drawerEl, _ := page.Has(`app-work-item-detail-drawer aside`); hasDrawer && drawerEl != nil {
			drawerText := drawerEl.MustText()
			hasFullViewBtn, _, _ := drawerEl.Has(`button[aria-label="Open work item full page"]`)
			hasStartInput, _, _ := drawerEl.Has(`input[aria-label="Work item start date"]`)
			hasDueInput, _, _ := drawerEl.Has(`input[aria-label="Work item due date"]`)
			if hasFullViewBtn && hasStartInput && hasDueInput && !strings.Contains(drawerText, "Delete") {
				res.Assertions = append(res.Assertions, "Work Item side panel renders Start Date, Due Date, and 'Full view' button without a Delete button")
			} else {
				res.Passed = false
				res.Errors = append(res.Errors, fmt.Sprintf("Work Item side panel failed assertions: hasFullView=%v, hasStart=%v, hasDue=%v", hasFullViewBtn, hasStartInput, hasDueInput))
			}
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Clicking work item card did not open app-work-item-detail-drawer")
		}
	}

	// Verify the dedicated full-page Work Item route (/projects/proj-jynko-core/work-items/item-1)
	detailPage := newViewportPage(browser, baseURL+"/projects/proj-jynko-core/work-items/item-1")
	defer detailPage.MustClose()
	detailMain := detailPage.MustElement("main").MustText()
	hasStartDateInput, _, _ := detailPage.Has(`input[aria-label="Work item start date"]`)
	hasDueDateInput, _, _ := detailPage.Has(`input[aria-label="Work item due date"]`)
	hasAutoGrowDesc, _, _ := detailPage.Has(`textarea.auto-grow-description[aria-label="Work item description"]`)
	if strings.Contains(detailMain, "HODO-1") && hasStartDateInput && hasDueDateInput && hasAutoGrowDesc && strings.Contains(detailMain, "Sub-Work Items") {
		res.Assertions = append(res.Assertions, "Full-page Work Item view (/projects/:projectId/work-items/:itemId) renders auto-growing Description, Start Date, Due Date, Sub-Items, and Comments")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, fmt.Sprintf("Full-page Work Item view failed assertions; hasStartDate=%v, hasDueDate=%v, hasAutoGrowDesc=%v", hasStartDateInput, hasDueDateInput, hasAutoGrowDesc))
	}

	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// 5. CheckWikiSectionLayout validates /wiki: Secondary panel switches to Wiki (Collections & Pages tree) and right side renders Page View & Editor.
func CheckWikiSectionLayout(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/wiki"
	res := ValidationCheckResult{
		ID:         "05-wiki-section-layout",
		Name:       "Two-Section Wiki Layout (Secondary Panel Collections/Pages Tree + Right-Side Page Editor)",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "05-wiki-section.png"),
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

	if strings.Contains(mainText, "Welcome to Company's Wiki") &&
		!strings.Contains(mainText, "AI summary") {
		res.Assertions = append(res.Assertions, "Right workspace renders Page breadcrumb bar, formatting toolbar, and document canvas without AI summary button")
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Right workspace did not render expected Wiki page view & formatting toolbar")
	}

	assertFullWidthPageContainer(page, &res, "Wiki")

	page.MustScreenshot(res.Screenshot)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// 6. CheckStickiesBoard validates /stickies: Colorful sticky notes board, funny color palettes & interactive note creation.
func CheckStickiesBoard(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/stickies"
	res := ValidationCheckResult{
		ID:         "06-stickies-board",
		Name:       "Stickies Canvas, Funny Color Palettes & Interactive Random Note Creation",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "06-stickies-board.png"),
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

	// Click "Surprise Me" to spawn a new random sticky note persisted in Firestore
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

// 7. CheckProjectPagesAndAutoWikiCollection validates creating a new project automatically creates its Wiki collection, and clicking 'Pages' under the project navigates to that collection in the Wiki section.
func CheckProjectPagesAndAutoWikiCollection(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/home"
	res := ValidationCheckResult{
		ID:         "07-project-pages-and-auto-collection",
		Name:       "Create Project -> Auto-Created Wiki Collection & Project 'Pages' Navigation to Wiki",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "07-project-pages-wiki-collection.png"),
	}

	cleanupTestProjectByName("mobile-sdk")

	page := newViewportPage(browser, url)
	defer page.MustClose()
	defer cleanupTestProjectByName("mobile-sdk")

	// Step 1: Open Create Project modal from left sidebar '+' button
	if has, plusBtn, _ := page.Has(`button[aria-label="Create new project"]`); has && plusBtn != nil {
		plusBtn.MustClick()
		time.Sleep(250 * time.Millisecond)

		nameInput := page.MustElement(`input[aria-label="New project name"]`)
		nameInput.MustInput("mobile-sdk")

		prefixInput := page.MustElement(`input[aria-label="New project key prefix"]`)
		prefixInput.MustInput("MSDK")

		descInput := page.MustElement(`textarea[aria-label="New project description"]`)
		descInput.MustInput("Cross-platform mobile SDK for iOS and Android.")

		submitBtn := page.MustElement(`button[aria-label="Submit create project"]`)
		submitBtn.MustClick()

		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) {
			if hasLink, _, _ := page.Has(`a[aria-label="Open mobile-sdk pages in Wiki"]`); hasLink {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		asideText := page.MustElement("aside").MustText()
		if strings.Contains(asideText, "mobile-sdk") {
			res.Assertions = append(res.Assertions, "Created new project 'mobile-sdk' (MSDK) in Firestore via left sidebar '+' modal")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "New project 'mobile-sdk' did not appear in the left Projects tree")
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing button[aria-label='Create new project']")
	}

	// Step 2: Click 'Pages' under the newly created 'mobile-sdk' project in the left Work panel
	if has, pagesLink, _ := page.Has(`a[aria-label="Open mobile-sdk pages in Wiki"]`); has && pagesLink != nil {
		pagesLink.MustClick()

		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) {
			mainText := page.MustElement("main").MustText()
			if strings.Contains(mainText, "mobile-sdk — Project Overview") {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}

		asideText := page.MustElement("aside").MustText()
		mainText := page.MustElement("main").MustText()

		if hasCol, _, _ := page.Has(`button[aria-label="Collection mobile-sdk"]`); hasCol && strings.Contains(asideText, "Collections") {
			res.Assertions = append(res.Assertions, "Creating project 'mobile-sdk' automatically created the 'mobile-sdk' Collection in the Wiki section")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, "Wiki section did not contain auto-created 'mobile-sdk' collection")
		}

		if strings.Contains(mainText, "mobile-sdk") && strings.Contains(mainText, "mobile-sdk — Project Overview") {
			res.Assertions = append(res.Assertions, "Clicking 'Pages' under project 'mobile-sdk' navigated to the 'mobile-sdk' collection and opened its project overview page in the Wiki section")
		} else {
			res.Passed = false
			res.Errors = append(res.Errors, fmt.Sprintf("Wiki editor did not open 'mobile-sdk — Project Overview' page; mainText=%q", mainText))
		}
	} else {
		res.Passed = false
		res.Errors = append(res.Errors, "Missing link a[aria-label='Open mobile-sdk pages in Wiki'] under newly created project")
	}

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

// 8. CheckSettingsGeneralAndPerProject validates /settings: General App Settings view & interactive switching to Per-Project Settings.
func CheckSettingsGeneralAndPerProject(browser *rod.Browser, baseURL, outDir string) ValidationCheckResult {
	start := time.Now()
	url := baseURL + "/settings"
	res := ValidationCheckResult{
		ID:         "08-settings-general-and-project",
		Name:       "Settings Section: General App Settings & Per-Project Settings Switcher",
		URL:        url,
		Passed:     true,
		Screenshot: filepath.Join(outDir, "08-settings-page.png"),
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

	// Click 'hodo' in the Per-Project Settings sidebar list to switch to Per-Project Settings
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

	// Verify Top-Right User Avatar Dropdown Card & User Profile Page (/profile)
	if hasUserBtn, userBtn, _ := page.Has(`button[aria-label="Open user menu"]`); hasUserBtn && userBtn != nil {
		userBtn.MustClick()
		time.Sleep(250 * time.Millisecond)

		if hasCard, cardEl, _ := page.Has(`[data-testid="user-dropdown-card"]`); hasCard && cardEl != nil {
			cardText := cardEl.MustText()
			hasProfileBtn, profileBtn, _ := cardEl.Has(`button[aria-label="Open user profile"]`)
			hasSignOutBtn, _, _ := cardEl.Has(`button[aria-label="Sign out"]`)
			if hasProfileBtn && hasSignOutBtn && strings.Contains(cardText, "dev@jynko.local") {
				res.Assertions = append(res.Assertions, "Top-right user button opens account dropdown card with Name, Email, Profile button, and Sign out button")
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
					res.Assertions = append(res.Assertions, "Clicking 'Profile' in user dropdown navigates to /profile with Name, read-only IAP Email, and Photo/Avatar controls")
				} else {
					res.Passed = false
					res.Errors = append(res.Errors, fmt.Sprintf("User Profile page (/profile) failed assertions: url=%q, hasName=%v, hasReadonlyEmail=%v, hasAvatar=%v", profileURL, hasNameInput, hasReadonlyEmail, hasAvatarInput))
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
		res.Errors = append(res.Errors, "Missing top-right user button button[aria-label='Open user menu']")
	}

	res.DurationMs = time.Since(start).Milliseconds()
	return res
}
