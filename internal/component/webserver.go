package component

import (
	"github.com/clin211/lin/internal/ast"
	"github.com/clin211/lin/internal/codegen"
	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/project"
)

// WebServerKind 是 WebServer 组件的注册 Kind。
const WebServerKind = "WebServer"

// WebServer 是 linctl 内置的 Web 服务组件实现。
//
// 当前范围（详见 docs/11-implementation-plan.md）：
//   - framework=gin（Phase 1）/ framework=grpc（Phase 3 Story 3.1 第一波）
//   - storage in [memory, gorm-postgres]（其他 storage 在 Phase 3 后续）
//   - 不在范围内的组合返回 ErrNotImplementedYet
type WebServer struct {
	spec project.Component
}

// NewWebServer 通过 project.Component 构造 WebServer 实例。
//
// 不做 Validate（由调用方在 Validate 阶段统一执行）。
func NewWebServer(spec project.Component) *WebServer {
	return &WebServer{spec: spec}
}

// WebServerFactory 是 Registry 用的 Factory（接受 map[string]any 形式）。
//
// 但实际 linctl 在 orchestrator 层会优先用强类型 NewWebServer。这里保留 Factory
// 形式以满足 Registry 接口的统一性。
func WebServerFactory(_ map[string]any) (Component, error) {
	return &WebServer{}, nil
}

// Kind 返回 "WebServer"。
func (w *WebServer) Kind() string { return WebServerKind }

// Name 返回组件实例名。
func (w *WebServer) Name() string { return w.spec.Name }

// Validate 校验配置合法性。Phase 1 范围检查在此完成。
func (w *WebServer) Validate(_ *project.Project) error {
	if w.spec.Name == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"WebServer.Name is required",
			"Set components[].name to a non-empty kebab-case identifier")
	}
	if w.spec.Framework == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"WebServer.Framework is required",
			"Set components[].framework=gin or grpc")
	}
	switch w.spec.Framework {
	case "gin", "grpc":
		// supported
	default:
		return linctlerr.Newf(linctlerr.ErrNotImplementedYet,
			"WebServer.Framework=%q not supported", w.spec.Framework).
			WithHint("Allowed: gin / grpc.")
	}
	if w.spec.GrpcGateway && w.spec.Framework != "grpc" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"WebServer.GrpcGateway is only valid when framework=grpc",
			"Remove grpcGateway or set components[].framework=grpc")
	}
	if w.spec.Storage == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"WebServer.Storage is required",
			"Set components[].storage to memory or gorm-postgres (Phase 1)")
	}
	switch w.spec.Storage {
	case "gorm-mysql", "gorm-postgres", "mongo":
		// supported by miniblog-v4 aligned templates
	default:
		return linctlerr.Newf(linctlerr.ErrNotImplementedYet,
			"WebServer.Storage=%q not supported", w.spec.Storage).
			WithHint("Allowed: gorm-mysql / gorm-postgres / mongo (memory / sqlite / redis are not supported yet).")
	}
	return nil
}

// BasePairs 返回 WebServer 自身的骨架文件 Pair 列表。
//
// 模板路径约定：templates/component/webserver/<file>.tpl，渲染到
// 用户项目内的对应路径。framework=grpc 时部分文件改用 grpc 子目录中的模板。
func (w *WebServer) BasePairs(_ *project.Project) []codegen.Pair {
	owner := "WebServer:" + w.spec.Name

	// `common` 是与 framework 无关的项目级文件（Makefile / go.mod / README / .gitignore）。
	//
	// 注意：biz/store 不放在这里——gin 模板的 biz/store 依赖 pkg/authz、pkg/store/where 等
	// web-gin 运行时基础设施，对 grpc 项目不适用。grpc 生成路径下的业务骨架由 grpc/handler.go.tpl 处理。
	//
	// third_party/protobuf/* 与 framework 无关（gin / grpc 都需要 protoc 时引用），
	// 因此放在这里被两条分支共用；多 WebServer 共存时由 PairBuilder 按 Dst 去重。
	common := []codegen.Pair{
		{
			Dst:        "Makefile",
			TemplateID: "templates/project/Makefile.tpl",
			Owner:      owner,
		},
		{
			Dst:        "go.mod",
			TemplateID: "templates/project/go.mod.tpl",
			Owner:      owner,
		},
		{
			Dst:        ".gitignore",
			TemplateID: "templates/project/gitignore.tpl",
			Owner:      owner,
		},
		{
			Dst:        "README.md",
			TemplateID: "templates/project/README.md.tpl",
			Owner:      owner,
		},
	}
	common = append(common, webGinThirdPartyPairs(owner)...)

	if w.spec.Framework == "grpc" {
		grpc := []codegen.Pair{
			{
				Dst:        "cmd/" + w.spec.Name + "/main.go",
				TemplateID: "templates/component/webserver/grpc/cmd_main.go.tpl",
				Owner:      owner,
			},
			{
				Dst:        "internal/" + w.spec.Name + "/server.go",
				TemplateID: "templates/component/webserver/grpc/server.go.tpl",
				Owner:      owner,
			},
			{
				Dst:        "internal/" + w.spec.Name + "/interceptor/interceptor.go",
				TemplateID: "templates/component/webserver/grpc/interceptor/interceptor.go.tpl",
				Owner:      owner,
			},
			{
				Dst:        "internal/" + w.spec.Name + "/handler/handler.go",
				TemplateID: "templates/component/webserver/grpc/handler.go.tpl",
				Owner:      owner,
			},
			{
				Dst:        "api/" + w.spec.Name + "/v1/api.proto",
				TemplateID: "templates/component/webserver/grpc/proto/api.proto.tpl",
				Owner:      owner,
			},
		}
		return append(grpc, common...)
	}

	gin := []codegen.Pair{
		// cmd/<app>/...
		{
			Dst:        "cmd/" + w.spec.Name + "/main.go",
			TemplateID: "templates/component/webserver/cmd_main.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "cmd/" + w.spec.Name + "/app/server.go",
			TemplateID: "templates/component/webserver/cmd/app/server.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "cmd/" + w.spec.Name + "/app/options/options.go",
			TemplateID: "templates/component/webserver/cmd/app/options/options.go.tpl",
			Owner:      owner,
		},

		// internal/<app>/...
		{
			Dst:        "internal/" + w.spec.Name + "/server.go",
			TemplateID: "templates/component/webserver/server.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/httpserver.go",
			TemplateID: "templates/component/webserver/httpserver.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/wire.go",
			TemplateID: "templates/component/webserver/wire.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/biz/biz.go",
			TemplateID: "templates/component/webserver/biz.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/store/store.go",
			TemplateID: "templates/component/webserver/store.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/handler/handler.go",
			TemplateID: "templates/component/webserver/handler/handler.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/handler/healthz.go",
			TemplateID: "templates/component/webserver/handler/healthz.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/pkg/validation/validation.go",
			TemplateID: "templates/component/webserver/internal/pkg/validation/validation.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/pkg/metrics/metrics.go",
			TemplateID: "templates/component/webserver/internal/pkg/metrics/metrics.go.tpl",
			Owner:      owner,
		},

		// configs/...
		{
			Dst:        "configs/" + w.spec.Name + ".yaml",
			TemplateID: "templates/component/webserver/configs/app.yaml.tpl",
			Owner:      owner,
		},
		{
			Dst:        "configs/casbin/model.conf",
			TemplateID: "templates/component/webserver/configs/casbin/model.conf",
			Owner:      owner,
		},
		{
			Dst:        "configs/casbin/policy.csv",
			TemplateID: "templates/component/webserver/configs/casbin/policy.csv",
			Owner:      owner,
		},
	}
	gin = append(gin, webGinPkgPairs(owner)...)
	gin = append(gin, webGinRuntimePairs(owner, w.spec.Storage)...)
	gin = append(gin, webGinUserResourcePairs(owner, w.spec.Name)...)
	gin = append(gin, webGinProtoPairs(owner, w.spec.Name)...)
	gin = append(gin, webGinExtraPkgPairs(owner)...)
	gin = append(gin, webGinEngineeringPairs(owner, w.spec.Name)...)
	gin = append(gin, webGinExtraConfigsPairs(owner, w.spec.Name)...)
	gin = append(gin, webGinExamplesAndDocsPairs(owner, w.spec.Name)...)
	return append(gin, common...)
}

// webGinPkgPairs 返回 web-gin 风格的「值类型 / 通用工具」骨架 Pair。
//
// 这些是不依赖于运行时框架（gin / gorm / cobra / viper）的纯类型工具：
// internal/pkg/{contextx,known,errno,rid} 与 pkg/{errorsx,id}。
//
// Dst 不带 component name，是项目级共享文件；多 WebServer 组件并存时
// 由 PairBuilder 按 Dst 去重，最终只生成一份。
func webGinPkgPairs(owner string) []codegen.Pair {
	return []codegen.Pair{
		{Dst: "internal/pkg/contextx/contextx.go", TemplateID: "templates/web-gin/internal/pkg/contextx/contextx.go.tpl", Owner: owner},
		{Dst: "internal/pkg/contextx/doc.go", TemplateID: "templates/web-gin/internal/pkg/contextx/doc.go.tpl", Owner: owner},
		{Dst: "internal/pkg/known/known.go", TemplateID: "templates/web-gin/internal/pkg/known/known.go.tpl", Owner: owner},
		{Dst: "internal/pkg/known/role.go", TemplateID: "templates/web-gin/internal/pkg/known/role.go.tpl", Owner: owner},
		{Dst: "internal/pkg/known/doc.go", TemplateID: "templates/web-gin/internal/pkg/known/doc.go.tpl", Owner: owner},
		{Dst: "pkg/errorsx/errorsx.go", TemplateID: "templates/web-gin/pkg/errorsx/errorsx.go.tpl", Owner: owner},
		{Dst: "pkg/errorsx/code.go", TemplateID: "templates/web-gin/pkg/errorsx/code.go.tpl", Owner: owner},
		{Dst: "pkg/errorsx/doc.go", TemplateID: "templates/web-gin/pkg/errorsx/doc.go.tpl", Owner: owner},
		{Dst: "internal/pkg/errno/code.go", TemplateID: "templates/web-gin/internal/pkg/errno/code.go.tpl", Owner: owner},
		{Dst: "internal/pkg/errno/user.go", TemplateID: "templates/web-gin/internal/pkg/errno/user.go.tpl", Owner: owner},
		{Dst: "internal/pkg/errno/doc.go", TemplateID: "templates/web-gin/internal/pkg/errno/doc.go.tpl", Owner: owner},
		{Dst: "pkg/id/sonyflake.go", TemplateID: "templates/web-gin/pkg/id/sonyflake.go.tpl", Owner: owner},
		{Dst: "pkg/id/options.go", TemplateID: "templates/web-gin/pkg/id/options.go.tpl", Owner: owner},
		{Dst: "pkg/id/code.go", TemplateID: "templates/web-gin/pkg/id/code.go.tpl", Owner: owner},
		{Dst: "pkg/id/doc.go", TemplateID: "templates/web-gin/pkg/id/doc.go.tpl", Owner: owner},
		{Dst: "internal/pkg/rid/rid.go", TemplateID: "templates/web-gin/internal/pkg/rid/rid.go.tpl", Owner: owner},
		{Dst: "internal/pkg/rid/salt.go", TemplateID: "templates/web-gin/internal/pkg/rid/salt.go.tpl", Owner: owner},
		{Dst: "internal/pkg/rid/doc.go", TemplateID: "templates/web-gin/internal/pkg/rid/doc.go.tpl", Owner: owner},
	}
}

// webGinRuntimePairs 返回 web-gin 风格的「运行时基础设施」骨架 Pair。
//
// 包含：
//   - pkg/core, pkg/db, pkg/server, pkg/options, pkg/store/*, pkg/authz, pkg/token,
//     pkg/middleware/gin, pkg/binding, pkg/version, pkg/otel/exporter/empty
//   - internal/pkg/middleware/gin/{header,context,authn,authz,requestid}
//
// 这些都依赖于 Gin / GORM / Mongo / Casbin / OTel / JWT 等运行时库；多 WebServer
// 组件时由 PairBuilder 按 Dst 去重。仅在 framework=gin 时由 BasePairs 调用。
//
// 部分文件按 storage 条件化生成：
//   - storage=gorm-mysql    → pkg/db/mysql.go    + pkg/options/mysql_options.go
//   - storage=gorm-postgres → pkg/db/postgresql.go + pkg/options/postgresql_options.go
//   - storage=mongo         → pkg/options/mongo_options.go（mongo 不需要单独 db driver 文件）
//
// 这样可以避免生成项目 import 不需要的数据库驱动（与 go.mod.tpl 的 storage 条件化一致）。
func webGinRuntimePairs(owner, storage string) []codegen.Pair {
	pairs := []codegen.Pair{
		// pkg/core
		{Dst: "pkg/core/config.go", TemplateID: "templates/web-gin/pkg/core/config.go.tpl", Owner: owner},
		{Dst: "pkg/core/core.go", TemplateID: "templates/web-gin/pkg/core/core.go.tpl", Owner: owner},

		// pkg/server
		{Dst: "pkg/server/server.go", TemplateID: "templates/web-gin/pkg/server/server.go.tpl", Owner: owner},
		{Dst: "pkg/server/http_server.go", TemplateID: "templates/web-gin/pkg/server/http_server.go.tpl", Owner: owner},

		// pkg/options 公共部分（与 storage 无关）
		{Dst: "pkg/options/options.go", TemplateID: "templates/web-gin/pkg/options/options.go.tpl", Owner: owner},
		{Dst: "pkg/options/helper.go", TemplateID: "templates/web-gin/pkg/options/helper.go.tpl", Owner: owner},
		{Dst: "pkg/options/http_options.go", TemplateID: "templates/web-gin/pkg/options/http_options.go.tpl", Owner: owner},
		{Dst: "pkg/options/tls_options.go", TemplateID: "templates/web-gin/pkg/options/tls_options.go.tpl", Owner: owner},
		{Dst: "pkg/options/otel_options.go", TemplateID: "templates/web-gin/pkg/options/otel_options.go.tpl", Owner: owner},
		{Dst: "pkg/options/slog_options.go", TemplateID: "templates/web-gin/pkg/options/slog_options.go.tpl", Owner: owner},
	}

	// storage 条件化：只生成实际使用的 db driver + options
	switch storage {
	case "gorm-mysql":
		pairs = append(pairs,
			codegen.Pair{Dst: "pkg/db/mysql.go", TemplateID: "templates/web-gin/pkg/db/mysql.go.tpl", Owner: owner},
			codegen.Pair{Dst: "pkg/options/mysql_options.go", TemplateID: "templates/web-gin/pkg/options/mysql_options.go.tpl", Owner: owner},
		)
	case "gorm-postgres":
		pairs = append(pairs,
			codegen.Pair{Dst: "pkg/db/postgresql.go", TemplateID: "templates/web-gin/pkg/db/postgresql.go.tpl", Owner: owner},
			codegen.Pair{Dst: "pkg/options/postgresql_options.go", TemplateID: "templates/web-gin/pkg/options/postgresql_options.go.tpl", Owner: owner},
		)
	case "mongo":
		pairs = append(pairs,
			codegen.Pair{Dst: "pkg/options/mongo_options.go", TemplateID: "templates/web-gin/pkg/options/mongo_options.go.tpl", Owner: owner},
		)
	}

	pairs = append(pairs, []codegen.Pair{

		// pkg/otelslog (vendored slog -> OTel bridge with web-gin-specific API)
		{Dst: "pkg/otelslog/handler.go", TemplateID: "templates/web-gin/pkg/otelslog/handler.go.tpl", Owner: owner},
		{Dst: "pkg/otelslog/convert.go", TemplateID: "templates/web-gin/pkg/otelslog/convert.go.tpl", Owner: owner},
		{Dst: "pkg/otelslog/gen.go", TemplateID: "templates/web-gin/pkg/otelslog/gen.go.tpl", Owner: owner},

		// pkg/store
		{Dst: "pkg/store/store.go", TemplateID: "templates/web-gin/pkg/store/store.go.tpl", Owner: owner},
		{Dst: "pkg/store/logger.go", TemplateID: "templates/web-gin/pkg/store/logger.go.tpl", Owner: owner},
		{Dst: "pkg/store/logger/empty/logger.go", TemplateID: "templates/web-gin/pkg/store/logger/empty/logger.go.tpl", Owner: owner},
		{Dst: "pkg/store/registry/registry.go", TemplateID: "templates/web-gin/pkg/store/registry/registry.go.tpl", Owner: owner},
		{Dst: "pkg/store/where/where.go", TemplateID: "templates/web-gin/pkg/store/where/where.go.tpl", Owner: owner},

		// pkg/authz
		{Dst: "pkg/authz/authz.go", TemplateID: "templates/web-gin/pkg/authz/authz.go.tpl", Owner: owner},

		// pkg/token
		{Dst: "pkg/token/token.go", TemplateID: "templates/web-gin/pkg/token/token.go.tpl", Owner: owner},

		// pkg/middleware/gin
		{Dst: "pkg/middleware/gin/observability.go", TemplateID: "templates/web-gin/pkg/middleware/gin/observability.go.tpl", Owner: owner},

		// pkg/binding
		{Dst: "pkg/binding/binding.go", TemplateID: "templates/web-gin/pkg/binding/binding.go.tpl", Owner: owner},

		// pkg/version
		{Dst: "pkg/version/version.go", TemplateID: "templates/web-gin/pkg/version/version.go.tpl", Owner: owner},
		{Dst: "pkg/version/flag.go", TemplateID: "templates/web-gin/pkg/version/flag.go.tpl", Owner: owner},

		// pkg/otel/exporter/empty
		{Dst: "pkg/otel/exporter/empty/empty.go", TemplateID: "templates/web-gin/pkg/otel/exporter/empty/empty.go.tpl", Owner: owner},

		// internal/pkg/middleware/gin
		{Dst: "internal/pkg/middleware/gin/header.go", TemplateID: "templates/web-gin/internal/pkg/middleware/gin/header.go.tpl", Owner: owner},
		{Dst: "internal/pkg/middleware/gin/context.go", TemplateID: "templates/web-gin/internal/pkg/middleware/gin/context.go.tpl", Owner: owner},
		{Dst: "internal/pkg/middleware/gin/authn.go", TemplateID: "templates/web-gin/internal/pkg/middleware/gin/authn.go.tpl", Owner: owner},
		{Dst: "internal/pkg/middleware/gin/authz.go", TemplateID: "templates/web-gin/internal/pkg/middleware/gin/authz.go.tpl", Owner: owner},
		{Dst: "internal/pkg/middleware/gin/requestid.go", TemplateID: "templates/web-gin/internal/pkg/middleware/gin/requestid.go.tpl", Owner: owner},
	}...)

	return pairs
}

// webGinUserResourcePairs 返回 web-gin 的「user 资源完整链路」骨架 Pair。
//
// 覆盖 biz / store / handler / model / pkg/conversion / pkg/validation 五层，
// 用于 framework=gin 时一次性把 miniblog-v4 的 user 资源完整代码路径生成出来。
// 注意 user.go 的 validation 与 BasePairs 中的 validation.go（公共校验框架）不冲突。
func webGinUserResourcePairs(owner, name string) []codegen.Pair {
	base := "internal/" + name
	return []codegen.Pair{
		{Dst: base + "/biz/v1/user/user.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/user.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/doc.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/doc.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/create.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/create.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/get.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/get.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/list.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/list.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/listwithbadperformance.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/listwithbadperformance.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/update.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/update.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/delete.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/delete.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/changepassword.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/changepassword.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/login.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/login.go.tpl", Owner: owner},
		{Dst: base + "/biz/v1/user/refreshtoken.go", TemplateID: "templates/component/webserver/internal/biz/v1/user/refreshtoken.go.tpl", Owner: owner},
		{Dst: base + "/store/user.go", TemplateID: "templates/component/webserver/internal/store/user.go.tpl", Owner: owner},
		{Dst: base + "/handler/user.go", TemplateID: "templates/component/webserver/internal/handler/user.go.tpl", Owner: owner},
		{Dst: base + "/model/user.gen.go", TemplateID: "templates/component/webserver/internal/model/user.gen.go.tpl", Owner: owner},
		{Dst: base + "/model/hook_user.go", TemplateID: "templates/component/webserver/internal/model/hook_user.go.tpl", Owner: owner},
		{Dst: base + "/pkg/conversion/user.go", TemplateID: "templates/component/webserver/internal/pkg/conversion/user.go.tpl", Owner: owner},
		{Dst: base + "/pkg/validation/user.go", TemplateID: "templates/component/webserver/internal/pkg/validation/user.go.tpl", Owner: owner},
	}
}

// webGinProtoPairs 返回 web-gin 的 proto 文件 Pair。
//
// 输出到 pkg/api/<name>/v1/，包含 healthz / user / <name>.proto 三份；
// 用户 `make protoc` 后会在同目录生成 *.pb.go / *.pb.gw.go。
func webGinProtoPairs(owner, name string) []codegen.Pair {
	base := "pkg/api/" + name + "/v1"
	return []codegen.Pair{
		{Dst: base + "/healthz.proto", TemplateID: "templates/component/webserver/proto/healthz.proto.tpl", Owner: owner},
		{Dst: base + "/user.proto", TemplateID: "templates/component/webserver/proto/user.proto.tpl", Owner: owner},
		{Dst: base + "/" + name + ".proto", TemplateID: "templates/component/webserver/proto/apiserver.proto.tpl", Owner: owner},
	}
}

// webGinThirdPartyPairs 返回 third_party/protobuf/ 下的 Google API + 扩展 proto。
//
// 这些 proto 是 protoc / grpc-gateway / openapiv2 等工具的依赖；与 framework
// 无关（gin 与 grpc 都需要），由 BasePairs 放进 common 列表共用。所有文件均
// 是无 .tpl 后缀的「逐字 pass-through」副本，不含模板指令。
func webGinThirdPartyPairs(owner string) []codegen.Pair {
	mk := func(rel string) codegen.Pair {
		return codegen.Pair{
			Dst:        "third_party/protobuf/" + rel,
			TemplateID: "templates/component/webserver/third_party/protobuf/" + rel,
			Owner:      owner,
		}
	}
	return []codegen.Pair{
		mk("github.com/gogo/protobuf/gogoproto/gogo.proto"),
		mk("github.com/onexstack/defaults/defaults.proto"),
		mk("google/api/annotations.proto"),
		mk("google/api/client.proto"),
		mk("google/api/field_behavior.proto"),
		mk("google/api/http.proto"),
		mk("google/api/httpbody.proto"),
		mk("google/protobuf/any.proto"),
		mk("google/protobuf/api.proto"),
		mk("google/protobuf/compiler/plugin.proto"),
		mk("google/protobuf/descriptor.proto"),
		mk("google/protobuf/duration.proto"),
		mk("google/protobuf/empty.proto"),
		mk("google/protobuf/field_mask.proto"),
		mk("google/protobuf/source_context.proto"),
		mk("google/protobuf/struct.proto"),
		mk("google/protobuf/timestamp.proto"),
		mk("google/protobuf/type.proto"),
		mk("google/protobuf/wrappers.proto"),
		mk("protoc-gen-openapiv2/options/annotations.proto"),
		mk("protoc-gen-openapiv2/options/openapiv2.proto"),
	}
}

// webGinExtraPkgPairs 返回 web-gin 在项目级 pkg/ 下的扩展工具包 Pair。
//
// 与 webGinPkgPairs / webGinRuntimePairs 互不重叠；这里覆盖 miniblog-v4
// 的：app（cobra/viper 应用框架）、authn/jwt（鉴权）、i18n、log/logger（多套
// 日志桥接）、ptr、validation、util/{strings,ip,file,gen,pagination,version,
// reflect,retry,controller,lint}、db/redis、options/{redis,kafka,health}。
//
// 这些都是项目级共享文件（Dst 不带 component name），多 WebServer 共存时由
// PairBuilder 按 Dst 去重。
func webGinExtraPkgPairs(owner string) []codegen.Pair {
	mk := func(rel string) codegen.Pair {
		return codegen.Pair{
			Dst:        "pkg/" + rel,
			TemplateID: "templates/web-gin/pkg/" + rel + ".tpl",
			Owner:      owner,
		}
	}
	return []codegen.Pair{
		// pkg/app — cobra+viper application framework
		mk("app/doc.go"),
		mk("app/app.go"),
		mk("app/config.go"),
		mk("app/help.go"),
		mk("app/options.go"),

		// pkg/authn — authentication facade
		mk("authn/doc.go"),
		mk("authn/authn.go"),

		// pkg/authn/jwt — JWT 实现
		mk("authn/jwt/doc.go"),
		mk("authn/jwt/jwt.go"),
		mk("authn/jwt/store.go"),
		mk("authn/jwt/token.go"),

		// pkg/authn/jwt/store/redis — JWT 黑名单 Redis 后端
		mk("authn/jwt/store/redis/doc.go"),
		mk("authn/jwt/store/redis/redis.go"),

		// pkg/db — Redis 客户端工厂
		mk("db/redis.go"),

		// pkg/i18n — 国际化
		mk("i18n/doc.go"),
		mk("i18n/i18n.go"),
		mk("i18n/context.go"),
		mk("i18n/options.go"),
		mk("i18n/i18n_test.go"),

		// pkg/log — 主日志包（zap-based）
		mk("log/doc.go"),
		mk("log/log.go"),
		mk("log/options.go"),
		mk("log/gorm.go"),
		mk("log/logger/store/logger.go"),

		// pkg/logger — 多套 logger 桥接（slog / klog / onex）
		mk("logger/logger.go"),
		mk("logger/empty/empty.go"),
		mk("logger/onex/onex.go"),
		mk("logger/klog/store/logger.go"),
		mk("logger/slog/breeze/breeze.go"),
		mk("logger/slog/gorm/gorm.go"),
		mk("logger/slog/resty/resty.go"),
		mk("logger/slog/store/logger.go"),
		mk("logger/slog/watch/watch.go"),

		// pkg/options — 额外 options 实现（redis / kafka / health）
		mk("options/redis_options.go"),
		mk("options/kafka_options.go"),
		mk("options/health_options.go"),

		// pkg/ptr — 指针辅助
		mk("ptr/doc.go"),
		mk("ptr/ptr.go"),
		mk("ptr/ptr_test.go"),

		// pkg/util/strings — 字符串工具
		mk("util/strings/doc.go"),
		mk("util/strings/strings.go"),
		mk("util/strings/base64.go"),
		mk("util/strings/strings_test.go"),

		// pkg/util/ip — IP 工具
		mk("util/ip/doc.go"),
		mk("util/ip/ip.go"),

		// pkg/util/file — 文件工具
		mk("util/file/doc.go"),
		mk("util/file/file.go"),

		// pkg/util/gen — 代码生成辅助
		mk("util/gen/doc.go"),
		mk("util/gen/gen.go"),
		mk("util/gen/gen_test.go"),

		// pkg/util/pagination — 分页参数
		mk("util/pagination/doc.go"),
		mk("util/pagination/pagination.go"),

		// pkg/util/version — version 工具
		mk("util/version/doc.go"),
		mk("util/version/version.go"),
		mk("util/version/version_test.go"),

		// pkg/util/reflect — reflect 工具
		mk("util/reflect/doc.go"),
		mk("util/reflect/reflect.go"),
		mk("util/reflect/reflect_test.go"),

		// pkg/util/retry — 重试
		mk("util/retry/doc.go"),
		mk("util/retry/retry.go"),

		// pkg/util/controller — controller-runtime 辅助
		mk("util/controller/controller.go"),

		// pkg/util/lint — 自定义 lint analyzer 集合
		mk("util/lint/doc.go"),
		mk("util/lint/lint.go"),
		mk("util/lint/errcheck/doc.go"),
		mk("util/lint/errcheck/errcheck.go"),
		mk("util/lint/errcheck/errcheck_no_bazel.go"),
		mk("util/lint/hash/doc.go"),
		mk("util/lint/hash/hash.go"),
		mk("util/lint/hash/testdata/src/a/doc.go"),
		mk("util/lint/hash/testdata/src/a/a.go"),
		mk("util/lint/pass/doc.go"),
		mk("util/lint/pass/pass.go"),
		mk("util/lint/shadow/doc.go"),
		mk("util/lint/shadow/shadow.go"),
		mk("util/lint/shadow/shadow_no_bazel.go"),

		// pkg/validation — validation 工厂
		mk("validation/validation.go"),
		mk("validation/validator.go"),
	}
}

// webGinEngineeringPairs 返回工程化骨架 Pair：Docker、CI、lint 配置、scripts。
//
// 对应 Stream D 的产出：根目录的 Dockerfile / docker-compose.env.yml /
// .golangci.yaml / .dockerignore / PROJECT / otel-collector.yaml；
// build/docker/<name>/{Dockerfile,docker-compose{,.prod}.yml}；
// scripts/{coverage.awk,boilerplate.txt,startup-test.sh}；
// .github/workflows/deploy.yml。
//
// 注：曾随项目附带的 scripts/sync_onexstack_pkg.sh（用于从上游 onexstack
// 拉取 pkg/ 并改写 import）已被废弃。templates/web-gin/pkg/ 已经把
// onexstack/pkg 的内容定制并模板化，由 webGinExtraPkgPairs 在 codegen
// 阶段一次性生成；运行时再跑同步脚本反而会引入版本不一致。
func webGinEngineeringPairs(owner, name string) []codegen.Pair {
	dockerDir := "build/docker/" + name
	return []codegen.Pair{
		{Dst: "Dockerfile", TemplateID: "templates/project/Dockerfile.tpl", Owner: owner},
		{Dst: "docker-compose.env.yml", TemplateID: "templates/project/docker-compose.env.yml.tpl", Owner: owner},
		{Dst: ".golangci.yaml", TemplateID: "templates/project/golangci.yaml.tpl", Owner: owner},
		{Dst: ".dockerignore", TemplateID: "templates/project/dockerignore.tpl", Owner: owner},
		{Dst: "PROJECT", TemplateID: "templates/project/PROJECT.tpl", Owner: owner},
		{Dst: "otel-collector.yaml", TemplateID: "templates/project/otel-collector.yaml.tpl", Owner: owner},
		{Dst: dockerDir + "/Dockerfile", TemplateID: "templates/component/webserver/build/docker/Dockerfile.tpl", Owner: owner},
		{Dst: dockerDir + "/docker-compose.yml", TemplateID: "templates/component/webserver/build/docker/docker-compose.yml.tpl", Owner: owner},
		{Dst: dockerDir + "/docker-compose.prod.yml", TemplateID: "templates/component/webserver/build/docker/docker-compose.prod.yml.tpl", Owner: owner},
		{Dst: "scripts/coverage.awk", TemplateID: "templates/project/scripts/coverage.awk", Owner: owner},
		{Dst: "scripts/boilerplate.txt", TemplateID: "templates/project/scripts/boilerplate.txt.tpl", Owner: owner},
		{Dst: "scripts/startup-test.sh", TemplateID: "templates/project/scripts/startup-test.sh.tpl", Owner: owner},
		{Dst: ".github/workflows/deploy.yml", TemplateID: "templates/project/github/workflows/deploy.yml.tpl", Owner: owner},
	}
}

// webGinExtraConfigsPairs 返回 configs/ 下的 Docker / SQL bootstrap 三件套。
//
// 注意：基础 configs/<name>.yaml 已由 BasePairs 提供，这里只补 docker 变体与
// 数据库初始化脚本（init_database.sql + basic.sql）。
func webGinExtraConfigsPairs(owner, name string) []codegen.Pair {
	return []codegen.Pair{
		{Dst: "configs/" + name + ".docker.yaml", TemplateID: "templates/component/webserver/configs/app.docker.yaml.tpl", Owner: owner},
		{Dst: "configs/init_database.sql", TemplateID: "templates/component/webserver/configs/init_database.sql.tpl", Owner: owner},
		{Dst: "configs/basic.sql", TemplateID: "templates/component/webserver/configs/basic.sql.tpl", Owner: owner},
	}
}

// webGinExamplesAndDocsPairs 返回 examples/gin-i18n、docs/、cmd/gen-gorm-model
// 三组 Pair。
//
// examples/gin-i18n 是一个独立的可运行 Go module（自带 go.mod），用于演示
// 跨层 i18n 用法；docs/ 提供错误码迁移、announcements、安装/部署等指南；
// cmd/gen-gorm-model 是开发期工具，用 gorm/gen 从 DB schema 生成 model.gen.go。
//
// `name` 参数预留用于未来按 component 名定制 examples 路径，当前未使用。
func webGinExamplesAndDocsPairs(owner, name string) []codegen.Pair {
	_ = name

	example := func(rel string) codegen.Pair {
		return codegen.Pair{
			Dst:        "examples/gin-i18n/" + rel,
			TemplateID: "templates/component/webserver/examples/gin-i18n/" + rel + ".tpl",
			Owner:      owner,
		}
	}
	doc := func(rel string) codegen.Pair {
		return codegen.Pair{
			Dst:        "docs/" + rel,
			TemplateID: "templates/component/webserver/docs/" + rel + ".tpl",
			Owner:      owner,
		}
	}
	// .keep 占位文件无 .tpl 后缀（0 字节），rendered 仍是 0 字节。
	keep := func(rel string) codegen.Pair {
		return codegen.Pair{
			Dst:        "docs/" + rel,
			TemplateID: "templates/component/webserver/docs/" + rel,
			Owner:      owner,
		}
	}

	return []codegen.Pair{
		// examples/gin-i18n/ — 13 条
		example("go.mod"),
		example("main.go"),
		example("README.md"),
		example("COOKIE_TESTING.md"),
		example("CROSS_LAYER_I18N.md"),
		example("handlers/example.go"),
		example("handlers/user.go"),
		example("service/user.go"),
		example("repository/user.go"),
		example("middleware/i18n.go"),
		example("locales/en.json"),
		example("locales/zh.json"),
		example("locales/ja.json"),

		// docs/ — 17 条（3 顶级 + 4 .keep + 10 guide/zh-CN）
		doc("error-examples.md"),
		doc("error-migration.md"),
		doc("fix-notes.md"),
		keep("devel/zh-CN/.keep"),
		keep("devel/en-US/.keep"),
		keep("images/.keep"),
		keep("guide/en-US/.keep"),
		doc("guide/zh-CN/README.md"),
		doc("guide/zh-CN/announcements.md"),
		doc("guide/zh-CN/otel-collector.md"),
		doc("guide/zh-CN/faq/README.md"),
		doc("guide/zh-CN/introduction/README.md"),
		doc("guide/zh-CN/installation/README.md"),
		doc("guide/zh-CN/quickstart/README.md"),
		doc("guide/zh-CN/operation-guide/README.md"),
		doc("guide/zh-CN/best-practice/README.md"),
		doc("guide/zh-CN/quickstart/DOCKER_DEPLOYMENT.md"),

		// cmd/gen-gorm-model/ — 1 条
		{Dst: "cmd/gen-gorm-model/gen_gorm_model.go", TemplateID: "templates/component/webserver/cmd/gen-gorm-model/gen_gorm_model.go.tpl", Owner: owner},
	}
}

// BaseMutators 返回组件 AST 修改。Phase 1 不做 AST 注入。
func (w *WebServer) BaseMutators(_ *project.Project) []ast.ASTMutator {
	return nil
}

// PostProcess 不需要副作用。Phase 1 返回 nil。
func (w *WebServer) PostProcess(_ *project.Project, _ FileSystem) error {
	return nil
}

// SpecComponent 暴露原始 YAML struct（仅供 orchestrator 在调度时用）。
func (w *WebServer) SpecComponent() project.Component {
	return w.spec
}
