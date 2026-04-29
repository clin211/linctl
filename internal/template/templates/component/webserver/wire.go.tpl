//go:build wireinject
// +build wireinject

package {{ .Component.Name }}

import (
	"github.com/google/wire"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/biz"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/pkg/validation"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/store"
	mw "{{ .Project.Metadata.Module }}/internal/pkg/middleware/gin"
	"{{ .Project.Metadata.Module }}/pkg/authz"
)

// NewServer 是 Wire 注入入口：把 *Config 装配成完整的 *Server。
//
// 修改本文件或任何 ProviderSet 之后，必须运行 `make wire`（或 `wire ./...`）
// 重新生成 wire_gen.go，否则 build 会找不到 NewServer 实现。
func NewServer(*Config) (*Server, error) {
	wire.Build(
		NewWebServer,
		wire.Struct(new(ServerConfig), "*"),
		wire.Struct(new(Server), "*"),
		wire.NewSet(store.ProviderSet, biz.ProviderSet),
{{- if eq .Component.Storage "mongo" }}
		ProvideMongo,
{{- else }}
		ProvideDB,
{{- end }}
		validation.ProviderSet,
		wire.NewSet(
			wire.Struct(new(UserRetriever), "*"),
			wire.Bind(new(mw.UserRetriever), new(*UserRetriever)),
		),
		authz.ProviderSet,
	)
	return nil, nil
}
