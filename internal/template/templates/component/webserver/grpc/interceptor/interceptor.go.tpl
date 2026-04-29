package interceptor

import (
	"context"
	"log"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerLogging logs each unary RPC method.
func UnaryServerLogging(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	log.Printf("grpc %s", info.FullMethod)
	return handler(ctx, req)
}

// UnaryServerRecovery turns panics in handlers into gRPC Internal errors.
func UnaryServerRecovery(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("grpc panic: %v\n%s", p, debug.Stack())
			err = status.Errorf(codes.Internal, "panic: %v", p)
		}
	}()
	return handler(ctx, req)
}
