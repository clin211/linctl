syntax = "proto3";

package {{.AppName}}.v1;

option go_package = "{{.Module}}/pkg/api/{{.AppName}}/v1;v1";

// {{.Resource | Pascal}}Info represents a {{.Resource | Pascal}} entity.
message {{.Resource | Pascal}}Info {
  string {{.Resource | Snake}}_id = 1;
  int64  created_at = 2;
  int64  updated_at = 3;
}

// Create{{.Resource | Pascal}}Request is the request message for Create{{.Resource | Pascal}}.
message Create{{.Resource | Pascal}}Request {}

// Create{{.Resource | Pascal}}Response is the response message for Create{{.Resource | Pascal}}.
message Create{{.Resource | Pascal}}Response {
  string {{.Resource | Snake}}_id = 1;
}

// Update{{.Resource | Pascal}}Request is the request message for Update{{.Resource | Pascal}}.
message Update{{.Resource | Pascal}}Request {
  string {{.Resource | Snake}}_id = 1;
}

// Update{{.Resource | Pascal}}Response is the response message for Update{{.Resource | Pascal}}.
message Update{{.Resource | Pascal}}Response {}

// Delete{{.Resource | Pascal}}Request is the request message for Delete{{.Resource | Pascal}}.
message Delete{{.Resource | Pascal}}Request {
  string {{.Resource | Snake}}_id = 1;
}

// Delete{{.Resource | Pascal}}Response is the response message for Delete{{.Resource | Pascal}}.
message Delete{{.Resource | Pascal}}Response {}

// Get{{.Resource | Pascal}}Request is the request message for Get{{.Resource | Pascal}}.
message Get{{.Resource | Pascal}}Request {
  string {{.Resource | Snake}}_id = 1;
}

// Get{{.Resource | Pascal}}Response is the response message for Get{{.Resource | Pascal}}.
message Get{{.Resource | Pascal}}Response {
  {{.Resource | Pascal}}Info {{.Resource | Snake}} = 1;
}

// List{{.Resource | Pascal}}Request is the request message for List{{.Resource | Pascal}}.
message List{{.Resource | Pascal}}Request {
  int64 offset = 1;
  int64 limit  = 2;
}

// List{{.Resource | Pascal}}Response is the response message for List{{.Resource | Pascal}}.
message List{{.Resource | Pascal}}Response {
  int64              total  = 1;
  repeated {{.Resource | Pascal}}Info {{.Resource | Snake | Plural}} = 2;
}
