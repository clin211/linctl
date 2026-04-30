// Package core provides HTTP handler utilities for {{.AppName | Title}}.
package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// WriteResponse writes a standardized JSON response to the gin context.
// On error it returns a 500 with the error message; on success it returns 200 with data.
func WriteResponse(c *gin.Context, data any, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500001,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// Handler is a generic business handler function type.
type Handler[T any, R any] func(c *gin.Context, req *T) (R, error)

// HandleJSONRequest binds a JSON body and dispatches to handler.
func HandleJSONRequest[T any, R any](c *gin.Context, handler func(*gin.Context, *T) (R, error)) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteResponse(c, nil, err)
		return
	}
	resp, err := handler(c, &req)
	WriteResponse(c, resp, err)
}

// HandleQueryRequest binds query parameters and dispatches to handler.
func HandleQueryRequest[T any, R any](c *gin.Context, handler func(*gin.Context, *T) (R, error)) {
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		WriteResponse(c, nil, err)
		return
	}
	resp, err := handler(c, &req)
	WriteResponse(c, resp, err)
}

// HandleUriRequest binds URI parameters and dispatches to handler.
func HandleUriRequest[T any, R any](c *gin.Context, handler func(*gin.Context, *T) (R, error)) {
	var req T
	if err := c.ShouldBindUri(&req); err != nil {
		WriteResponse(c, nil, err)
		return
	}
	resp, err := handler(c, &req)
	WriteResponse(c, resp, err)
}
