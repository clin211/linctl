package gin

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"{{ .Project.Metadata.Module }}/pkg/core"

	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
)

// Authorizer 用于定义授权接口的实现.
type Authorizer interface {
	Authorize(subject, object, action string) (bool, error)
}

// AuthzMiddleware 是一个 Gin 中间件，用于进行请求授权.
func AuthzMiddleware(authorizer Authorizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject := contextx.UserID(c.Request.Context())
		object := c.Request.URL.Path
		action := c.Request.Method

		// 记录授权上下文信息
		slog.Info("Build authorize context", "subject", subject, "object", object, "action", action)

		// 调用授权接口进行验证
		if allowed, err := authorizer.Authorize(subject, object, action); err != nil || !allowed {
			core.WriteResponse(c, nil, errno.ErrPermissionDenied.WithMessage(
				fmt.Sprintf("access denied: subject=%s, object=%s, action=%s, reason=%v",
					subject,
					object,
					action,
					err,
				)))
			c.Abort()
			return
		}

		c.Next() // 继续处理请求
	}
}
