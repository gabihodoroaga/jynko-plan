package scion

import (
	"net/url"
	"strings"
)

// DefaultImplementPrompt is the built-in prompt template for /implement.
const DefaultImplementPrompt = `You are implementing Jynko work item {{item_key}}: {{item_title}}
Project: {{project_name}} ({{project_id}}) | Requested by: {{user_name}} <{{user_email}}>

Repository: {{repo_url}}
Base branch: {{base_branch}}
Target feature branch: {{feature_branch}}

## Description
{{item_description}}

## Additional Instructions
{{instructions}}

## Execution Guidelines
1. Clone or sync {{repo_url}}, check out {{base_branch}}, and create/switch to branch {{feature_branch}}.
2. Use the Jynko MCP tools (get_work_item, list_attachments, download_attachment) if you need sub-items, comments, or attachments for {{item_key}}.
3. Implement the requested changes cleanly, add/update tests, and verify all builds and tests pass.
4. Commit with a clear message referencing {{item_key}} and push {{feature_branch}} (do NOT merge directly to stg or main).
5. Post a concise summary comment back on {{item_key}} via Jynko MCP (add_comment) when done.`

// DefaultReviewPrompt is the built-in prompt template for /review.
const DefaultReviewPrompt = `You are reviewing Jynko work item {{item_key}}: {{item_title}}
Project: {{project_name}} ({{project_id}}) | Requested by: {{user_name}} <{{user_email}}>

Repository: {{repo_url}}
Base branch: {{base_branch}}
Feature branch to review: {{feature_branch}}

## Description
{{item_description}}

## Review Focus / Instructions
{{instructions}}

## Review Guidelines
1. Fetch {{repo_url}} and inspect the diff between {{base_branch}} and {{feature_branch}}.
2. Check correctness, edge cases, security/RBAC invariants, test coverage, and code quality.
3. Run the relevant test suite to verify nothing is broken.
4. Post your actionable review findings back on {{item_key}} via Jynko MCP (add_comment).`

// DefaultFollowupPrompt is the built-in prompt template for /fix, /plan, and /ask.
const DefaultFollowupPrompt = `Follow-up command /{{command}} for Jynko work item {{item_key}}: {{item_title}}
Project: {{project_name}} ({{project_id}}) | Requested by: {{user_name}} <{{user_email}}>
Repository: {{repo_url}} (base: {{base_branch}}, feature: {{feature_branch}})

## Instructions
{{instructions}}

Please respond concisely and post any status updates or results back on {{item_key}} via Jynko MCP (add_comment) if appropriate.`

// PlaceholderInfo describes a supported template variable for UI helper chips.
type PlaceholderInfo struct {
	Token       string `json:"token"`
	Description string `json:"description"`
}

// AvailablePlaceholders returns the canonical list of supported prompt template placeholders.
func AvailablePlaceholders() []PlaceholderInfo {
	return []PlaceholderInfo{
		{Token: "{{project_id}}", Description: "Jynko project ID"},
		{Token: "{{project_name}}", Description: "Jynko project name"},
		{Token: "{{item_id}}", Description: "Work item Firestore ID"},
		{Token: "{{item_key}}", Description: "Human-readable key (e.g. HODO-12)"},
		{Token: "{{item_key_lower}}", Description: "Lowercase item key (e.g. hodo-12)"},
		{Token: "{{item_title}}", Description: "Work item title"},
		{Token: "{{item_description}}", Description: "Work item Markdown description"},
		{Token: "{{repo_url}}", Description: "Resolved Git repository URL"},
		{Token: "{{repo_owner}}", Description: "Git repository owner"},
		{Token: "{{repo_name}}", Description: "Git repository name"},
		{Token: "{{base_branch}}", Description: "Base branch (e.g. dev)"},
		{Token: "{{feature_branch}}", Description: "Feature branch (default feat/{{item_key_lower}})"},
		{Token: "{{command}}", Description: "Slash command name (implement, review, fix, plan, ask)"},
		{Token: "{{instructions}}", Description: "Optional user instructions after command"},
		{Token: "{{user_name}}", Description: "Commenting user's display name"},
		{Token: "{{user_email}}", Description: "Commenting user's email"},
	}
}

// PromptVars holds all template substitution values.
type PromptVars struct {
	ProjectID       string
	ProjectName     string
	ItemID          string
	ItemKey         string
	ItemKeyLower    string
	ItemTitle       string
	ItemDescription string
	RepoURL         string
	RepoOwner       string
	RepoName        string
	BaseBranch      string
	FeatureBranch   string
	Command         string
	Instructions    string
	UserName        string
	UserEmail       string
}

// RenderPrompt substitutes all {{...}} placeholders in tmpl using vars.
func RenderPrompt(tmpl string, vars PromptVars) string {
	keyLower := strings.TrimSpace(vars.ItemKeyLower)
	if keyLower == "" {
		keyLower = strings.ToLower(strings.TrimSpace(vars.ItemKey))
	}
	featureBranch := strings.TrimSpace(vars.FeatureBranch)
	if featureBranch == "" && keyLower != "" {
		featureBranch = "feat/" + keyLower
	}
	repoOwner, repoName := vars.RepoOwner, vars.RepoName
	if (repoOwner == "" || repoName == "") && strings.TrimSpace(vars.RepoURL) != "" {
		parsedOwner, parsedName := ParseRepoOwnerAndName(vars.RepoURL)
		if repoOwner == "" {
			repoOwner = parsedOwner
		}
		if repoName == "" {
			repoName = parsedName
		}
	}

	replacer := strings.NewReplacer(
		"{{project_id}}", vars.ProjectID,
		"{{project_name}}", vars.ProjectName,
		"{{item_id}}", vars.ItemID,
		"{{item_key}}", vars.ItemKey,
		"{{item_key_lower}}", keyLower,
		"{{item_title}}", vars.ItemTitle,
		"{{item_description}}", vars.ItemDescription,
		"{{repo_url}}", vars.RepoURL,
		"{{repo_owner}}", repoOwner,
		"{{repo_name}}", repoName,
		"{{base_branch}}", vars.BaseBranch,
		"{{feature_branch}}", featureBranch,
		"{{command}}", vars.Command,
		"{{instructions}}", vars.Instructions,
		"{{user_name}}", vars.UserName,
		"{{user_email}}", vars.UserEmail,
	)
	return replacer.Replace(tmpl)
}

// ParseRepoOwnerAndName extracts owner and repository name from HTTPS, SSH, or Jynko Git URLs.
func ParseRepoOwnerAndName(rawURL string) (string, string) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", ""
	}
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.TrimRight(trimmed, "/")

	// SSH format: git@github.com:owner/repo
	if strings.Contains(trimmed, ":") && !strings.Contains(trimmed, "://") {
		if idx := strings.Index(trimmed, ":"); idx != -1 {
			path := strings.TrimLeft(trimmed[idx+1:], "/")
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				return parts[len(parts)-2], parts[len(parts)-1]
			}
		}
		return "", ""
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "", ""
}
