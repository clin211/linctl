package server

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"

	"k8s.io/klog/v2"

	genericoptions "{{ .Project.Metadata.Module }}/pkg/options"
)

// HTTPServer 是 *http.Server 的轻量封装，实现 Server 接口。
type HTTPServer struct {
	srv *http.Server
}

// NewHTTPServer 根据 HTTP / TLS 选项与 handler 构造一个 *HTTPServer。
func NewHTTPServer(httpOptions *genericoptions.HTTPOptions, tlsOptions *genericoptions.TLSOptions, handler http.Handler) *HTTPServer {
	var tlsConfig *tls.Config
	if tlsOptions != nil && tlsOptions.UseTLS {
		tlsConfig = tlsOptions.MustTLSConfig()
	}

	return &HTTPServer{
		srv: &http.Server{
			Addr:      httpOptions.Addr,
			Handler:   handler,
			TLSConfig: tlsConfig,
		},
	}
}

// RunOrDie 启动 HTTP 服务器；监听异常时通过 klog.Fatalf 终止进程。
func (s *HTTPServer) RunOrDie() {
	klog.InfoS("Start to listening the incoming requests", "protocol", protocolName(s.srv), "addr", s.srv.Addr)
	// 默认按明文启动 HTTP；启用 TLS 时切换到 ListenAndServeTLS。
	serveFn := func() error { return s.srv.ListenAndServe() }
	if s.srv.TLSConfig != nil {
		serveFn = func() error { return s.srv.ListenAndServeTLS("", "") }
	}

	if err := serveFn(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		klog.Fatalf("Failed to server HTTP(s) server: %v", err)
	}
}

// GracefulStop 优雅地关停 HTTP 服务器。
func (s *HTTPServer) GracefulStop(ctx context.Context) {
	klog.InfoS("Gracefully stop HTTP(s) server")
	if err := s.srv.Shutdown(ctx); err != nil {
		klog.ErrorS(err, "HTTP(s) server forced to shutdown")
	}
}
