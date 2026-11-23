package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/scion"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// ScionHandler exposes project-scoped Scion Hub defaults, agents list, and ticket agent status.
type ScionHandler struct {
	cfg        *config.Config
	client     scion.HubClient
	projectSvc *service.ProjectService
	itemSvc    *service.ItemService
}

// NewScionHandler creates a new ScionHandler.
func NewScionHandler(
	cfg *config.Config,
	client scion.HubClient,
	projectSvc *service.ProjectService,
	itemSvc *service.ItemService,
) *ScionHandler {
	return &ScionHandler{
		cfg:        cfg,
		client:     client,
		projectSvc: projectSvc,
		itemSvc:    itemSvc,
	}
}

// GetDefaults returns built-in default prompt templates and supported placeholders.
func (h *ScionHandler) GetDefaults(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"configured":                h.cfg != nil && h.cfg.IsScionConfigured(),
		"hub_url":                   h.hubURL(),
		"default_project_slug":      h.defaultProjectSlug(),
		"default_template":          h.defaultTemplate(),
		"implement_prompt_template": scion.DefaultImplementPrompt,
		"review_prompt_template":    scion.DefaultReviewPrompt,
		"followup_prompt_template":  scion.DefaultFollowupPrompt,
		"placeholders":              scion.AvailablePlaceholders(),
	})
}

// ListAgents returns Scion agents for the project's configured Scion project slug.
func (h *ScionHandler) ListAgents(c *gin.Context) {
	projectID := c.Param("projectId")
	configured := h.cfg != nil && h.cfg.IsScionConfigured() && h.client != nil
	if !configured {
		c.JSON(http.StatusOK, gin.H{
			"configured": false,
			"hub_url":    h.hubURL(),
			"agents":     []scion.Agent{},
		})
		return
	}

	slug := h.defaultProjectSlug()
	if proj, err := h.projectSvc.GetProject(c.Request.Context(), projectID); err == nil && proj != nil && proj.Settings != nil {
		if s := strings.TrimSpace(proj.Settings.ScionProjectSlug); s != "" {
			slug = s
		}
	}

	agents, err := h.client.ListAgents(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"configured": true,
			"hub_url":    h.hubURL(),
			"agents":     []scion.Agent{},
			"error":      err.Error(),
		})
		return
	}
	if agents == nil {
		agents = []scion.Agent{}
	}
	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"hub_url":    h.hubURL(),
		"agents":     agents,
	})
}

// GetItemAgents returns live Scion Hub status for a work item's bound implementation and review agents.
func (h *ScionHandler) GetItemAgents(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")

	item, err := h.itemSvc.GetByID(c.Request.Context(), projectID, itemID)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "work item not found"})
			return
		}
		respondInternalError(c, "failed to load work item", err)
		return
	}

	configured := h.cfg != nil && h.cfg.IsScionConfigured() && h.client != nil
	resolveAgent := func(id, name, harness string) *scion.Agent {
		if strings.TrimSpace(id) == "" {
			return nil
		}
		out := &scion.Agent{
			ID:      id,
			Name:    name,
			Harness: harness,
			Phase:   "unknown",
			URL:     h.agentURL(id),
		}
		if out.Name == "" {
			out.Name = id
		}
		if configured {
			if live, err := h.client.GetAgent(c.Request.Context(), id); err == nil && live != nil {
				if live.Name != "" {
					out.Name = live.Name
				}
				if live.Phase != "" {
					out.Phase = live.Phase
				}
				if live.Activity != "" {
					out.Activity = live.Activity
				}
				if live.Harness != "" {
					out.Harness = live.Harness
				}
			}
		}
		return out
	}

	c.JSON(http.StatusOK, gin.H{
		"configured":   configured,
		"hub_url":      h.hubURL(),
		"impl_agent":   resolveAgent(item.ImplAgentID, item.ImplAgentName, item.ImplAgentHarness),
		"review_agent": resolveAgent(item.ReviewAgentID, item.ReviewAgentName, item.ReviewAgentHarness),
	})
}

func (h *ScionHandler) hubURL() string {
	if h.cfg == nil {
		return ""
	}
	return strings.TrimRight(h.cfg.ScionHubURL, "/")
}

func (h *ScionHandler) agentURL(agentID string) string {
	base := h.hubURL()
	if base == "" || strings.TrimSpace(agentID) == "" {
		return ""
	}
	return base + "/agents/" + agentID
}

func (h *ScionHandler) defaultProjectSlug() string {
	if h.cfg != nil && strings.TrimSpace(h.cfg.ScionDefaultProjectSlug) != "" {
		return h.cfg.ScionDefaultProjectSlug
	}
	return "global"
}

func (h *ScionHandler) defaultTemplate() string {
	if h.cfg != nil && strings.TrimSpace(h.cfg.ScionDefaultTemplate) != "" {
		return h.cfg.ScionDefaultTemplate
	}
	return "jynko-dev"
}
