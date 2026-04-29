package {{ .Component.Name }}

import (
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/interceptor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// NewServer returns a configured *grpc.Server with the v1 health service
// and server reflection enabled.
//
// After generating stubs via `make protoc` (see api/{{ .Component.Name }}/v1/api.proto),
// register concrete service handlers here, e.g.:
//
//	v1.RegisterAPIServerServer(srv, handler.New())
func NewServer() *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.UnaryServerLogging,
			interceptor.UnaryServerRecovery,
		),
	)

	healthpb.RegisterHealthServer(srv, health.NewServer())
	reflection.Register(srv)

	return srv
}
