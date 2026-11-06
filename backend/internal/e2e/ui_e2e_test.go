package e2e_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/e2e"
)

func TestJynkoUIEndToEnd(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("JYNKO_E2E_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:4202"
	}

	outDir := os.Getenv("JYNKO_E2E_OUT_DIR")
	if outDir == "" {
		outDir = filepath.Join(os.TempDir(), "jynko-e2e-artifacts")
	}

	requireE2E := os.Getenv("JYNKO_E2E") == "1" || os.Getenv("CI") == "true"

	if !isFrontendReachable(baseURL) {
		if requireE2E {
			t.Fatalf("Jynko frontend is not reachable at %s (JYNKO_E2E=1 or CI=true is set)", baseURL)
		}
		t.Skipf("Skipping Go-Rod UI E2E suite because frontend is not listening at %s (run ./scripts/run-e2e.sh or set JYNKO_E2E=1)", baseURL)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create E2E screenshot directory %s: %v", outDir, err)
	}

	browser, err := e2e.LaunchBrowser()
	if err != nil {
		t.Fatalf("failed to launch headless browser via go-rod: %v", err)
	}
	defer browser.MustClose()

	report := e2e.ValidationReport{
		Timestamp: time.Now().Format(time.RFC3339),
		BaseURL:   baseURL,
		AllPassed: true,
	}

	for _, spec := range e2e.DefaultChecks() {
		spec := spec
		t.Run(spec.ID, func(t *testing.T) {
			res := spec.Run(browser, baseURL, outDir)
			report.Results = append(report.Results, res)
			report.TotalChecks++
			if res.Passed {
				report.PassedCount++
			} else {
				report.FailedCount++
				report.AllPassed = false
			}

			for _, assertion := range res.Assertions {
				t.Logf("✓ %s", assertion)
			}
			for _, errMsg := range res.Errors {
				t.Errorf("✗ %s", errMsg)
			}
			t.Logf("📸 Screenshot saved: %s (%dms)", res.Screenshot, res.DurationMs)
		})
	}

	reportPath := filepath.Join(outDir, "report.json")
	if raw, err := json.MarshalIndent(report, "", "  "); err == nil {
		_ = os.WriteFile(reportPath, raw, 0o644)
		t.Logf("📄 Full E2E report written to %s", reportPath)
	}
}

func isFrontendReachable(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}
