package template_test

import (
	"strings"
	"testing"

	"github.com/clin211/lin/internal/project"
	tpl "github.com/clin211/lin/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProtocStreamProject 构造一个用于 protoc/proto/Makefile 烟测的样例项目。
//
// 与 stream_a_smoke_test.go 配合使用：本子流（Round 2 protoc 子流）只需要确认模板
// 在 ApplyDefaults 之后渲染通过、关键 token（package / service / target）齐全。
func newProtocStreamProject() *project.Project {
	p := &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:        "acme-blog",
			Module:      "github.com/acme/acme-blog",
			Description: "Acme demo blog (linctl protoc stream).",
			Author: project.Author{
				Name:  "Acme Team",
				Email: "team@acme.io",
			},
		},
		Spec: project.Spec{
			Components: []project.Component{
				{
					Kind:      "WebServer",
					Name:      "apiserver",
					Framework: "gin",
					Storage:   "gorm-postgres",
					Port:      5556,
					Features:  []string{"healthz", "user"},
				},
			},
		},
	}
	project.ApplyDefaults(p)
	return p
}

// TestProtocStream_ProtoTemplatesRender 校验 webserver/proto/*.proto.tpl
// （healthz / user / apiserver）模板渲染通过且产出关键 proto token。
func TestProtocStream_ProtoTemplatesRender(t *testing.T) {
	eng, err := tpl.New()
	require.NoError(t, err)

	p := newProtocStreamProject()
	data := tpl.TemplateData{Project: p, Component: p.Spec.Components[0]}

	cases := []struct {
		path     string
		mustHave []string
	}{
		{
			path: "templates/component/webserver/proto/healthz.proto.tpl",
			mustHave: []string{
				"package apiserver.v1;",
				"github.com/acme/acme-blog/pkg/api/apiserver/v1;v1",
				"message HealthzResponse",
			},
		},
		{
			path: "templates/component/webserver/proto/user.proto.tpl",
			mustHave: []string{
				"package apiserver.v1;",
				"message User ",
				"message CreateUserRequest",
				"message ListUserResponse",
			},
		},
		{
			path: "templates/component/webserver/proto/apiserver.proto.tpl",
			mustHave: []string{
				"package apiserver.v1;",
				"service ApiserverService",
				`import "apiserver/v1/healthz.proto";`,
				`import "apiserver/v1/user.proto";`,
				"rpc Healthz(google.protobuf.Empty)",
				"rpc CreateUser(CreateUserRequest)",
				"/v1/users",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			out, err := eng.RenderToString(tc.path, data)
			require.NoError(t, err, "render %s", tc.path)
			for _, want := range tc.mustHave {
				assert.Contains(t, out, want, "%s missing %q", tc.path, want)
			}
		})
	}
}

// TestProtocStream_MakefileTemplate 校验 Makefile.tpl 完整对齐 miniblog-v4
// 的全部 protoc / build / image / lint / deps / cover 等 target。
func TestProtocStream_MakefileTemplate(t *testing.T) {
	eng, err := tpl.New()
	require.NoError(t, err)

	p := newProtocStreamProject()
	data := tpl.TemplateData{Project: p, Component: p.Spec.Components[0]}
	out, err := eng.RenderToString("templates/project/Makefile.tpl", data)
	require.NoError(t, err)

	want := []string{
		"PROJ_ROOT_DIR :=",
		"APIROOT=$(PROJ_ROOT_DIR)/pkg/api",
		"ROOT_PACKAGE=github.com/acme/acme-blog",
		"REGISTRY_PREFIX ?= acme-blog",
		"VERSION_PACKAGE=github.com/acme/acme-blog/pkg/version",
		"all: deps protoc tidy format generate build cover",
		".PHONY: protoc",
		".PHONY: protoc.%",
		"--go_out=paths=source_relative:$(APIROOT)",
		"--go-grpc_out=paths=source_relative:$(APIROOT)",
		"--grpc-gateway_out=allow_delete_body=true",
		"--openapiv2_out=$(PROJ_ROOT_DIR)/api/openapi",
		"--defaults_out=paths=source_relative:$(APIROOT)",
		"protoc-go-inject-tag",
		".PHONY: deps",
		"protoc-gen-go-grpc",
		"protoc-gen-grpc-gateway",
		"protoc-gen-openapiv2",
		"protoc-gen-defaults",
		"google/wire/cmd/wire",
		"onexstack/addlicense",
		".PHONY: wire",
		".PHONY: run",
		"APP ?= apiserver",
	}
	for _, w := range want {
		assert.Contains(t, out, w, "Makefile.tpl missing %q", w)
	}
}

// TestProtocStream_ProtolintTemplate 校验 .protolint.yaml.tpl 渲染后包含正确
// 的 ignore 路径（pkg/api/<component>/v1/<component>.proto），并保留所有
// miniblog 提供的 lint rule。
func TestProtocStream_ProtolintTemplate(t *testing.T) {
	eng, err := tpl.New()
	require.NoError(t, err)

	p := newProtocStreamProject()
	data := tpl.TemplateData{Project: p, Component: p.Spec.Components[0]}
	out, err := eng.RenderToString("templates/project/protolint.yaml.tpl", data)
	require.NoError(t, err)

	want := []string{
		"- pkg/api/apiserver/v1/apiserver.proto",
		"MAX_LINE_LENGTH",
		"max_chars: 120",
		"convention: lower_camel_case",
		"FILE_NAMES_LOWER_SNAKE_CASE",
	}
	for _, w := range want {
		assert.Contains(t, out, w, "protolint.yaml.tpl missing %q", w)
	}
}

// TestProtocStream_ProtoPackageSnake 校验组件名含连字符时 proto `package`
// 行被转为合法标识符（snake_case），而 go_package 的 import 路径保留 kebab-case。
func TestProtocStream_ProtoPackageSnake(t *testing.T) {
	eng, err := tpl.New()
	require.NoError(t, err)

	p := newProtocStreamProject()
	p.Spec.Components[0].Name = "mb-apiserver"
	data := tpl.TemplateData{Project: p, Component: p.Spec.Components[0]}

	out, err := eng.RenderToString(
		"templates/component/webserver/proto/healthz.proto.tpl",
		data,
	)
	require.NoError(t, err)
	if !strings.Contains(out, "package mb_apiserver.v1;") {
		t.Errorf("expected snake_case proto package, got:\n%s", out)
	}
	if !strings.Contains(out, "/pkg/api/mb-apiserver/v1") {
		t.Errorf("go_package should keep kebab-case import path, got:\n%s", out)
	}
}

// TestProtocStream_ThirdPartyProtosCopiedVerbatim 校验 third_party/protobuf
// 下的 .proto 文件在模板 FS 内可读，且不含模板控制符（{{ / }}）——确保
// applier 走 Render 路径时是 verbatim pass-through。
func TestProtocStream_ThirdPartyProtosCopiedVerbatim(t *testing.T) {
	eng, err := tpl.New()
	require.NoError(t, err)

	p := newProtocStreamProject()
	data := tpl.TemplateData{Project: p, Component: p.Spec.Components[0]}

	// 抽样 5 个代表性文件覆盖三个目录树
	samples := map[string][]string{
		"templates/component/webserver/third_party/protobuf/google/api/annotations.proto": {
			"package google.api;",
			`import "google/api/http.proto";`,
		},
		"templates/component/webserver/third_party/protobuf/google/protobuf/timestamp.proto": {
			"message Timestamp",
		},
		"templates/component/webserver/third_party/protobuf/protoc-gen-openapiv2/options/annotations.proto": {
			"package grpc.gateway.protoc_gen_openapiv2.options;",
		},
		"templates/component/webserver/third_party/protobuf/github.com/onexstack/defaults/defaults.proto": {
			"package defaults;",
			"message FieldDefaults",
		},
		"templates/component/webserver/third_party/protobuf/github.com/gogo/protobuf/gogoproto/gogo.proto": {
			"package gogoproto;",
		},
	}

	for path, mustHave := range samples {
		t.Run(path, func(t *testing.T) {
			out, err := eng.RenderToString(path, data)
			require.NoError(t, err, "render %s", path)
			for _, want := range mustHave {
				assert.Contains(t, out, want, "third_party verbatim copy missing %q in %s", want, path)
			}
			assert.NotContains(t, out, "{{", "third_party file %s leaked template directives", path)
			assert.NotContains(t, out, "}}", "third_party file %s leaked template directives", path)
		})
	}
}
