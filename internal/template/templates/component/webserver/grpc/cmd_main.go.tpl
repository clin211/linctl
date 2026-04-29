package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
{{- if .Component.GrpcGateway }}
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
{{- end }}

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}"
{{- if .Component.GrpcGateway }}
	v1 "{{ .Project.Metadata.Module }}/api/{{ .Component.Name }}/v1"
{{- end }}
)

func main() {
{{- if .Component.GrpcGateway }}
	grpcPort := envInt("GRPC_LISTEN_PORT", {{ if .Component.GRPCPort }}{{ .Component.GRPCPort }}{{ else }}9090{{ end }})
	httpPort := envInt("HTTP_LISTEN_PORT", {{ if .Component.Port }}{{ .Component.Port }}{{ else }}8080{{ end }})
	grpcBind := envOr("GRPC_ADDR", fmt.Sprintf(":%d", grpcPort))
	httpAddr := envOr("HTTP_ADDR", fmt.Sprintf(":%d", httpPort))

	srv := {{ .Component.Name }}.NewServer()
	lis, err := net.Listen("tcp", grpcBind)
	if err != nil {
		log.Fatalf("listen gRPC %s: %v", grpcBind, err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("gRPC listening on %s", lis.Addr().String())
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("gRPC serve: %v", err)
		}
	}()

	// brief wait so the gRPC server accepts connections (local dev; use readiness in production).
	time.Sleep(150 * time.Millisecond)
	_, portStr, serr := net.SplitHostPort(lis.Addr().String())
	if serr != nil {
		log.Fatalf("parse listener addr: %v", serr)
	}
	grpcClientTarget := net.JoinHostPort("127.0.0.1", portStr)
	conn, err := grpc.NewClient(grpcClientTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc client: %v", err)
	}
	defer conn.Close()

	mux := runtime.NewServeMux()
	if err := v1.RegisterAPIServerHandler(ctx, mux, conn); err != nil {
		log.Fatalf("register gateway: %v", err)
	}
	httpSrv := &http.Server{Addr: httpAddr, Handler: mux}
	go func() {
		log.Printf("grpc-gateway (HTTP) listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	_ = httpSrv.Shutdown(context.Background())
	srv.GracefulStop()
	log.Println("bye")
	return
{{ else }}
	addr := envOr("ADDR", ":{{ if .Component.GRPCPort }}{{ .Component.GRPCPort }}{{ else }}9090{{ end }}")

	srv := {{ .Component.Name }}.NewServer()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("gRPC listening on %s", addr)
		if err := srv.Serve(listener); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	srv.GracefulStop()
	log.Println("bye")
{{ end }}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
{{- if .Component.GrpcGateway }}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
{{- end }}
