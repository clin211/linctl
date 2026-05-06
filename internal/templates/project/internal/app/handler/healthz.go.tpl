package handler

import (
	"net/http"
	"time"

	"github.com/clin211/linhub/log"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(v1 *gin.RouterGroup, h *Handler) {
		v1.GET("/healthz", h.Healthz)
	})
}

// Healthz 返回服务健康状态。
func (h *Handler) Healthz(c *gin.Context) {
	log.W(c.Request.Context()).Infow("Healthz called", "status", "healthy")
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
