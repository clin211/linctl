syntax = "proto3";

package {{ .Component.Name | snake }}.v1;

option go_package = "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1;v1";

// 提供用于定义 HTTP 映射的功能，比如通过 option (google.api.http) 实现 gRPC 到 HTTP 的映射
import "google/api/annotations.proto";
// 提供了一个标准的空消息类型 google.protobuf.Empty，适用于 RPC 方法不需要输入消息或输出消息的场景
import "google/protobuf/empty.proto";
// 为生成 OpenAPI 文档提供相关注释（如标题、版本、作者、许可证等信息）
import "protoc-gen-openapiv2/options/annotations.proto";
import "{{ .Component.Name }}/v1/healthz.proto";
import "{{ .Component.Name }}/v1/user.proto";

option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
    info: {
        // API 名称
        title: "{{ .Component.Name | pascal }} Service API v1";
        // API 版本
        version: "1.0";
        // API 描述
        description: "{{ .Project.Metadata.Description }}";
        // 开发者的联系方式
        contact: {
            name: "{{ .Project.Metadata.Name }}";
            email: "{{ .Project.Metadata.Author.Email }}";
        };
    };
    // 同时支持开发环境的 HTTP 与生产环境的 HTTPS
    schemes: HTTP;
    schemes: HTTPS;
    // 定义服务的请求和响应格式
    consumes: "application/json";
    produces: "application/json";
};

// {{ .Component.Name | pascal }}Service 是 {{ .Component.Name }} 组件对外暴露的聚合服务定义。
service {{ .Component.Name | pascal }}Service {
    // 健康检查
    rpc Healthz(google.protobuf.Empty) returns (HealthzResponse) {
        option (google.api.http) = {
            get: "/healthz"
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary: "健康检查";
            description: "检查服务是否健康运行";
            tags: "服务治理";
        };
    }
    // 创建用户
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
        option (google.api.http) = {
            post: "/v1/users"
            body: "*"
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary: "创建用户";
            description: "创建一个新的用户";
            tags: "用户管理";
        };
    }
    // 获取用户
    rpc GetUser(GetUserRequest) returns (GetUserResponse) {
        option (google.api.http) = {
            get: "/v1/users/{userID}"
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary: "获取用户";
            description: "根据用户 ID 获取用户信息";
            tags: "用户管理";
        };
    }
    // 更新用户
    rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse) {
        option (google.api.http) = {
            put: "/v1/users/{userID}"
            body: "*"
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary: "更新用户";
            description: "根据用户 ID 更新用户信息";
            tags: "用户管理";
        };
    }
    // 删除用户
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse) {
        option (google.api.http) = {
            delete: "/v1/users/{userID}"
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary: "删除用户";
            description: "根据用户 ID 删除用户";
            tags: "用户管理";
        };
    }
    // 列表用户
    rpc ListUsers(ListUserRequest) returns (ListUserResponse) {
        option (google.api.http) = {
            get: "/v1/users"
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary: "列表用户";
            description: "获取所有用户的列表";
            tags: "用户管理";
        };
    }
}
