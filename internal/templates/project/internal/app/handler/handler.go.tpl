package handler

import (
	"github.com/gin-gonic/gin"

	"{{.Module}}/internal/{{.AppName}}/biz"
)

// Handler 持有 HTTP handler 所需的依赖。
type Handler struct {
	biz biz.IBiz
}

// Registrar 是向 gin RouterGroup 注册路由的函数类型。
type Registrar func(v1 *gin.RouterGroup, h *Handler)

var registrars []Registrar

// NewHandler 创建一个 Handler 实例。
func NewHandler(biz biz.IBiz) *Handler {
	return &Handler{biz: biz}
}

// Register 将一个 Registrar 追加到全局列表。
// 由各 handler 文件的 init() 函数调用。
func Register(r Registrar) {
	registrars = append(registrars, r)
}

// InstallAll 将所有已注册的路由挂载到给定的 RouterGroup 上。
func (h *Handler) InstallAll(v1 *gin.RouterGroup) {
	for _, r := range registrars {
		r(v1, h)
	}
}
