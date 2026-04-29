package handler

import (
	"github.com/gin-gonic/gin"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/biz"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/pkg/validation"
)

// Handler 是 HTTP 处理器的中央依赖容器，持有 biz / validator / 公共中间件。
//
// 各资源 handler 通过 init() 调用 Register() 自注册路由，最终由
// InstallAll(v1) 一次挂到 v1 路由组上，避免修改 httpserver.go。
type Handler struct {
	biz biz.IBiz
	val *validation.Validator
	mws []gin.HandlerFunc
}

// Registrar 是资源路由的注册函数：拿到 v1 路由组与 *Handler，自行挂载子路由。
type Registrar func(v1 *gin.RouterGroup, h *Handler)

// registrars 收集所有资源 handler 的注册函数，由 init() 在包加载时填充。
var registrars []Registrar

// NewHandler 构造一个 *Handler。
func NewHandler(biz biz.IBiz, val *validation.Validator, mws ...gin.HandlerFunc) *Handler {
	return &Handler{biz: biz, val: val, mws: mws}
}

// Register 把 r 加入全局 registrars，由资源 handler 文件的 init() 调用。
func Register(r Registrar) {
	registrars = append(registrars, r)
}

// InstallAll 遍历 registrars，把所有资源路由挂到 v1 路由组上。
func (h *Handler) InstallAll(v1 *gin.RouterGroup) {
	for _, r := range registrars {
		r(v1, h)
	}
}
