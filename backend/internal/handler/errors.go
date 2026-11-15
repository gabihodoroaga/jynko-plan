package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// respondInternalError records the underlying error in Gin's context and structured logs,
// while returning a safe generic JSON error message to the client.
func respondInternalError(c *gin.Context, publicMsg string, err error) {
	if err != nil {
		_ = c.Error(err)
		slog.ErrorContext(c.Request.Context(), publicMsg,
			slog.String("error", err.Error()),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": publicMsg})
}
