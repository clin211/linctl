// Package component 提供 linctl 的 Component 抽象与内置实现。
//
// Component 是「可独立编译/运行的产物」的抽象（详见 docs/09-component-design.md §9.1）。
// 实现本接口的类型可被注册到 Registry，参与 plan/apply 流水线。
//
// 与 [project.Component]（YAML struct）的命名区别：
//   - component.Component（本接口）：行为契约
//   - project.Component（YAML struct）：用户配置数据
package component

import (
	"github.com/clin211/linctl/internal/ast"
	"github.com/clin211/linctl/internal/codegen"
	"github.com/clin211/linctl/internal/project"
)

// FileSystem 是 Component.PostProcess 对文件系统的访问抽象。
//
// PostProcess 必须通过此接口访问磁盘，禁止直接 os.OpenFile 等调用。
type FileSystem interface {
	Read(path string) ([]byte, error)
	Write(path string, content []byte) error
	Exists(path string) bool
}

// Component 是 linctl 中"可独立编译/运行的产物"的抽象。
//
// 设计契约（详见 docs/09-component-design.md §9.2）：
//   - BasePairs / BaseMutators 必须是纯函数：不做 IO、不读环境
//   - PostProcess 是显式声明的副作用扩展点，仅 Applier 在持锁阶段调用
//   - 接口方法 ≤ 6 个，便于第三方插件实现
type Component interface {
	// Kind 返回组件类型名，必须与 project.Component.Kind 一致（如 "WebServer"）。
	Kind() string

	// Name 返回组件实例名（项目内唯一）。
	Name() string

	// Validate 校验组件配置（结合 *project.Project 上下文）。
	Validate(p *project.Project) error

	// BasePairs 返回组件自身（不含 Feature）需要生成的 Pair 列表。纯函数。
	BasePairs(p *project.Project) []codegen.Pair

	// BaseMutators 返回组件自身需要的 AST 修改（不含 Feature 贡献）。纯函数。
	BaseMutators(p *project.Project) []ast.ASTMutator

	// PostProcess 是显式副作用扩展点。99% 场景应返回 nil。
	PostProcess(p *project.Project, fs FileSystem) error
}
