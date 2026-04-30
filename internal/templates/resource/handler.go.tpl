package handler

import (
	"{{.Module}}/pkg/core"
	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"

	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(rg1 *gin.RouterGroup, handler *Handler) {
		rg := rg1.Group("/{{.Resource | Lower | Plural}}")
		rg.POST("", handler.Create{{.Resource | Pascal}})
		rg.PUT(":{{.Resource | LowerCamel}}ID", handler.Update{{.Resource | Pascal}})
		rg.DELETE(":{{.Resource | LowerCamel}}ID", handler.Delete{{.Resource | Pascal}})
		rg.GET(":{{.Resource | LowerCamel}}ID", handler.Get{{.Resource | Pascal}})
		rg.GET("", handler.List{{.Resource | Pascal}})
	})
}

// Create{{.Resource | Pascal}} creates a new {{.Resource | Pascal}}.
func (h *Handler) Create{{.Resource | Pascal}}(c *gin.Context) {
	core.HandleJSONRequest(c, func(c *gin.Context, req *v1.Create{{.Resource | Pascal}}Request) (*v1.Create{{.Resource | Pascal}}Response, error) {
		return h.biz.{{.Resource | Pascal}}V1().Create(c.Request.Context(), req)
	})
}

// Update{{.Resource | Pascal}} updates a {{.Resource | Pascal}}.
func (h *Handler) Update{{.Resource | Pascal}}(c *gin.Context) {
	core.HandleJSONRequest(c, func(c *gin.Context, req *v1.Update{{.Resource | Pascal}}Request) (*v1.Update{{.Resource | Pascal}}Response, error) {
		return h.biz.{{.Resource | Pascal}}V1().Update(c.Request.Context(), req)
	})
}

// Delete{{.Resource | Pascal}} deletes a {{.Resource | Pascal}}.
func (h *Handler) Delete{{.Resource | Pascal}}(c *gin.Context) {
	core.HandleJSONRequest(c, func(c *gin.Context, req *v1.Delete{{.Resource | Pascal}}Request) (*v1.Delete{{.Resource | Pascal}}Response, error) {
		return h.biz.{{.Resource | Pascal}}V1().Delete(c.Request.Context(), req)
	})
}

// Get{{.Resource | Pascal}} retrieves a {{.Resource | Pascal}}.
func (h *Handler) Get{{.Resource | Pascal}}(c *gin.Context) {
	core.HandleUriRequest(c, func(c *gin.Context, req *v1.Get{{.Resource | Pascal}}Request) (*v1.Get{{.Resource | Pascal}}Response, error) {
		return h.biz.{{.Resource | Pascal}}V1().Get(c.Request.Context(), req)
	})
}

// List{{.Resource | Pascal}} lists {{.Resource | Pascal}} entries.
func (h *Handler) List{{.Resource | Pascal}}(c *gin.Context) {
	core.HandleQueryRequest(c, func(c *gin.Context, req *v1.List{{.Resource | Pascal}}Request) (*v1.List{{.Resource | Pascal}}Response, error) {
		return h.biz.{{.Resource | Pascal}}V1().List(c.Request.Context(), req)
	})
}
