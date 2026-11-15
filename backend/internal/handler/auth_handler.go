package handler

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

const (
	maxDisplayNameLength = 120
	maxAvatarColorLength = 64
	maxAvatarURLLength   = 512 * 1024 // 512 KiB (fits Firestore 1 MiB document limit safely)
	maxProfileBodyBytes  = 768 * 1024
)

type userProfileOverride struct {
	Name        string
	AvatarURL   string
	AvatarColor string
}

// AuthHandler handles authentication identity inspection and profile endpoints.
type AuthHandler struct {
	repo     repository.ProfileRepository
	mu       sync.RWMutex
	profiles map[string]userProfileOverride
}

// NewAuthHandler creates a new AuthHandler with in-memory storage (for tests).
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		profiles: make(map[string]userProfileOverride),
	}
}

// NewAuthHandlerWithRepo creates a Firestore-backed AuthHandler.
func NewAuthHandlerWithRepo(repo repository.ProfileRepository) *AuthHandler {
	return &AuthHandler{
		repo:     repo,
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

	if h.repo != nil {
		prof, err := h.repo.Get(c.Request.Context(), userCtx.UserID)
		if err == nil && prof != nil {
			if prof.Name != "" {
				userCtx.Name = prof.Name
			}
			userCtx.AvatarURL = prof.AvatarURL
			userCtx.AvatarColor = prof.AvatarColor
			c.JSON(http.StatusOK, userCtx)
			return
		}
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

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxProfileBodyBytes)

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "profile payload exceeds maximum allowed size"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	curr := userProfileOverride{Name: userCtx.Name}
	if h.repo != nil {
		if existing, err := h.repo.Get(c.Request.Context(), userCtx.UserID); err == nil && existing != nil {
			if existing.Name != "" {
				curr.Name = existing.Name
			}
			curr.AvatarURL = existing.AvatarURL
			curr.AvatarColor = existing.AvatarColor
		}
	} else {
		h.mu.RLock()
		if existing, has := h.profiles[userCtx.UserID]; has {
			if existing.Name != "" {
				curr.Name = existing.Name
			}
			curr.AvatarURL = existing.AvatarURL
			curr.AvatarColor = existing.AvatarColor
		}
		h.mu.RUnlock()
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if len(trimmed) > maxDisplayNameLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name must be at most 120 characters"})
			return
		}
		if trimmed != "" {
			curr.Name = trimmed
		}
	}
	if req.AvatarURL != nil {
		trimmed := strings.TrimSpace(*req.AvatarURL)
		if !isValidAvatarURL(trimmed) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar_url: must be empty, http(s) URL, or base64 PNG/JPEG/GIF/WebP data URI under 512KB"})
			return
		}
		curr.AvatarURL = trimmed
	}
	if req.AvatarColor != nil {
		trimmed := strings.TrimSpace(*req.AvatarColor)
		if len(trimmed) > maxAvatarColorLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "avatar_color is too long"})
			return
		}
		curr.AvatarColor = trimmed
	}

	if h.repo != nil {
		err := h.repo.Upsert(c.Request.Context(), &repository.UserProfile{
			UserID:      userCtx.UserID,
			Email:       userCtx.Email,
			Name:        curr.Name,
			AvatarURL:   curr.AvatarURL,
			AvatarColor: curr.AvatarColor,
		})
		if err != nil {
			respondInternalError(c, "failed to save user profile", err)
			return
		}
	} else {
		h.mu.Lock()
		h.profiles[userCtx.UserID] = curr
		h.mu.Unlock()
	}

	out := model.UserContext{
		UserID:      userCtx.UserID,
		Email:       userCtx.Email,
		Name:        curr.Name,
		AvatarURL:   curr.AvatarURL,
		AvatarColor: curr.AvatarColor,
	}
	c.JSON(http.StatusOK, out)
}

func isValidAvatarURL(raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > maxAvatarURLLength {
		return false
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		return true
	}
	for _, prefix := range []string{
		"data:image/png;base64,",
		"data:image/jpeg;base64,",
		"data:image/jpg;base64,",
		"data:image/gif;base64,",
		"data:image/webp;base64,",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
