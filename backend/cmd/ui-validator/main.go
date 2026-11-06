package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gabihodoroaga/jynko-plan/internal/e2e"
)

func main() {
	baseURL := flag.String("base-url", "http://localhost:4202", "Base URL of the running Jynko frontend")
	outDir := flag.String("out-dir", "./ui-validation-screenshots", "Directory to write validation screenshots and report.json")
	flag.Parse()

	browser, err := e2e.LaunchBrowser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer browser.MustClose()

	report, reportPath, err := e2e.RunSuite(browser, *baseURL, *outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	e2e.PrintReport(report, reportPath)

	if !report.AllPassed {
		os.Exit(1)
	}
}
