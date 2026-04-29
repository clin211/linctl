package server

import (
	"context"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

// Server 是各种服务器实现共同满足的接口。
type Server interface {
	// RunOrDie 启动服务器；如果失败会终止进程（OrDie 即此意）。
	RunOrDie()
	// GracefulStop 优雅地关停服务器，关停时遵循 ctx 的超时控制。
	GracefulStop(ctx context.Context)
}

// Serve 启动 srv 并阻塞，直到 ctx 被取消；ctx 关闭时调用 srv.GracefulStop。
func Serve(ctx context.Context, srv Server) error {
	go srv.RunOrDie()

	// 阻塞，直到 ctx 被取消。
	<-ctx.Done()

	// 优雅停机
	klog.InfoS("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv.GracefulStop(ctx)

	klog.InfoS("Server exited successfully.")

	return nil
}

// protocolName 根据 *http.Server 是否启用 TLS 返回 "http" / "https"。
func protocolName(server *http.Server) string {
	if server.TLSConfig != nil {
		return "https"
	}
	return "http"
}
