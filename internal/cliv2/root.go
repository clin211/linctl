// Package cliv2 是 lin v2 的 cobra 命令分发层骨架。
//
// 设计来源：lin/docs/features/02-command-set.md §1「命令一览」、§2「全局 flag」。
//
// MVP（Phase 1）阶段：
//   - 仅注册 root 命令 + 全局 flag + 占位子命令（version 已可用，其它返回 stub）
//   - lin --help 应能打印预期的命令列表
//   - 后续 Phase 2-4 各 stage 逐步替换占位为真实实现
//
// 注意：v1 的命令集合在 internal/cli 包，与此包并存不冲突。
package cliv2

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/clin211/lin/internal/pkg/errs"
)

// Globals 持有所有命令共享的全局 flag 状态。
type Globals struct {
	LogLevel       string
	LogFormat      string
	NoColor        bool
	NonInteractive bool
	Yes            bool
	Chdir          string
}

// NewRootCmd 构造 lin v2 的根命令（含全局 flag 与所有子命令）。
func NewRootCmd() *cobra.Command {
	g := &Globals{}

	root := &cobra.Command{
		Use:           "lin",
		Short:         "Go project scaffolder (miniblog-v4 style)",
		Long: `lin is a minimal scaffold generator for Go backend services.

It does exactly two things:
  1. lin new <project>        Generate a fresh project skeleton.
  2. lin add <Resource>...    Add a full-stack resource to an existing project.

Plus a few helpers: lint / doctor / version / completion.`,
		Version:       "2.0.0-rc1",
		SilenceErrors: true,
		SilenceUsage:  true,

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if g.Chdir != "" {
				if err := os.Chdir(g.Chdir); err != nil {
					return errs.Wrap(errs.CodeInvalidArg, "chdir failed", err)
				}
			}
			return nil
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&g.LogLevel, "log-level", "info", "Log level: debug|info|warn|error")
	pf.StringVar(&g.LogFormat, "log-format", "text", "Log format: text|json")
	pf.StringVarP(&g.Chdir, "chdir", "C", "", "Change to dir before running (like git -C)")
	pf.BoolVar(&g.NoColor, "no-color", false, "Disable colorized output")
	pf.BoolVar(&g.NonInteractive, "non-interactive", false, "Force non-interactive mode")
	pf.BoolVarP(&g.Yes, "yes", "y", false, "Skip final confirmation prompt")

	root.AddCommand(
		newNewCmd(g),
		newAddCmd(g),
		newLintCmd(g),
		newDoctorCmd(g),
		newVersionCmd(g),
		newCompletionCmd(),
	)

	return root
}

// Execute 是 cmd/lin/main.go 调用的统一入口。
func Execute(ctx context.Context, args []string) error {
	root := NewRootCmd()
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

// stubCmd 在 Phase 1 阶段为尚未实现的命令提供占位 RunE。
func stubCmd(name string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"⚠  %s is a Phase 1 skeleton; not yet implemented.\n"+
				"   Track progress in lin/docs/features/06-migration-plan.md\n",
			name)
		return errs.New(errs.CodeUnknown,
			fmt.Sprintf("%s: not implemented in Phase 1 skeleton", name))
	}
}
