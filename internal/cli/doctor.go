package cli

import (
	"github.com/spf13/cobra"

	"github.com/clin211/lin/internal/check"
	"github.com/clin211/lin/internal/pkg/errs"
)

func newDoctorCmd(g *Globals) *cobra.Command {
	var opts struct {
		ReportFormat string
		Strict       bool
		Offline      bool
		Checks       []string
		Skip         []string
	}

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local toolchain and runtime environment",
		Long: `Doctor inspects the local environment for tools required by linctl and the generated project.

Checks include:
  - go/version        go >= 1.22 (error)
  - go/goflags        GOFLAGS must not contain -mod=vendor (warning)
  - git/version       git >= 2.30 (warning)
  - tools/protoc      protoc in PATH (warning; needed for linctl add --with proto)
  - tools/protoc-gen-go  protoc-gen-go in PATH (warning)
  - tools/wire        wire in PATH (warning)
  - system/terminal   TTY detection + terminal size + color support (info)
  - network/proxy     proxy.golang.org reachable (info; use --offline to skip)

Exit codes:
  0  all error-level checks pass
  35 at least one error item
  36 --strict and at least one warning`,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := check.Doctor(check.DoctorOptions{
				Strict:       opts.Strict,
				Offline:      opts.Offline,
				ReportFormat: opts.ReportFormat,
				Checks:       opts.Checks,
				Skip:         opts.Skip,
			})
			if err != nil {
				return err
			}

			if err := check.PrintReport(report, opts.ReportFormat, cmd.OutOrStdout()); err != nil {
				return errs.Wrap(errs.CodeUnknown, "doctor: print report", err)
			}

			if report.Summary.Errors > 0 {
				return errs.New(errs.CodeDoctorErrors,
					"doctor: environment has errors; fix the reported issues before using linctl")
			}

			if opts.Strict && report.Summary.Warnings > 0 {
				return errs.New(errs.CodeDoctorWarnStrict,
					"doctor: --strict mode: warnings are treated as errors")
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.ReportFormat, "report-format", "text", "Output format: text|json")
	flags.BoolVar(&opts.Strict, "strict", false, "Treat warnings as errors (exit 36)")
	flags.BoolVar(&opts.Offline, "offline", false, "Skip network-dependent checks (CI/airgap friendly)")
	flags.StringSliceVar(&opts.Checks, "check", nil, "Subset of check IDs to run (default: all)")
	flags.StringSliceVar(&opts.Skip, "skip", nil, "Check IDs to skip")

	return cmd
}
