package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/biz"
	"{{ .Project.Metadata.Module }}/pkg/core"
)

// init 把 {{ .Custom.ResourcePascal }} 的路由注册到 v1 路由组上。
//
// 利用 `Register()` 自注册：你**无需**在执行 `linctl add api {{ .Custom.ResourcePascal }}` 后
// 手动修改 httpserver.go 或本文件的兄弟文件。
func init() {
	Register(func(v1 *gin.RouterGroup, h *Handler) {
		rg := v1.Group("/{{ kebab (plural .Custom.ResourcePascal) }}")
		// 公共路由（例如注册接口），不挂认证中间件。
		rg.POST("", h.Create{{ .Custom.ResourcePascal }})

		// 需要认证 + 鉴权的路由放到子分组下。
		auth := rg.Group("")
		auth.Use(h.mws...)
		auth.GET("", h.List{{ .Custom.ResourcePascal }}s)
		auth.GET("/:id", h.Get{{ .Custom.ResourcePascal }})
		auth.PUT("/:id", h.Update{{ .Custom.ResourcePascal }})
		auth.DELETE("/:id", h.Delete{{ .Custom.ResourcePascal }})
	})
}

// list{{ .Custom.ResourcePascal }}sQuery 是 List 的 query-string 请求结构。
type list{{ .Custom.ResourcePascal }}sQuery struct {
	// 在这里按需追加分页等字段，将自动从 query 绑定。
}

// uriID 用来从 URI 中绑定 :id。
type uriID struct {
	ID int64 `uri:"id" binding:"required"`
}

// List{{ .Custom.ResourcePascal }}s 列出 {{ .Custom.ResourceLower }}s。
func (h *Handler) List{{ .Custom.ResourcePascal }}s(c *gin.Context) {
	core.HandleQueryRequest(c, func(ctx context.Context, req *list{{ .Custom.ResourcePascal }}sQuery) (any, error) {
		_ = req
		return h.biz.{{ .Custom.ResourcePascal }}V1().List(ctx)
	})
}

// Get{{ .Custom.ResourcePascal }} 根据 ID 返回单个 {{ .Custom.ResourceLower }}。
func (h *Handler) Get{{ .Custom.ResourcePascal }}(c *gin.Context) {
	core.HandleUriRequest(c, func(ctx context.Context, req *uriID) (any, error) {
		return h.biz.{{ .Custom.ResourcePascal }}V1().Get(ctx, req.ID)
	})
}

// Create{{ .Custom.ResourcePascal }} 新建一个 {{ .Custom.ResourceLower }}。
func (h *Handler) Create{{ .Custom.ResourcePascal }}(c *gin.Context) {
	core.HandleJSONRequest(c, func(ctx context.Context, req *biz.Create{{ .Custom.ResourcePascal }}Request) (any, error) {
		return nil, h.biz.{{ .Custom.ResourcePascal }}V1().Create(ctx, req)
	})
}

// Update{{ .Custom.ResourcePascal }} 更新已有 {{ .Custom.ResourceLower }}。
func (h *Handler) Update{{ .Custom.ResourcePascal }}(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	core.HandleJSONRequest(c, func(ctx context.Context, req *biz.Update{{ .Custom.ResourcePascal }}Request) (any, error) {
		return nil, h.biz.{{ .Custom.ResourcePascal }}V1().Update(ctx, id, req)
	})
}

// Delete{{ .Custom.ResourcePascal }} 根据 ID 删除 {{ .Custom.ResourceLower }}。
func (h *Handler) Delete{{ .Custom.ResourcePascal }}(c *gin.Context) {
	core.HandleUriRequest(c, func(ctx context.Context, req *uriID) (any, error) {
		return nil, h.biz.{{ .Custom.ResourcePascal }}V1().Delete(ctx, req.ID)
	})
}
