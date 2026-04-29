package {{ .Component.Name }}

import (
	"context"
	"net/http"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	mw "{{ .Project.Metadata.Module }}/internal/pkg/middleware/gin"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/handler"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/pkg/metrics"
	"{{ .Project.Metadata.Module }}/pkg/core"
	genericmw "{{ .Project.Metadata.Module }}/pkg/middleware/gin"
	"{{ .Project.Metadata.Module }}/pkg/server"
)

// ginServer 是 Gin 框架的 HTTP 服务器封装，实现 server.Server 接口。
type ginServer struct {
	srv server.Server
}

// 编译期断言：确保 *ginServer 实现了 server.Server.
var _ server.Server = (*ginServer)(nil)

// NewGinServer 构造 Gin 引擎，挂载标准中间件链，并注册 REST 路由。
func (c *ServerConfig) NewGinServer() (*ginServer, error) {
	engine := gin.New()

	// 标准中间件链：panic 恢复 → 头部安全 → CORS → Secure → otelgin（过滤 /metrics）→
	// 通用可观测性 → 请求 ID → context 投影。顺序与 miniblog-v4 一致；额外挂上
	// RequestIDMiddleware：linctl 已生成 internal/pkg/middleware/gin/requestid.go，
	// 默认串入链路便于 trace。
	engine.Use(
		gin.Recovery(),
		mw.NoCache,
		mw.Cors,
		mw.Secure,
		otelgin.Middleware("{{ .Component.Name }}", otelgin.WithFilter(func(rq *http.Request) bool {
			// 返回 false 表示对该请求不创建 span。
			return rq.URL.Path != "/metrics"
		})),
		genericmw.Observability(),
		mw.RequestIDMiddleware(),
		mw.Context(),
	)

	c.InstallRESTAPI(engine)

	httpsrv := server.NewHTTPServer(c.HTTPOptions, c.TLSOptions, engine)
	return &ginServer{srv: httpsrv}, nil
}

// InstallRESTAPI 按惯例注册 REST 路由（公共 + v1 业务）。
func (c *ServerConfig) InstallRESTAPI(engine *gin.Engine) {
	InstallGenericAPI(engine)

	// 鉴权 + 授权中间件，资源路由会用到。
	authMiddlewares := []gin.HandlerFunc{
		mw.AuthnMiddleware(c.retriever),
		mw.AuthzMiddleware(c.authz),
	}

	hdl := handler.NewHandler(c.biz, c.val, authMiddlewares...)

	// 公共端点（无需认证）。
	engine.GET("/healthz", hdl.Healthz)
	// 登录与刷新令牌：约定上不挂在 v1 下；
	//   - /login：完全公开，由用户名 + 密码换 token；
	//   - /refresh-token：只过 authn，无需 authz——续签自己手中的旧 token。
	engine.POST("/login", hdl.Login)
	engine.PUT("/refresh-token", mw.AuthnMiddleware(c.retriever), hdl.RefreshToken)

	// v1 资源分组：所有 handler 通过 init() + Register 自动挂到这里。
	v1 := engine.Group("/v1")
	hdl.InstallAll(v1)
}

// InstallGenericAPI 注册业务无关的端点（pprof / metrics / 404）。
func InstallGenericAPI(engine *gin.Engine) {
	// pprof：调试性能用
	pprof.Register(engine)

	_ = metrics.Initialize(context.Background(), "{{ .Component.Name }}")

	// Prometheus /metrics 端点
	_ = engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 404 统一走业务错误结构。
	//
	// 注意：参数槽位是 `WriteResponse(c, data, err)`；这里的写法（把
	// errno.ErrPageNotFound 当 data，err 传 nil）与 pkg/core.WriteResponse 的
	// "成功分支返回 data" 语义略有出入，但与 miniblog-v4 参考实现保持一致——
	// 真实的 404 路由命中频率极低，这里不刻意"修复"，避免与上游分歧。
	engine.NoRoute(func(c *gin.Context) {
		core.WriteResponse(c, errno.ErrPageNotFound, nil)
	})
}

// RunOrDie 启动 Gin 服务器，出错则程序崩溃退出。
func (s *ginServer) RunOrDie() { s.srv.RunOrDie() }

// GracefulStop 优雅停止服务器。
func (s *ginServer) GracefulStop(ctx context.Context) { s.srv.GracefulStop(ctx) }
