syntax = "proto3";

package {{.AppName}}.v1;

option go_package = "{{.Module}}/pkg/api/{{.AppName}}/v1;v1";

// {{.Resource | Pascal}}Info 表示一个 {{.Resource | Pascal}} 实体。
message {{.Resource | Pascal}}Info {
  string {{.Resource | Snake}}_id = 1;
  int64  created_at = 2;
  int64  updated_at = 3;
}

// Create{{.Resource | Pascal}}Request 是 Create{{.Resource | Pascal}} 的请求消息。
message Create{{.Resource | Pascal}}Request {}

// Create{{.Resource | Pascal}}Response 是 Create{{.Resource | Pascal}} 的响应消息。
message Create{{.Resource | Pascal}}Response {
  string {{.Resource | Snake}}_id = 1;
}

// Update{{.Resource | Pascal}}Request 是 Update{{.Resource | Pascal}} 的请求消息。
message Update{{.Resource | Pascal}}Request {
  string {{.Resource | Snake}}_id = 1;
}

// Update{{.Resource | Pascal}}Response 是 Update{{.Resource | Pascal}} 的响应消息。
message Update{{.Resource | Pascal}}Response {}

// Delete{{.Resource | Pascal}}Request 是 Delete{{.Resource | Pascal}} 的请求消息。
message Delete{{.Resource | Pascal}}Request {
  string {{.Resource | Snake}}_id = 1;
}

// Delete{{.Resource | Pascal}}Response 是 Delete{{.Resource | Pascal}} 的响应消息。
message Delete{{.Resource | Pascal}}Response {}

// Get{{.Resource | Pascal}}Request 是 Get{{.Resource | Pascal}} 的请求消息。
message Get{{.Resource | Pascal}}Request {
  string {{.Resource | Snake}}_id = 1;
}

// Get{{.Resource | Pascal}}Response 是 Get{{.Resource | Pascal}} 的响应消息。
message Get{{.Resource | Pascal}}Response {
  {{.Resource | Pascal}}Info {{.Resource | Snake}} = 1;
}

// List{{.Resource | Pascal}}Request 是 List{{.Resource | Pascal}} 的请求消息。
message List{{.Resource | Pascal}}Request {
  int64 offset = 1;
  int64 limit  = 2;
}

// List{{.Resource | Pascal}}Response 是 List{{.Resource | Pascal}} 的响应消息。
message List{{.Resource | Pascal}}Response {
  int64              total  = 1;
  repeated {{.Resource | Pascal}}Info {{.Resource | Snake | Plural}} = 2;
}
