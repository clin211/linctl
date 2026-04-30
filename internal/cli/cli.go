// Package cli is the cobra command-dispatch layer for lin v2.
//
// Layout convention:
//   - cli.go        registers the root command and global flags.
//   - add.go        `lin add <Resource>...`
//   - new.go        `lin new <project>`
//   - lint.go       `lin lint`
//   - doctor.go     `lin doctor`
//   - version.go    `lin version`
//   - completion.go `lin completion <shell>`
//
// Design source: lin/docs/features/02-command-set.md §1, §2.
package cli

import (
	"context"
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

