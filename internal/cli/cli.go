// Package cli is the cobra command-dispatch layer for linctl (lin v2).
//
// Layout convention:
//   - cli.go        registers the root command and global flags.
//   - add.go        `linctl add <Resource>...`
//   - new.go        `linctl new <project>`
//   - lint.go       `linctl lint`
//   - doctor.go     `linctl doctor`
//   - version.go    `linctl version`
//   - completion.go `linctl completion <shell>`
//
// Design source: lin/docs/features/02-command-set.md §1, §2.
package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/clin211/linctl/internal/pkg/errs"
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
		Use:           "linctl",
		Short:         "Go project scaffolder (layered architecture inspired by DDD)",
		Long: `linctl is a minimal scaffold generator for Go backend services.

It does exactly two things:
  1. linctl new <project>        Generate a fresh project skeleton.
  2. linctl add <Resource>...    Add a full-stack resource to an existing project.

Plus a few helpers: lint / doctor / version / completion.`,
		Version:       "0.1.0-alpha",
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

// Execute 是 main.go 调用的统一入口。
func Execute(ctx context.Context, args []string) error {
	root := NewRootCmd()
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

