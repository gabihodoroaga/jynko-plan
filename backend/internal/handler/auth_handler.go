package handler

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"jynko/internal/middleware"
	"jynko/internal/model"
)

type userProfileOverride struct {
	Name        string
	AvatarURL   string
	AvatarColor string
}

// AuthHandler handles authentication identity inspection and profile endpoints.
type AuthHandler struct {
	mu       sync.RWMutex
	profiles map[string]userProfileOverride
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		profiles: make(map[string]userProfileOverride),
	}
}

// GetMe returns the current IAP-authenticated user identity (/api/v1/auth/me).
func (h *AuthHandler) GetMe(c *gin.Context) {
	userCtx, ok := middleware.GetUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	h.mu.RLock()
	override, hasOverride := h.profiles[userCtx.UserID]
	h.mu.RUnlock()
	if hasOverride {
		if override.Name != "" {
			userCtx.Name = override.Name
		}
		userCtx.AvatarURL = override.AvatarURL
		userCtx.AvatarColor = override.AvatarColor
	}
	c.JSON(http.StatusOK, userCtx)
}

type updateMeRequest struct {
	Name        *string `json:"name"`
	AvatarURL   *string `json:"avatar_url"`
	AvatarColor *string `json:"avatar_color"`
}

// UpdateMe updates the current user's display name and avatar while keeping IAP email read-only (PATCH /api/v1/auth/me).
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userCtx, ok := middleware.GetUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.mu.Lock()
	curr := h.profiles[userCtx.UserID]
	if curr.Name == "" {
		curr.Name = userCtx.Name
	}
	if req.Name != nil {
		if trimmed := strings.TrimSpace(*req.Name); trimmed != "" {
			curr.Name = trimmed
		}
	}
	if req.AvatarURL != nil {
		curr.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}
	if req.AvatarColor != nil {
		curr.AvatarColor = strings.TrimSpace(*req.AvatarColor)
	}
	h.profiles[userCtx.UserID] = curr
	h.mu.Unlock()

	out := model.UserContext{
		UserID:      userCtx.UserID,
		Email:       userCtx.Email,
		Name:        curr.Name,
		AvatarURL:   curr.AvatarURL,
		AvatarColor: curr.AvatarColor,
	}
	c.JSON(http.StatusOK, out)
}
