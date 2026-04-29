package cliv2

import "github.com/spf13/cobra"

func newAddCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <Resource>...",
		Aliases: []string{"generate"},
		Short:   "Add full-stack business resource(s) to an existing project",
		Long: `Add one or more full-stack business resources (handler/biz/store/model/...).

Phase 1 status: skeleton only. Full implementation tracked in lin/docs/features/06-migration-plan.md (Phase 3).`,
		Args: cobra.MinimumNArgs(0),
		RunE: stubCmd("add"),
	}

	flags := cmd.Flags()
	flags.String("app", "", "Override app name (default: inferred from cmd/*)")
	flags.StringSlice("with", []string{"conversion", "validation", "proto", "errno"},
		"Optional layers: conversion,validation,proto,errno")
	flags.StringSlice("without", nil, "Inverse of --with; mutually exclusive")
	flags.StringSlice("ops", []string{"create", "update", "delete", "get", "list"},
		"CRUD verbs to generate")
	flags.String("version", "v1", "API version segment")
	flags.String("plural", "", "Override plural form (e.g. Octopuses)")
	flags.String("template-dir", "", "External template directory")
	flags.Bool("dry-run", false, "Print plan, do not write files")
	flags.Bool("no-inject", false, "Skip AST injection (only create new files)")
	flags.Bool("skip-imports", false, "Skip auto-add of import statements")
	flags.Bool("force", false, "Overwrite existing resource files (backed up to .lin/.backup/<ts>/)")

	return cmd
}
