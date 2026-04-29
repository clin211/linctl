// Package scaffold 是 lin v2 的骨架生成核心。
//
// 设计来源：lin/docs/features/01-architecture-blueprint.md §3「模块职责矩阵」§7「关键抽象」。
//
// 模块职责（L2 核心生成层）：
//   - context.go : 项目上下文（Module/AppName/Storage 推断）
//   - plan.go    : 文件清单 + 注入清单
//   - render.go  : 模板渲染封装（后续 stage）
//   - project.go : lin new 入口（后续 stage）
//   - resource.go: lin add 入口（后续 stage）
package scaffold

import (
	"github.com/clin211/lin/internal/pkg/tpl"
)

// Context 持有当前命令的全部上下文信息。
//
// 字段来源详见：
//   - lin new : 由命令行 flags 直接构造
//   - lin add : 由 LoadContext 从 ./go.mod / ./cmd/* 推断（详见 02 §4.4）
type Context struct {
	// RootDir 是项目根的绝对路径（必填）。
	RootDir string

	// Module 是 Go module 路径，从 go.mod 推断（add）或由用户传入（new）。
	Module string

	// AppName 是应用名，从 cmd/<app>/ 推断（add）或由 project-name 派生（new）。
	AppName string

	// Storage 表示存储后端：memory | gorm-postgres | gorm-mysql | mongo。
	Storage string

	// Framework 表示 Web 框架：MVP 阶段仅支持 gin。
	Framework string

	// Features 是启用的可选特性：otel | healthz | user | swagger 等。
	Features []string

	// Resource 仅在 lin add 时有效：PascalCase 资源名（如 "Post"）。
	Resource string

	// Templates 是模板加载器（按 04 §2 优先级查找）。
	Templates *tpl.Loader

	// DryRun 为 true 时仅打印计划，不写入磁盘。
	DryRun bool

	// Force 为 true 时允许覆盖既有文件（含 .lin/.backup/<ts>/ 备份）。
	Force bool
}

// Flags 是 LoadContext 的输入：从命令行解析的原始 flags。
//
// 设计意图：保持 cli 层与 scaffold 层解耦——cli 仅负责把 cobra flags 装进 Flags 结构体，
// scaffold 接管推断与校验。
type Flags struct {
	Module       string
	AppName      string
	Storage      string
	Framework    string
	Features     []string
	Resource     string
	TemplateDir  string
	DryRun       bool
	Force        bool
	NonInteract  bool
}

// LoadContext 用于 lin add：从已有项目根推断元信息（go.mod / cmd/<app>/ / store 中的 import）。
//
// MVP 阶段返回 placeholder error；Phase 3 由 ast/inferrer 实现。
func LoadContext(rootDir string, flags Flags) (*Context, error) {
	// TODO(Phase 3): 实现真正的推断逻辑。
	// 1) 读 ./go.mod 第一行 → Module
	// 2) 列 ./cmd/* 唯一子目录 → AppName（或要求 --app）
	// 3) 探测 ./internal/<app>/store/ 的 import → Storage
	return nil, errPhase1Stub
}

// NewContextFromFlags 用于 lin new：直接由 flags 构造（不需要推断）。
//
// MVP 阶段返回 placeholder error；Phase 2 由 cli/new 调用并补全。
func NewContextFromFlags(flags Flags) (*Context, error) {
	// TODO(Phase 2): 实现初始 Context 构造（含 Templates loader）。
	return nil, errPhase1Stub
}
