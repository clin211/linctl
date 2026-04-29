package cliv2

import "github.com/spf13/cobra"

func newLintCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Validate project structure and AST anchor integrity",
		Long: `Lint runs a set of checks over the project structure and AST anchors.

Phase 1 status: skeleton only. Full implementation tracked in lin/docs/features/06-migration-plan.md (Phase 4).`,
		RunE: stubCmd("lint"),
	}

	flags := cmd.Flags()
	flags.Bool("fix", false, "Auto-fix issues (e.g. restore missing anchors)")
	flags.String("report-format", "text", "Output format: text|json")
	flags.StringSlice("rules", nil, "Subset of rule IDs to enable (default: all)")
	flags.StringSlice("skip", nil, "Rule IDs to skip")

	return cmd
}
