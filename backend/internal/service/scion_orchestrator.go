package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/scion"
)

// ScionOrchestrator executes slash commands posted in work item comments against Scion Hub.
type ScionOrchestrator struct {
	cfg        *config.Config
	client     scion.HubClient
	projectSvc *ProjectService
	moduleSvc  *ModuleService
	itemSvc    *ItemService
}

// NewScionOrchestrator creates a new ScionOrchestrator.
func NewScionOrchestrator(
	cfg *config.Config,
	client scion.HubClient,
	projectSvc *ProjectService,
	moduleSvc *ModuleService,
	itemSvc *ItemService,
) *ScionOrchestrator {
	return &ScionOrchestrator{
		cfg:        cfg,
		client:     client,
		projectSvc: projectSvc,
		moduleSvc:  moduleSvc,
		itemSvc:    itemSvc,
	}
}

// DispatchSummary describes the outcome of a slash-command execution.
type DispatchSummary struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Action    string `json:"action,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	AgentURL  string `json:"agent_url,omitempty"`
	Harness   string `json:"harness,omitempty"`
	Error     string `json:"error,omitempty"`
}

// PrecheckResult holds validated project/item/repo metadata before saving a slash-command comment.
type PrecheckResult struct {
	Project    *model.Project
	Item       *model.WorkItem
	Module     *model.Module
	RepoURL    string
	BaseBranch string
	Parsed     scion.ParsedCommand
}

// Precheck validates that Scion is enabled and configured and resolves repo/branch for a slash command.
func (o *ScionOrchestrator) Precheck(ctx context.Context, projectID, itemID string, parsed scion.ParsedCommand) (*PrecheckResult, error) {
	if o == nil || o.cfg == nil || !o.cfg.IsScionConfigured() || o.client == nil {
		return nil, errors.New("Scion Hub is not configured on the server (set SCION_HUB_URL and SCION_HUB_TOKEN)")
	}

	proj, err := o.projectSvc.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if proj.Settings == nil || !proj.Settings.ScionEnabled {
		return nil, errors.New("Scion AI agents are not enabled for this project (enable in Project Settings > Scion AI Agents & Slash Commands)")
	}

	item, err := o.itemSvc.GetByID(ctx, projectID, itemID)
	if err != nil {
		return nil, err
	}

	var mod *model.Module
	if item.ModuleID != "" && o.moduleSvc != nil {
		mod, _ = o.moduleSvc.GetByID(ctx, projectID, item.ModuleID)
	}

	repoURL := strings.TrimSpace(parsed.RepoURL)
	if repoURL == "" && mod != nil {
		repoURL = strings.TrimSpace(mod.GitRepoURL)
	}
	if repoURL == "" {
		repoURL = strings.TrimSpace(proj.Settings.DefaultGitRepoURL)
	}

	baseBranch := strings.TrimSpace(parsed.BaseBranch)
	if baseBranch == "" && mod != nil {
		baseBranch = strings.TrimSpace(mod.BaseBranch)
	}
	if baseBranch == "" {
		baseBranch = strings.TrimSpace(proj.Settings.DefaultBaseBranch)
	}
	if baseBranch == "" {
		baseBranch = "dev"
	}

	switch parsed.Command {
	case scion.CmdImplement, scion.CmdReview, scion.CmdFix, scion.CmdPlan:
		if repoURL == "" {
			return nil, errors.New("no Git repository URL configured: set Default Git Repo URL in Project Settings, on the Module, or pass --repo=<url>")
		}
	}

	return &PrecheckResult{
		Project:    proj,
		Item:       item,
		Module:     mod,
		RepoURL:    repoURL,
		BaseBranch: baseBranch,
		Parsed:     parsed,
	}, nil
}

// Execute runs the validated slash command, updates WorkItem agent bindings, and creates a system dispatch comment.
func (o *ScionOrchestrator) Execute(
	ctx context.Context,
	pre *PrecheckResult,
	user model.UserContext,
	targetAgentIDOverride string,
) (*model.Comment, *DispatchSummary) {
	parsed := pre.Parsed
	proj := pre.Project
	item := pre.Item
	settings := proj.Settings

	projectSlug := strings.TrimSpace(settings.ScionProjectSlug)
	if projectSlug == "" {
		projectSlug = nonEmptyOrDefault(o.cfg.ScionDefaultProjectSlug, "global")
	}
	templateName := strings.TrimSpace(settings.ScionTemplate)
	if templateName == "" {
		templateName = nonEmptyOrDefault(o.cfg.ScionDefaultTemplate, "jynko-dev")
	}

	vars := scion.PromptVars{
		ProjectID:       proj.ID,
		ProjectName:     proj.Name,
		ItemID:          item.ID,
		ItemKey:         item.Key,
		ItemTitle:       item.Title,
		ItemDescription: item.Description,
		RepoURL:         pre.RepoURL,
		BaseBranch:      pre.BaseBranch,
		Command:         parsed.Command,
		Instructions:    parsed.Instructions,
		UserName:        user.Name,
		UserEmail:       user.Email,
	}

	explicitTarget := strings.TrimSpace(parsed.TargetAgent)
	if explicitTarget == "" {
		explicitTarget = strings.TrimSpace(targetAgentIDOverride)
	}
	if explicitTarget == "__new__" {
		parsed.NewAgent = true
		explicitTarget = ""
	}

	summary := &DispatchSummary{
		OK:      true,
		Command: parsed.Command,
	}

	var sysBody string
	switch parsed.Command {
	case scion.CmdStop:
		var toStop []string
		if explicitTarget != "" {
			toStop = append(toStop, explicitTarget)
		} else {
			if item.ImplAgentID != "" {
				toStop = append(toStop, item.ImplAgentID)
			}
			if parsed.StopAll && item.ReviewAgentID != "" && item.ReviewAgentID != item.ImplAgentID {
				toStop = append(toStop, item.ReviewAgentID)
			}
		}
		if len(toStop) == 0 {
			summary.OK = false
			summary.Error = "no bound Scion agent found on this work item to stop"
			sysBody = fmt.Sprintf("Could not execute `/stop`: %s.", summary.Error)
		} else {
			var stopped []string
			var stopErrs []string
			for _, id := range toStop {
				if err := o.client.StopAgent(ctx, id); err != nil {
					stopErrs = append(stopErrs, fmt.Sprintf("%s (%v)", id, err))
				} else {
					stopped = append(stopped, id)
				}
			}
			summary.Action = "stopped"
			summary.AgentID = toStop[0]
			summary.AgentURL = o.agentWebURL(toStop[0])
			if len(stopErrs) > 0 {
				summary.OK = false
				summary.Error = strings.Join(stopErrs, "; ")
				sysBody = fmt.Sprintf("Stopped [%s]; failed on: %s.", strings.Join(stopped, ", "), summary.Error)
			} else {
				sysBody = fmt.Sprintf("Stopped Scion agent(s): `%s`.", strings.Join(stopped, "`, `"))
			}
		}

	case scion.CmdImplement, scion.CmdPlan, scion.CmdFix, scion.CmdAsk:
		harness := nonEmptyOrDefault(parsed.Harness, nonEmptyOrDefault(settings.ImplementHarness, "claude"))
		tmpl := nonEmptyOrDefault(settings.ImplementPromptTemplate, scion.DefaultImplementPrompt)
		if parsed.Command == scion.CmdPlan || parsed.Command == scion.CmdFix || parsed.Command == scion.CmdAsk {
			tmpl = nonEmptyOrDefault(settings.FollowupPromptTemplate, scion.DefaultFollowupPrompt)
		}
		prompt := scion.RenderPrompt(tmpl, vars)

		targetID := explicitTarget
		if !parsed.NewAgent && targetID == "" {
			targetID = item.ImplAgentID
		}

		if parsed.Command == scion.CmdAsk && targetID == "" {
			summary.OK = false
			summary.Error = "no implementation or review agent is bound to this work item yet (run /implement or select a target agent)"
			sysBody = fmt.Sprintf("Could not execute `/ask`: %s.", summary.Error)
			break
		}

		agent, action, err := o.dispatchToAgent(ctx, dispatchParams{
			TargetID:    targetID,
			ForceNew:    parsed.NewAgent,
			DefaultSlug: fmt.Sprintf("impl-%s", strings.ToLower(item.Key)),
			ProjectSlug: projectSlug,
			Template:    templateName,
			Harness:     harness,
			Prompt:      prompt,
			ProjectID:   proj.ID,
			User:        user,
		})
		if err != nil {
			summary.OK = false
			summary.Error = err.Error()
			sysBody = fmt.Sprintf("Failed to dispatch `/%s` to Scion Hub: %s", parsed.Command, err.Error())
		} else {
			summary.Action = action
			summary.AgentID = agent.ID
			summary.AgentName = agent.Name
			summary.AgentURL = o.agentWebURL(agent.ID)
			summary.Harness = harness

			// Bind as implementation agent unless user explicitly targeted the bound review agent
			if explicitTarget == "" || explicitTarget != item.ReviewAgentID {
				item.ImplAgentID = agent.ID
				item.ImplAgentName = agent.Name
				item.ImplAgentHarness = harness
				_ = o.itemSvc.SaveItem(ctx, item)
			}
			sysBody = fmt.Sprintf("Dispatched `/%s` (%s) to Scion agent **%s** (`%s`, harness: `%s`).",
				parsed.Command, action, nonEmptyOrDefault(agent.Name, agent.ID), agent.ID, harness)
		}

	case scion.CmdReview:
		harness := nonEmptyOrDefault(parsed.Harness, nonEmptyOrDefault(settings.ReviewHarness, "antigravity"))
		tmpl := nonEmptyOrDefault(settings.ReviewPromptTemplate, scion.DefaultReviewPrompt)
		prompt := scion.RenderPrompt(tmpl, vars)

		targetID := explicitTarget
		if !parsed.NewAgent && targetID == "" {
			targetID = item.ReviewAgentID
		}

		agent, action, err := o.dispatchToAgent(ctx, dispatchParams{
			TargetID:    targetID,
			ForceNew:    parsed.NewAgent,
			DefaultSlug: fmt.Sprintf("rev-%s", strings.ToLower(item.Key)),
			ProjectSlug: projectSlug,
			Template:    templateName,
			Harness:     harness,
			Prompt:      prompt,
			ProjectID:   proj.ID,
			User:        user,
		})
		if err != nil {
			summary.OK = false
			summary.Error = err.Error()
			sysBody = fmt.Sprintf("Failed to dispatch `/review` to Scion Hub: %s", err.Error())
		} else {
			summary.Action = action
			summary.AgentID = agent.ID
			summary.AgentName = agent.Name
			summary.AgentURL = o.agentWebURL(agent.ID)
			summary.Harness = harness

			item.ReviewAgentID = agent.ID
			item.ReviewAgentName = agent.Name
			item.ReviewAgentHarness = harness
			_ = o.itemSvc.SaveItem(ctx, item)

			sysBody = fmt.Sprintf("Dispatched `/review` (%s) to Scion agent **%s** (`%s`, harness: `%s`).",
				action, nonEmptyOrDefault(agent.Name, agent.ID), agent.ID, harness)
		}
	}

	sysAuthor := model.UserContext{
		UserID: "scion-orchestrator",
		Name:   "Scion Orchestrator",
	}
	dispatchComment, _ := o.itemSvc.CreateComment(ctx, proj.ID, item.ID, sysAuthor, CreateCommentInput{
		Body:         sysBody,
		IsSystem:     true,
		SlashCommand: parsed.Command,
		AgentID:      summary.AgentID,
		AgentURL:     summary.AgentURL,
	})
	return dispatchComment, summary
}

type dispatchParams struct {
	TargetID    string
	ForceNew    bool
	DefaultSlug string
	ProjectSlug string
	Template    string
	Harness     string
	Prompt      string
	ProjectID   string
	User        model.UserContext
}

func (o *ScionOrchestrator) dispatchToAgent(ctx context.Context, p dispatchParams) (*scion.Agent, string, error) {
	if !p.ForceNew && p.TargetID != "" {
		err := o.client.EnsureRunningAndSendMessage(ctx, p.TargetID, p.Prompt)
		if err == nil {
			agent, getErr := o.client.GetAgent(ctx, p.TargetID)
			if getErr != nil || agent == nil {
				agent = &scion.Agent{ID: p.TargetID, Name: p.TargetID, Harness: p.Harness}
			}
			return agent, "messaged", nil
		}
		if !errors.Is(err, scion.ErrAgentNotFound) {
			return nil, "", err
		}
	}

	env := map[string]string{
		"DEFAULT_PROJECT_ID": p.ProjectID,
	}
	if p.User.Name != "" {
		env["SCION_USER_NAME"] = p.User.Name
		env["GIT_AUTHOR_NAME"] = p.User.Name
		env["GIT_COMMITTER_NAME"] = p.User.Name
	}
	if p.User.Email != "" {
		env["SCION_USER_EMAIL"] = p.User.Email
		env["GIT_AUTHOR_EMAIL"] = p.User.Email
		env["GIT_COMMITTER_EMAIL"] = p.User.Email
	}

	created, err := o.client.CreateAgent(ctx, scion.CreateAgentRequest{
		Name:              p.DefaultSlug,
		Slug:              p.DefaultSlug,
		ProjectSlug:       p.ProjectSlug,
		Template:          p.Template,
		Task:              p.Prompt,
		AcceptAsyncLaunch: true,
		Config: &scion.AgentConfig{
			Harness: p.Harness,
			Env:     env,
		},
	})
	if err != nil {
		return nil, "", err
	}
	return created, "created", nil
}

func (o *ScionOrchestrator) agentWebURL(agentID string) string {
	if o == nil || o.cfg == nil || strings.TrimSpace(o.cfg.ScionHubURL) == "" || strings.TrimSpace(agentID) == "" {
		return ""
	}
	return strings.TrimRight(o.cfg.ScionHubURL, "/") + "/agents/" + agentID
}
