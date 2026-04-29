package cliv2

import "github.com/spf13/cobra"

func newDoctorCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local toolchain and runtime environment",
		Long: `Doctor inspects the local environment for tools required by lin and the generated project.

Phase 1 status: skeleton only. Full implementation tracked in lin/docs/features/06-migration-plan.md (Phase 4).`,
		RunE: stubCmd("doctor"),
	}

	flags := cmd.Flags()
	flags.String("report-format", "text", "Output format: text|json")
	flags.Bool("strict", false, "Treat warnings as errors")
	flags.Bool("offline", false, "Skip network-dependent checks (CI/airgap friendly)")
	flags.StringSlice("check", nil, "Subset of check IDs to run")
	flags.StringSlice("skip", nil, "Check IDs to skip")

	return cmd
}
