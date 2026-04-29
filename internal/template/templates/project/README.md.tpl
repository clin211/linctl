{{- $app := (index .Project.Spec.Components 0).Name -}}
{{- $framework := (index .Project.Spec.Components 0).Framework -}}
{{- $storage := (index .Project.Spec.Components 0).Storage -}}
# {{ .Project.Metadata.Name }}

{{ .Project.Metadata.Description | default "由 linctl 生成的 Go 微服务项目。" }}

本项目由 [linctl](https://github.com/clin211/lin) 脚手架生成，采用企业级分层架构。

{{ if eq $framework "gin" -}}
## 架构概览

- **HTTP 框架**：[Gin](https://github.com/gin-gonic/gin)，标准中间件链：Recovery / NoCache / CORS / Secure / OTel / Observability / Context。
- **存储**：`{{ $storage }}`{{ if eq $storage "gorm-mysql" }}（GORM + MySQL）{{ else if eq $storage "gorm-postgres" }}（GORM + PostgreSQL）{{ else if eq $storage "mongo" }}（MongoDB）{{ end }}。
- **认证**：JWT（`pkg/token`）。
- **鉴权**：Casbin RBAC{{ if eq $storage "mongo" }}，使用文件适配器读取 `configs/casbin/policy.csv`{{ else }}，使用 GORM 适配器（`pkg/authz`，见 `configs/casbin/`）{{ end }}。
- **可观测性**：OpenTelemetry traces / metrics / logs + Prometheus `/metrics`。
- **依赖注入**：Google Wire（执行 `make wire` 重新生成 `wire_gen.go`）。
- **配置**：Viper + Cobra；YAML 配置文件在 `configs/{{ $app }}.yaml`，环境变量前缀为 `{{ upper (snake $app) }}_`。

## 目录布局

```
.
├── cmd/{{ $app }}/                 # 进程入口（Cobra / Viper）
│   └── app/
│       ├── server.go         # NewWebServerCommand
│       └── options/          # ServerOptions（HTTP / TLS / DB / OTel / JWT）
├── internal/{{ $app }}/            # 私有应用代码
│   ├── server.go             # Config / Server / NewDB / UserRetriever
│   ├── httpserver.go         # Gin 引擎 + InstallRESTAPI + 中间件链
│   ├── wire.go               # Wire DI 指令（生成 wire_gen.go）
│   ├── biz/                  # 业务层（IBiz 接口）
│   ├── store/                # 数据访问层（IStore 接口）
│   ├── handler/              # HTTP handler（init+Register 自注册路由）
│   └── pkg/{validation,metrics}/
├── internal/pkg/             # 项目私有工具包
│   ├── contextx, known, errno, rid
│   └── middleware/gin/       # NoCache / Cors / Secure / Authn / Authz / RequestID / Context
├── pkg/                      # 可复用基础设施
│   ├── core, db, server, options
│   ├── store/{registry,where,logger}
│   ├── authz, token, binding, version
│   ├── middleware/gin        # Observability
│   ├── otel/exporter/empty
│   └── errorsx, id
└── configs/
    ├── {{ $app }}.yaml       # 应用配置
    └── casbin/{model.conf,policy.csv}
```
{{- end }}

## 构建与生成

```bash
# 1) 拉取所有依赖
go mod tidy

# 2) 生成 Wire DI 代码（创建 internal/{{ $app }}/wire_gen.go）。
#    在 `linctl new` 之后必须执行一次；之后每次修改 wire.go / ProviderSet 也要重新生成。
make wire

# 3) 构建
make build

# 4) 运行（默认读取 configs/{{ $app }}.yaml）
./_output/bin/{{ $app }} -c configs/{{ $app }}.yaml
```

## 新增 REST 资源

```bash
# 自动生成 biz_<resource>.go / store_<resource>.go / handler_<resource>.go
# 并通过 AST 注入将新方法添加到 IBiz / IStore 接口。
linctl add api Post

# 每次 `linctl add api` 之后都要重新执行 wire：
make wire
```

资源 handler 通过 `init() { Register(...) }` 自动注册路由，**无需**手动修改 `httpserver.go`。

## 重新生成

```bash
linctl apply
```

linctl 只会更新自己管理的文件（通过 hash 注释追踪）。你对非托管文件的手动修改会被保留。
