package gin

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
)

// Context 是一个 Gin 中间件，把 OTel span 中的 traceID 注入到请求 context 中，
// 让下游的业务函数可以通过 contextx.TraceID(ctx) 取到当前 trace ID。
func Context() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从当前 span 中读取 traceID
		traceID := trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()

		// 把 traceID 写入 context 并替换原 request 的 context
		ctx := contextx.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
