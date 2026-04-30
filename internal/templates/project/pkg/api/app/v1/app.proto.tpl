syntax = "proto3";

package {{.AppName}}.v1;

option go_package = "{{.Module}}/pkg/api/{{.AppName}}/v1;v1";

import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";

option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
  info: {
    title: "{{.AppName | Title}} API";
    version: "v1";
    contact: {
      name: "{{.Author}}";
      email: "{{.Email}}";
    };
  };
  schemes: HTTP;
  schemes: HTTPS;
  consumes: "application/json";
  produces: "application/json";
};

// {{.AppName | Title | Plural}}Service defines the {{.AppName | Title}} API.
service {{.AppName | Pascal}}Service {
}

// HealthzRequest is the request message for the Healthz RPC.
message HealthzRequest {}

// HealthzResponse is the response message for the Healthz RPC.
message HealthzResponse {
  // status is the service health status.
  string status = 1;
  // timestamp is the current server time.
  string timestamp = 2;
}
