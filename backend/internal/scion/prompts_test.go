package scion

import (
	"strings"
	"testing"
)

func TestRenderPromptAndRepoParsing(t *testing.T) {
	vars := PromptVars{
		ProjectID:       "proj-jynko-core",
		ProjectName:     "hodo",
		ItemID:          "item-1",
		ItemKey:         "HODO-12",
		ItemTitle:       "Implement Scion commands",
		ItemDescription: "Support /implement and /review",
		RepoURL:         "https://github.com/gabihodoroaga/jynko.git",
		BaseBranch:      "dev",
		Command:         "implement",
		Instructions:    "Keep changes focused",
		UserName:        "Gabriel",
		UserEmail:       "gabriel@example.com",
	}

	rendered := RenderPrompt(DefaultImplementPrompt, vars)
	expectedSubstrings := []string{
		"HODO-12",
		"hodo (proj-jynko-core)",
		"Gabriel <gabriel@example.com>",
		"https://github.com/gabihodoroaga/jynko.git",
		"feat/hodo-12",
		"Keep changes focused",
	}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(rendered, sub) {
			t.Errorf("expected rendered prompt to contain %q, got:\n%s", sub, rendered)
		}
	}

	owner, name := ParseRepoOwnerAndName("git@github.com:gabihodoroaga/jynko-plan.git")
	if owner != "gabihodoroaga" || name != "jynko-plan" {
		t.Errorf("ParseRepoOwnerAndName SSH = (%q, %q), want (gabihodoroaga, jynko-plan)", owner, name)
	}

	custom := RenderPrompt("{{repo_owner}}/{{repo_name}} -> {{feature_branch}}", PromptVars{
		ItemKey: "PLAN-7",
		RepoURL: "https://jynko-git.example.com/git/gabriel/jynko-plan",
	})
	if custom != "gabriel/jynko-plan -> feat/plan-7" {
		t.Errorf("custom RenderPrompt = %q", custom)
	}

	if len(AvailablePlaceholders()) < 16 {
		t.Errorf("expected at least 16 documented placeholders")
	}
}
