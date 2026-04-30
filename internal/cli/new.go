package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/clin211/lin/internal/pkg/errs"
	"github.com/clin211/lin/internal/scaffold"
)

func newNewCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "new <project-name>",
		Short:        "Generate a new Go project skeleton",
		Long:         `Generate a fresh miniblog-v4 style project skeleton.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(cmd, args, g)
		},
	}

	flags := cmd.Flags()
	flags.String("module", "", "Go module path (e.g. github.com/foo/myblog)")
	flags.String("app-name", "", "Application name (default: derived from project-name)")
	flags.String("framework", "gin", "Web framework: gin (MVP only)")
	flags.String("storage", "memory", "Storage layer: memory|gorm-postgres|gorm-mysql|gorm-sqlite|mongo")
	flags.StringSlice("features", nil, "Optional features: otel,healthz,user,swagger,preloader")
	flags.String("template-dir", "", "External template directory (overrides embed)")
	flags.String("output-dir", ".", "Parent directory to create project in")
	flags.String("author", "", "Author name (default: git config user.name)")
	flags.String("email", "", "Author email (default: git config user.email)")
	flags.Bool("force", false, "Overwrite if target dir exists")
	flags.Bool("dry-run", false, "Print plan, do not write files")
	flags.Bool("yes", false, "Skip final confirmation prompt")
	flags.Bool("non-interactive", false, "Force non-interactive mode (required when stdin is not a TTY)")

	return cmd
}

func runNew(cmd *cobra.Command, args []string, g *Globals) error {
	flags := cmd.Flags()

	// project-name 必填（非交互模式下）
	if len(args) == 0 {
		return errs.New(errs.CodeInvalidArg, "project-name is required").
			WithHint("usage: lin new <project-name> --module <module-path>")
	}
	projectName := args[0]

	module, _ := flags.GetString("module")
	appName, _ := flags.GetString("app-name")
	framework, _ := flags.GetString("framework")
	storage, _ := flags.GetString("storage")
	features, _ := flags.GetStringSlice("features")
	templateDir, _ := flags.GetString("template-dir")
	outputDir, _ := flags.GetString("output-dir")
	author, _ := flags.GetString("author")
	email, _ := flags.GetString("email")
	force, _ := flags.GetBool("force")
	dryRun, _ := flags.GetBool("dry-run")

	// 全局 --yes / --non-interactive 也覆盖局部
	if g.Yes {
		// no-op, confirmation is already skipped
	}

	sf := scaffold.Flags{
		ProjectName: projectName,
		Module:      module,
		AppName:     appName,
		Framework:   framework,
		Storage:     storage,
		Features:    features,
		TemplateDir: templateDir,
		OutputDir:   outputDir,
		Author:      author,
		Email:       email,
		Force:       force,
		DryRun:      dryRun,
		NonInteract: g.NonInteractive,
	}

	ctx, err := scaffold.NewContextFromFlags(sf)
	if err != nil {
		if e, ok := err.(*errs.Error); ok {
			fmt.Fprintf(cmd.ErrOrStderr(), "✗ %s\n", e.Message)
			if e.Hint != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "  Hint: %s\n", e.Hint)
			}
		}
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔ project plan computing for %q ...\n", projectName)

	if err := scaffold.NewProject(ctx); err != nil {
		if e, ok := err.(*errs.Error); ok {
			fmt.Fprintf(cmd.ErrOrStderr(), "✗ %s\n", e.Message)
			if e.Hint != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "  Hint: %s\n", e.Hint)
			}
		}
		return err
	}

	return nil
}
