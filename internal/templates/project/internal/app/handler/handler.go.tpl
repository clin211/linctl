package handler

import (
	"github.com/gin-gonic/gin"

	"{{.Module}}/internal/{{.AppName}}/biz"
)

// Handler holds the dependencies for HTTP handlers.
type Handler struct {
	biz biz.IBiz
}

// Registrar is a function that registers routes on a gin RouterGroup.
type Registrar func(v1 *gin.RouterGroup, h *Handler)

var registrars []Registrar

// NewHandler creates a new Handler instance.
func NewHandler(biz biz.IBiz) *Handler {
	return &Handler{biz: biz}
}

// Register adds a Registrar to the global list.
// Called from init() functions in each handler file.
func Register(r Registrar) {
	registrars = append(registrars, r)
}

// InstallAll installs all registered routes on the given RouterGroup.
func (h *Handler) InstallAll(v1 *gin.RouterGroup) {
	for _, r := range registrars {
		r(v1, h)
	}
}
