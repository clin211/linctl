package handler

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/core"
	"{{ .Project.Metadata.Module }}/pkg/version"
)

// Healthz 返回 200 OK 表示当前进程存活（不会检查下游依赖）。
//
// 真实的就绪态检测请额外实现 /readyz 并在其中 ping 数据库 / 缓存 / MQ。
//
// v1.HealthzResponse / v1.ServiceStatus 由 pkg/api/{{ .Component.Name }}/v1
// 的 proto 文件生成（healthz.proto）。
func (h *Handler) Healthz(c *gin.Context) {
	slog.InfoContext(c.Request.Context(), "Healthz handler is called", "method", "Healthz", "status", "healthy")
	core.WriteResponse(c, v1.HealthzResponse{
		Status:    v1.ServiceStatus_Healthy,
		Version:   version.Get().Text(),
		Timestamp: time.Now().Format(time.DateTime),
	}, nil)
}
