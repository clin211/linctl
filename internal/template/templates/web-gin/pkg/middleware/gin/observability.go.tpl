package gin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// 标准 trace 头部 key 常量。
const (
	// W3C Trace Context 标准（推荐）
	TraceParentHeaderKey = "traceparent"

	// 简单 trace ID（最广泛使用）
	TraceIDHeaderKey = "X-Trace-Id"

	// 通用请求 ID（兼容性最好）
	RequestIDHeaderKey = "X-Request-Id"

	// W3C tracestate 附加上下文
	TraceStateHeaderKey = "tracestate"
)

// TraceInjectionMode 控制 trace 信息以何种形式写入 HTTP 响应头。
type TraceInjectionMode int

const (
	// InjectW3CTraceContext 注入完整的 W3C trace context（推荐）
	InjectW3CTraceContext TraceInjectionMode = iota
	// InjectTraceIDOnly 只注入 trace ID
	InjectTraceIDOnly
	// InjectBoth 同时注入 W3C 与简单 trace ID
	InjectBoth
	// InjectNone 关闭 trace 头注入
	InjectNone
)

// ObservabilityOptions 收集中间件的可调配置项。
type ObservabilityOptions struct {
	TraceInjectionMode TraceInjectionMode
	CustomTraceHeader  string   // 自定义 trace ID 的头部名
	SkipPaths          []string // 跳过日志的路径（支持通配符）
}

// Option 是 ObservabilityOptions 的函数选项类型。
type Option func(*ObservabilityOptions)

// WithTraceInjection 配置 trace 注入模式。
func WithTraceInjection(mode TraceInjectionMode) Option {
	return func(o *ObservabilityOptions) {
		o.TraceInjectionMode = mode
	}
}

// WithCustomTraceHeader 自定义 trace ID 的响应头名。
func WithCustomTraceHeader(headerName string) Option {
	return func(o *ObservabilityOptions) {
		o.CustomTraceHeader = headerName
	}
}

// WithSkipPaths 配置需要跳过日志的路径列表（支持精确匹配与通配符）。
func WithSkipPaths(paths ...string) Option {
	return func(o *ObservabilityOptions) {
		o.SkipPaths = append(o.SkipPaths, paths...)
	}
}

// WithSkipMetrics 是便捷函数：跳过常见的指标 / 健康检查端点的日志。
func WithSkipMetrics() Option {
	return func(o *ObservabilityOptions) {
		commonPaths := []string{
			"/health",
			"/healthz",
			"/health/*",
			"/ready",
			"/readiness",
			"/live",
			"/liveness",
			"/metrics",
			"/prometheus",
			"/status",
			"/ping",
			"/version",
			"/info",
			"/favicon.ico",
			"/robots.txt",
		}
		o.SkipPaths = append(o.SkipPaths, commonPaths...)
	}
}

// Observability 返回一个 Gin 中间件：记录请求日志、注入 trace 头、可选抓 body。
func Observability(opts ...Option) gin.HandlerFunc {
	// 默认配置
	config := &ObservabilityOptions{
		TraceInjectionMode: InjectTraceIDOnly,
		SkipPaths:          []string{"/metrics"}, // 默认跳过 /metrics
	}

	// 应用所有可选项
	for _, opt := range opts {
		opt(config)
	}

	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// 当前请求是否在跳过列表里
		shouldSkip := shouldSkipPath(c.Request.URL.Path, c.Request.Method, config.SkipPaths)
		if shouldSkip {
			c.Next()
			return
		}

		// 提前提取 trace 信息
		span := trace.SpanFromContext(ctx)
		spanCtx := span.SpanContext()

		// 按配置注入 trace 头
		injectTraceHeaders(c, spanCtx, config)

		var requestBody string
		var responseBuffer bytes.Buffer

		// 仅当 debug 级别开启时才抓 request / response body
		isDebugLevel := isDebugEnabled()

		if isDebugLevel && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		if isDebugLevel {
			writer := &bodyCaptureWriter{ResponseWriter: c.Writer, body: &responseBuffer}
			c.Writer = writer
		}

		c.Next()

		duration := time.Since(start).Seconds()

		// 拼装结构化日志
		httpData := map[string]any{
			"request": map[string]any{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			},
			"response": map[string]any{
				"status_code": c.Writer.Status(),
			},
		}

		if isDebugLevel {
			httpData["request"].(map[string]any)["body"] = map[string]any{
				"content": requestBody,
				"bytes":   len(requestBody),
			}

			httpData["response"].(map[string]any)["body"] = map[string]any{
				"content": responseBuffer.String(),
				"bytes":   responseBuffer.Len(),
			}
		}

		logLevel := slog.LevelInfo
		if isDebugLevel {
			logLevel = slog.LevelDebug
		}

		slog.Log(ctx, logLevel, "HTTP request completed",
			"duration_sec", duration,
			"source", map[string]any{"ip": c.ClientIP()},
			"http", httpData,
			"user", map[string]any{"agent": c.Request.UserAgent()},
			"trace", map[string]any{"id": spanCtx.TraceID().String()},
			"span", map[string]any{"id": spanCtx.SpanID().String()},
		)
	}
}

// shouldSkipPath 判断当前路径是否在跳过列表里。
func shouldSkipPath(path, method string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if matchPath(path, method, skipPath) {
			return true
		}
	}
	return false
}

// matchPath 把请求路径与 skip 模式匹配（支持 "GET /xxx" 形式按方法过滤）。
func matchPath(requestPath, method, pattern string) bool {
	// 处理 "METHOD path" 形式（例如 "GET /metrics"）
	if strings.Contains(pattern, " ") {
		parts := strings.SplitN(pattern, " ", 2)
		if len(parts) == 2 {
			patternMethod := strings.ToUpper(strings.TrimSpace(parts[0]))
			patternPath := strings.TrimSpace(parts[1])

			if patternMethod != strings.ToUpper(method) {
				return false
			}
			return matchPathPattern(requestPath, patternPath)
		}
	}

	// 仅按路径匹配
	return matchPathPattern(requestPath, pattern)
}

// matchPathPattern 把 path 与 pattern 匹配（支持通配符与前缀匹配）。
func matchPathPattern(path, pattern string) bool {
	// 完全匹配
	if path == pattern {
		return true
	}

	// 通配符匹配
	if strings.Contains(pattern, "*") {
		return matchWildcard(path, pattern)
	}

	// 前缀匹配（pattern 以 "/" 结尾时）
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}

	return false
}

// matchWildcard 实现简单的通配符匹配。
func matchWildcard(text, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 包含匹配
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := pattern[1 : len(pattern)-1]
		return strings.Contains(text, substr)
	}

	// 后缀匹配
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	}

	// 前缀匹配
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	}

	return text == pattern
}

// injectTraceHeaders 根据配置把 trace 信息写入响应头。
func injectTraceHeaders(c *gin.Context, spanCtx trace.SpanContext, config *ObservabilityOptions) {
	if !spanCtx.IsValid() {
		return
	}

	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	switch config.TraceInjectionMode {
	case InjectW3CTraceContext:
		// W3C Trace Context 格式：version-trace_id-parent_id-trace_flags
		traceFlags := "01" // sampled
		if !spanCtx.IsSampled() {
			traceFlags = "00" // not sampled
		}
		traceparent := fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags)
		c.Header(TraceParentHeaderKey, traceparent)

	case InjectTraceIDOnly:
		headerKey := TraceIDHeaderKey
		if config.CustomTraceHeader != "" {
			headerKey = config.CustomTraceHeader
		}
		c.Header(headerKey, traceID)

	case InjectBoth:
		// W3C 格式
		traceFlags := "01"
		if !spanCtx.IsSampled() {
			traceFlags = "00"
		}
		traceparent := fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags)
		c.Header(TraceParentHeaderKey, traceparent)

		// 简单 trace ID
		headerKey := TraceIDHeaderKey
		if config.CustomTraceHeader != "" {
			headerKey = config.CustomTraceHeader
		}
		c.Header(headerKey, traceID)

	case InjectNone:
		// 不注入任何 trace 头
	}
}

// ===== 常见配置组合的便捷构造函数 =====

// ObservabilityWithW3CTraceContext 构造启用 W3C trace context 的中间件。
func ObservabilityWithW3CTraceContext() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectW3CTraceContext))
}

// ObservabilityWithTraceID 构造只注入简单 trace ID 的中间件。
func ObservabilityWithTraceID() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectTraceIDOnly))
}

// ObservabilityWithCustomHeader 构造使用自定义头部名注入 trace ID 的中间件。
func ObservabilityWithCustomHeader(headerName string) gin.HandlerFunc {
	return Observability(
		WithTraceInjection(InjectTraceIDOnly),
		WithCustomTraceHeader(headerName),
	)
}

// ObservabilitySkipMetrics 构造跳过常见指标 / 健康检查端点日志的中间件。
func ObservabilitySkipMetrics() gin.HandlerFunc {
	return Observability(WithSkipMetrics())
}

// ObservabilityWithSkipPaths 构造按自定义路径列表跳过日志的中间件。
func ObservabilityWithSkipPaths(paths ...string) gin.HandlerFunc {
	return Observability(WithSkipPaths(paths...))
}

// bodyCaptureWriter 是 gin.ResponseWriter 的包装，复制响应 body 到 buffer。
type bodyCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// isDebugEnabled 判断全局 slog logger 是否启用了 debug 级别。
func isDebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}
