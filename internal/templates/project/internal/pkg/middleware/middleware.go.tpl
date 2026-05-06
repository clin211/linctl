package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/clin211/linhub/errx"

	"{{.Module}}/internal/pkg/contextx"
)

// RequestID 为每个请求注入唯一的请求 ID 到上下文与响应头中。
// 请求头名称与 github.com/clin211/linhub/core 中的 errx.HeaderRequestID 保持一致。
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
