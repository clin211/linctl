// Package project 定义 linctl 的核心领域模型——`linctl.yaml`（用户输入）
// 和独立的 `PROJECT` 文件（工具维护的运行时状态镜像）所对应的 Go 类型。
//
// 包含：
//   - 顶层 Project（Spec 文件）/ ProjectState（State 文件）
//   - 完整的 metadata / spec / defaults / component / hook 等子类型
//   - schema 演进（Migrator 接口 + v1alpha1 → v1）
//   - 加载 / 应用默认值 / 校验 / 保存 等行为
//
// 设计原则（与 docs/04-config-schema.md §4.4 严格对齐，并采纳
// docs/META-fix-decisions-2026-04-25.md 中的 SSOT 决策）：
//
//   - 字段命名：`spec.defaults.protoVersion` 而非历史的 `apiVersion`（SSOT §1.18）
//   - linctl.yaml 仅含 apiVersion / kind / metadata / spec（SSOT §1.7）
//   - PROJECT 文件含 status（generatedAt / cliVersion / schemaMigrations / lastApplyHash）
//   - Hook 含 policy 字段（restricted / confirm / unrestricted；SSOT §5.3）
//   - 错误类型统一为 *linctlerr.LinctlError（SSOT §1.1 / §5.1）
//
// 严格 KnownFields(true) 模式：未识别字段会在 Loader 阶段直接报错。
package project

// APIVersion 是 linctl Schema 的版本号，**不要与 spec.defaults.protoVersion 混淆**：
// 后者是用户项目内 proto 文件的版本前缀（如 v1 / v1alpha1），与 linctl Schema 完全无关。
//
// SSOT §1.18 锁定的命名修正：旧版本曾把 spec.defaults.apiVersion 作 proto 版本含义，
// 现已统一重命名为 spec.defaults.protoVersion。
type APIVersion = string

// 已发布的 linctl Schema 版本常量。
const (
	// APIVersionV1Alpha1 是早期版本，仅通过 `linctl upgrade` 自动迁移到 v1。
	APIVersionV1Alpha1 APIVersion = "linctl.dev/v1alpha1"

	// APIVersionV1 是当前推荐使用的稳定版本。
	APIVersionV1 APIVersion = "linctl.dev/v1"
)

// KindProject 是 linctl 当前唯一支持的 kind。未来若引入 Plugin / Module 等，复用同一 schema 体系。
const KindProject = "Project"

// Project 对应用户编辑的 `linctl.yaml`。
//
// 注意：**Project 不含 Status 字段**（SSOT §1.7）。运行时状态由独立的 `PROJECT`
// 文件承载，对应 [ProjectState] 类型。两者均入 git，但用户**仅编辑** `linctl.yaml`，
// 永不手动改 `PROJECT`。
type Project struct {
	APIVersion APIVersion `yaml:"apiVersion" json:"apiVersion" validate:"required,oneof=linctl.dev/v1 linctl.dev/v1alpha1"`
	Kind       string     `yaml:"kind"       json:"kind"       validate:"required,oneof=Project"`
	Metadata   Metadata   `yaml:"metadata"   json:"metadata"`
	Spec       Spec       `yaml:"spec"       json:"spec"`
}

// Metadata 描述项目的身份信息（不可变核心 + 可选标签）。
type Metadata struct {
	Name        string            `yaml:"name"                  json:"name"                  validate:"required,projectname"`
	Module      string            `yaml:"module"                json:"module"                validate:"required,modulepath"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty" validate:"max=200"`
	Author      Author            `yaml:"author,omitempty"      json:"author,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"      json:"labels,omitempty"`
}

// Author 是项目维护者信息。Email 字段做 RFC 5321 校验（仅在非空时）。
type Author struct {
	Name  string `yaml:"name,omitempty"  json:"name,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty" validate:"omitempty,email"`
}

// Spec 是用户输入的项目骨架。包含 defaults（项目级默认）、components（1..N）、
// hooks（生命周期钩子）。
type Spec struct {
	Defaults   Defaults    `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Components []Component `yaml:"components"         json:"components"         validate:"required,min=1,dive"`
	Hooks      Hooks       `yaml:"hooks,omitempty"    json:"hooks,omitempty"`
}

// Defaults 是项目级默认值，可被每个 Component 覆盖。
//
// **字段命名修正（SSOT §1.18）**：proto 文件版本前缀字段名为 `protoVersion`
// （而非历史的 `apiVersion`），避免与顶层 [Project.APIVersion] 多义。
type Defaults struct {
	Framework    string             `yaml:"framework,omitempty"    json:"framework,omitempty"    validate:"omitempty,oneof=gin grpc"`
	Storage      string             `yaml:"storage,omitempty"      json:"storage,omitempty"      validate:"omitempty,oneof=memory gorm-mysql gorm-postgres gorm-sqlite mongo"`
	Deploy       string             `yaml:"deploy,omitempty"       json:"deploy,omitempty"       validate:"omitempty,oneof=none docker kubernetes systemd"`
	Makefile     string             `yaml:"makefile,omitempty"     json:"makefile,omitempty"     validate:"omitempty,oneof=none unstructured structured"`
	ProtoVersion string             `yaml:"protoVersion,omitempty" json:"protoVersion,omitempty" validate:"omitempty,startswith=v"`
	Image        *ImageDefaults     `yaml:"image,omitempty"        json:"image,omitempty"`
	Docs         *DocsDefaults      `yaml:"docs,omitempty"         json:"docs,omitempty"`
	Telemetry    *TelemetryDefaults `yaml:"telemetry,omitempty"    json:"telemetry,omitempty"`
}

// ImageDefaults 控制容器镜像生成行为。
type ImageDefaults struct {
	RegistryPrefix string `yaml:"registryPrefix,omitempty" json:"registryPrefix,omitempty"`
	DockerfileMode string `yaml:"dockerfileMode,omitempty" json:"dockerfileMode,omitempty" validate:"omitempty,oneof=none runtime-only multi-stage combined"`
	DistrolessMode string `yaml:"distrolessMode,omitempty" json:"distrolessMode,omitempty" validate:"omitempty,oneof=always never auto"`
}

// DocsDefaults 控制文档骨架的语言。
type DocsDefaults struct {
	Languages []string `yaml:"languages,omitempty" json:"languages,omitempty" validate:"omitempty,dive,oneof=zh-CN en-US ja-JP"`
}

// TelemetryDefaults 控制 logging / metrics / tracing 三者的实现选择。
type TelemetryDefaults struct {
	Logging string `yaml:"logging,omitempty" json:"logging,omitempty" validate:"omitempty,oneof=slog zap"`
	Metrics string `yaml:"metrics,omitempty" json:"metrics,omitempty" validate:"omitempty,oneof=prometheus otlp"`
	Tracing string `yaml:"tracing,omitempty" json:"tracing,omitempty" validate:"omitempty,oneof=otlp jaeger zipkin"`
}

// Component 是项目下的一个独立可生成单元（WebServer / Worker / CLI）。
//
// 字段子集：
//   - 通用：Kind / Name / Features / Framework / Storage
//   - WebServer：Port / GRPCPort / Registry / Clients / Resources
//   - Worker：Variants / Cron / Kafka / Customized
//   - CLI：Commands
//
// 跨字段约束（如 `framework=grpc` 时 `grpcPort` 必填）由 Loader 后阶段的强校验承载，
// 不在 struct tag 内表达。
type Component struct {
	Kind     string   `yaml:"kind" json:"kind" validate:"required,oneof=WebServer Worker CLI"`
	Name     string   `yaml:"name" json:"name" validate:"required,componentname"`
	Features []string `yaml:"features,omitempty" json:"features,omitempty" validate:"omitempty,dive,featurename"`

	Framework string `yaml:"framework,omitempty" json:"framework,omitempty" validate:"omitempty,oneof=gin grpc"`
	Storage   string `yaml:"storage,omitempty"   json:"storage,omitempty"   validate:"omitempty,oneof=memory gorm-mysql gorm-postgres gorm-sqlite mongo"`

	// WebServer-only 字段
	Port         int  `yaml:"port,omitempty"         json:"port,omitempty"         validate:"omitempty,min=1024,max=65535"`
	GRPCPort     int  `yaml:"grpcPort,omitempty"     json:"grpcPort,omitempty"     validate:"omitempty,min=1024,max=65535"`
	// GrpcGateway enables grpc-gateway (HTTP/JSON) in front of gRPC. Only for framework=grpc;
	// uses Port for HTTP and GRPCPort (or 9090) for the native gRPC listener.
	GrpcGateway  bool `yaml:"grpcGateway,omitempty"  json:"grpcGateway,omitempty"`
	Registry  string     `yaml:"registry,omitempty"  json:"registry,omitempty"  validate:"omitempty,oneof=none polaris nacos consul eureka"`
	Clients   []string   `yaml:"clients,omitempty"   json:"clients,omitempty"`
	Resources []Resource `yaml:"resources,omitempty" json:"resources,omitempty" validate:"omitempty,dive"`

	// Worker-only 字段
	Variants   []string    `yaml:"variants,omitempty"   json:"variants,omitempty"   validate:"omitempty,dive,oneof=cron kafka customized"`
	Cron       *CronSpec   `yaml:"cron,omitempty"       json:"cron,omitempty"`
	Kafka      *KafkaSpec  `yaml:"kafka,omitempty"      json:"kafka,omitempty"`
	Customized []NamedSpec `yaml:"customized,omitempty" json:"customized,omitempty" validate:"omitempty,dive"`

	// CLI-only 字段
	Commands []NamedSpec `yaml:"commands,omitempty" json:"commands,omitempty" validate:"omitempty,dive"`
}

// Resource 描述一个已被 linctl 维护的 REST 资源。Path 与 CreatedAt 是工具回写字段。
type Resource struct {
	Name      string `yaml:"name"                json:"name"                validate:"required,kindname"`
	Path      string `yaml:"path,omitempty"      json:"path,omitempty"`
	CreatedAt string `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`
}

// CronSpec 是 Worker 启用 cron variant 时的配置载荷。
type CronSpec struct {
	Jobs []NamedSpec `yaml:"jobs" json:"jobs" validate:"required,min=1,dive"`
}

// KafkaSpec 是 Worker 启用 kafka variant 时的配置载荷。
type KafkaSpec struct {
	Brokers []string    `yaml:"brokers" json:"brokers" validate:"required,min=1,dive,hostname_port"`
	Topics  []NamedSpec `yaml:"topics"  json:"topics"  validate:"required,min=1,dive"`
}

// NamedSpec 是仅含 name 字段的简单结构（cron job / kafka topic / cli command 共用）。
type NamedSpec struct {
	Name string `yaml:"name" json:"name" validate:"required,kindname"`
}

// Hooks 是项目级生命周期钩子的集合（pre/post apply）。
type Hooks struct {
	PreApply  []Hook `yaml:"preApply,omitempty"  json:"preApply,omitempty"  validate:"omitempty,dive"`
	PostApply []Hook `yaml:"postApply,omitempty" json:"postApply,omitempty" validate:"omitempty,dive"`
}

// Hook 描述一条 shell 命令的执行配置。
//
// **Policy 字段（SSOT §5.3）**：
//   - "restricted"：仅允许 allowlist 中的命令前缀（gofumpt / go fmt / goimports / buf / make / protoc 等）。
//   - "confirm"（**本地默认**）：每条 hook 命令独立确认（即使有 `-y`）。
//   - "unrestricted"：执行任意 shell 命令；**仅本地可用**，必须二次确认；CI 环境
//     检测到 `unrestricted` 时使用退出码 7（安全策略违规）。
//
// 留空表示"使用全局策略"——本地为 confirm，CI 强制 restricted。
type Hook struct {
	Name   string `yaml:"name"             json:"name"             validate:"required"`
	Run    string `yaml:"run"              json:"run"              validate:"required"`
	Policy string `yaml:"policy,omitempty" json:"policy,omitempty" validate:"omitempty,oneof=restricted confirm unrestricted"`
}
