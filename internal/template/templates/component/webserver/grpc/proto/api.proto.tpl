syntax = "proto3";

package {{ .Component.Name }}.v1;
{{- if .Component.GrpcGateway }}

import "google/api/annotations.proto";
{{- end }}

option go_package = "{{ .Project.Metadata.Module }}/api/{{ .Component.Name }}/v1;v1";

// APIServer is the public gRPC surface for {{ .Component.Name }}.
//
// `linctl add api <Resource>` will inject CRUD rpc methods here via
// AddProtoRPCMutator (see lin/docs/07-ast-injection.md §7.6).
service APIServer {
{{- if .Component.GrpcGateway }}
  rpc Ping (PingRequest) returns (PingResponse) {
    option (google.api.http) = {
      get: "/v1/ping"
    };
  }
{{- else }}
  rpc Ping (PingRequest) returns (PingResponse);
{{- end }}
}

message PingRequest {}

message PingResponse {
  string message = 1;
}
