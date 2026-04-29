package component

import (
	"sort"

	"github.com/clin211/linctl/internal/ast"
	"github.com/clin211/linctl/internal/codegen"
	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/project"
)

// CLIKind 是独立 CLI 工具组件的注册 Kind。
const CLIKind = "CLI"

// CLI 是 linctl 内置的命令行工具组件实现（Phase 3 Story 3.4）。
//
// 适合「随项目分发的运维 / 管理工具」场景：例如配套 webserver 的 `admin` 工具，
// 提供 `dump` / `migrate` / `seed` 等子命令。
//
// 与 WebServer / Worker 的差异：
//   - 无 framework / variants
//   - Commands 必填（每个子命令一个文件 + 一次注册）
type CLI struct {
	spec project.Component
}

// NewCLI 通过 project.Component 构造 CLI 实例。
func NewCLI(spec project.Component) *CLI {
	return &CLI{spec: spec}
}

// CLIFactory 是 Registry 用的 Factory（接受 map[string]any 形式）。
func CLIFactory(_ map[string]any) (Component, error) {
	return &CLI{}, nil
}

// Kind 返回 "CLI"。
func (c *CLI) Kind() string { return CLIKind }

// Name 返回组件实例名。
func (c *CLI) Name() string { return c.spec.Name }

// Validate 校验 CLI 配置：Name + 至少一个 Command。
func (c *CLI) Validate(_ *project.Project) error {
	if c.spec.Name == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"CLI.Name is required",
			"Set components[].name to a non-empty kebab-case identifier")
	}
	if len(c.spec.Commands) == 0 {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"CLI.Commands is required",
			"Add at least one entry under spec.commands[].name")
	}
	seen := make(map[string]bool, len(c.spec.Commands))
	for _, cmd := range c.spec.Commands {
		if cmd.Name == "" {
			return linctlerr.New(linctlerr.ErrConfigInvalid,
				"CLI.Commands[].Name is required")
		}
		if seen[cmd.Name] {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"CLI.Commands has duplicate %q", cmd.Name)
		}
		seen[cmd.Name] = true
	}
	return nil
}

// BasePairs 返回 CLI 自身的骨架文件 Pair 列表。
//
// 文件布局：
//   - cmd/<name>/main.go              # cobra root + 注册所有子命令
//   - internal/<name>/cmd/all.go      # all.go 聚合所有子命令引用（AST 友好的扩展点）
//   - internal/<name>/cmd/<command>.go # 每个子命令独立文件
func (c *CLI) BasePairs(p *project.Project) []codegen.Pair {
	owner := "CLI:" + c.spec.Name

	pairs := []codegen.Pair{
		{
			Dst:        "cmd/" + c.spec.Name + "/main.go",
			TemplateID: "templates/component/cli/cmd_main.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + c.spec.Name + "/cmd/all.go",
			TemplateID: "templates/component/cli/all.go.tpl",
			Owner:      owner,
		},
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

	// 字典序处理子命令，输出稳定（snapshot 友好）。
	cmds := make([]string, 0, len(c.spec.Commands))
	for _, cmd := range c.spec.Commands {
		cmds = append(cmds, cmd.Name)
	}
	sort.Strings(cmds)
	for _, name := range cmds {
		// command 模板是为单个命令渲染的，需要在 .Project / .Component 之外
		// 知道「当前命令是哪一个」。用 map 包装，模板里通过 .CommandName 访问。
		// 与 cmd_add.go 中 resource 模板的 Custom 字段思路一致。
		pairs = append(pairs, codegen.Pair{
			Dst:        "internal/" + c.spec.Name + "/cmd/" + name + ".go",
			TemplateID: "templates/component/cli/command.go.tpl",
			Owner:      owner,
			Data: map[string]any{
				"Project":     p,
				"Component":   c.spec,
				"CLIVersion":  "dev",
				"CommandName": name,
			},
		})
	}
	return pairs
}

// BaseMutators 返回组件 AST 修改。Phase 3 第一波不做 AST 注入；
// `linctl add cli <name> <cmd>` 命令会在 all.go 注入 import + 注册调用（待实现）。
func (c *CLI) BaseMutators(_ *project.Project) []ast.ASTMutator {
	return nil
}

// PostProcess 不需要副作用。
func (c *CLI) PostProcess(_ *project.Project, _ FileSystem) error {
	return nil
}

// SpecComponent 暴露原始 YAML struct（仅供 orchestrator 调度时用）。
func (c *CLI) SpecComponent() project.Component {
	return c.spec
}
