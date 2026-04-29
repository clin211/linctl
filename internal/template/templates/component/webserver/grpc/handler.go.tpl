// Package handler implements gRPC service handlers for {{ .Component.Name }}.
//
// This package is intentionally minimal until you generate stubs:
//
//	make protoc            # generate api.pb.go + api_grpc.pb.go
//	# then register your service in internal/{{ .Component.Name }}/server.go:
//	#   v1.RegisterAPIServerServer(srv, handler.New())
package handler

// Ping is a placeholder helper exposed so the package compiles before
// protoc-generated stubs are available. Replace with real handler types
// once the proto stubs exist.
func Ping() string {
	return "pong"
}
