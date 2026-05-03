// Package middleware provides gin HTTP middleware for {{.AppName | Title}}.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/clin211/linhub/errx"

	"{{.Module}}/internal/pkg/contextx"
)

// RequestID injects a unique request ID into each request context and response header.
// Header name matches github.com/clin211/linhub/core (errx.HeaderRequestID).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get(errx.HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := contextx.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set(errx.HeaderRequestID, requestID)

		c.Next()
	}
}
