package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(v1 *gin.RouterGroup, h *Handler) {
		v1.GET("/healthz", h.Healthz)
	})
}

// Healthz returns the service health status.
func (h *Handler) Healthz(c *gin.Context) {
	slog.InfoContext(c.Request.Context(), "Healthz called", "status", "healthy")
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
