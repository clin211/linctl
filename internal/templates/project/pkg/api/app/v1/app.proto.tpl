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

// {{.AppName | Title | Plural}}Service 定义 {{.AppName | Title}} API。
service {{.AppName | Pascal}}Service {
}

// HealthzRequest 是 Healthz RPC 的请求消息。
message HealthzRequest {}

// HealthzResponse 是 Healthz RPC 的响应消息。
message HealthzResponse {
  // status 表示服务的健康状态。
  string status = 1;
  // timestamp 为当前服务器时间。
  string timestamp = 2;
}
