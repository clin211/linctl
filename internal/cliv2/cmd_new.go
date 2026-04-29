package cliv2

import "github.com/spf13/cobra"

func newNewCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Generate a new Go project skeleton",
		Long: `Generate a fresh miniblog-v4 style project skeleton.

Phase 1 status: skeleton only. Run 'lin new --help' to inspect flags.
Full implementation tracked in lin/docs/features/06-migration-plan.md (Phase 2).`,
		Args: cobra.MaximumNArgs(1),
		RunE: stubCmd("new"),
	}

	flags := cmd.Flags()
	flags.String("module", "", "Go module path (e.g. github.com/foo/myblog)")
	flags.String("app-name", "", "Application name (default: derived from project-name)")
	flags.String("framework", "gin", "Web framework: gin (MVP only)")
	flags.String("storage", "memory", "Storage layer: memory|gorm-postgres|gorm-mysql|mongo")
	flags.StringSlice("features", nil, "Optional features: otel,healthz,user,swagger")
	flags.String("template-dir", "", "External template directory (overrides embed)")
	flags.String("output-dir", ".", "Parent directory to create project in")
	flags.Bool("force", false, "Overwrite if target dir exists")
	flags.Bool("dry-run", false, "Print plan, do not write files")

	return cmd
}
