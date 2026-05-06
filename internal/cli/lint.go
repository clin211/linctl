package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/clin211/linctl/internal/check"
	"github.com/clin211/linctl/internal/pkg/errs"
)

func newLintCmd(g *Globals) *cobra.Command {
	var opts struct {
		Fix          bool
		DryRun       bool
		ReportFormat string
		Rules        []string
		Skip         []string
	}

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Validate project structure and registration consistency",
		Long: `Lint runs a set of checks over the project structure and registration consistency.

Checks include:
  - dir/cmd-app                  cmd/<app>/main.go exists
  - dir/internal-app             internal/<app>/{handler,biz,store,model} exist
  - register/biz-impl            every biz/v1/<res>/ is wired into biz.IBiz
  - register/store-impl          every store/<res>.go is wired into store.IStore
  - proto/missing-pb-go           .proto files missing generated .pb.go counterparts
  - safety/path-traversal        scaffold path safety check

--fix is reserved for future auto-fixers (currently a no-op for register/* rules).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return errs.Wrap(errs.CodeUnknown, "lint: get working directory", err)
			}

			report, err := check.Lint(cwd, check.LintOptions{
				Fix:          opts.Fix,
				DryRun:       opts.DryRun,
				Rules:        opts.Rules,
				Skip:         opts.Skip,
				ReportFormat: opts.ReportFormat,
			})
			if err != nil {
				return err
			}

			if err := check.PrintReport(report, opts.ReportFormat, cmd.OutOrStdout()); err != nil {
				return errs.Wrap(errs.CodeUnknown, "lint: print report", err)
			}

			if opts.Fix && opts.DryRun {
				// dry-run 仅作信息输出；错误已被降级为 info
				return nil
			}

			if report.Summary.Errors > 0 {
				if opts.Fix {
					return errs.New(errs.CodeFixPartial,
						fmt.Sprintf("lint: %d issue(s) could not be auto-fixed", report.Summary.Errors)).
						WithHint("run 'linctl lint' without --fix for details")
				}
				return errs.New(errs.CodeLintIssues,
					fmt.Sprintf("lint: %d issue(s) found", report.Summary.Errors)).
					WithHint("review the report and re-run 'linctl add <Resource>' if needed")
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.Fix, "fix", false, "Auto-fix issues (reserved; currently no-op)")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Print fix plan without modifying files (use with --fix)")
	flags.StringVar(&opts.ReportFormat, "report-format", "text", "Output format: text|json")
	flags.StringSliceVar(&opts.Rules, "rules", nil, "Subset of rule IDs to enable (default: all)")
	flags.StringSliceVar(&opts.Skip, "skip", nil, "Rule IDs to skip")

	return cmd
}
