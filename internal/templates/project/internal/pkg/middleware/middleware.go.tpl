// Package middleware provides gin HTTP middleware for {{.AppName | Title}}.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"{{.Module}}/internal/pkg/contextx"
	"{{.Module}}/internal/pkg/known"
)

// RequestID injects a unique request ID into each request context and response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get(known.XRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := contextx.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set(known.XRequestID, requestID)

		c.Next()
	}
}
