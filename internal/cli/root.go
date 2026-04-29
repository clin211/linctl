// Package cli 实现 linctl 的命令行界面（基于 cobra）。
//
// 设计要点（详见 docs/03-cli-design.md）：
//   - 唯一入口 Execute(ctx, args) 由 cmd/linctl/main.go 调用
//   - 全局 flag 与子命令分离（globals.go）
//   - 每个变更类子命令实现 5 段式生命周期（Complete → Validate → Plan → Apply → Report）
package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

// Execute 是 linctl CLI 的唯一入口。它构造完整的命令树、解析参数、绑定 ctx，并执行匹配的子命令。
//
// 返回的 error 由 main 通过 exitCodeFor 映射为退出码。所有用户面错误应为 *linctlerr.LinctlError。
func Execute(ctx context.Context, args []string) error {
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	// SilenceErrors=true 避免 cobra 自己打印 error；由 main 统一处理
	root.SilenceErrors = true
	root.SilenceUsage = true

	return root.ExecuteContext(ctx)
}

// NewRootCommand 构造 linctl 的根命令树。导出供测试 / 文档生成器使用。
func NewRootCommand() *cobra.Command {
	g := &GlobalOptions{}

	root := &cobra.Command{
		Use:           "linctl",
		Short:         "linctl - declarative Go microservice scaffold (plan/apply oriented)",
		Long:          longDescription,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	g.RegisterFlags(root.PersistentFlags())

	root.AddCommand(
		newVersionCmd(g),
		newNewCmd(g),
		newAddCmd(g),
		newInternalCmd(g), // hidden: maintainer-only commands (templatesync, etc.)
		// 后续 Story 1.10 在此追加：
		// newDoctorCmd(g),
		// newCompletionCmd(g),
	)

	return root
}

const longDescription = `linctl is a declarative, plan/apply-oriented scaffold tool
for Go microservices. It generates idempotent, AST-friendly code that you can
re-run safely (no overwrites of your changes).

Documentation: https://github.com/clin211/linctl
Architecture:  See docs/01-architecture.md
`

