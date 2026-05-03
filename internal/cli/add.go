package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/clin211/lin/internal/pkg/errs"
	"github.com/clin211/lin/internal/scaffold"
)

func newAddCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <Resource>...",
		Aliases: []string{"generate"},
		Short:   "Add full-stack business resource(s) to an existing project",
		Long: `Add one or more full-stack business resources (handler/biz/store/model/...).

Each resource follows the miniblog-v4 layout:
  - handler/<lower>.go
  - biz/v1/<lower>/<lower>.go + create/update/delete/get/list.go
  - store/<lower>.go
  - model/<lower>.gen.go
  - pkg/conversion/<lower>.go (--with conversion)
  - pkg/validation/<lower>.go (--with validation)
  - internal/pkg/errno/<lower>.go (--with errno)
  - pkg/api/<app>/v1/<lower>.proto (--with proto)

Plus AST injection into 4 central files.`,
		Args: cobra.MinimumNArgs(0),
		RunE: runAdd(g),
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

func runAdd(g *Globals) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// 1. Parse flags
		flags := cmd.Flags()

		appName, _ := flags.GetString("app")
		with, _ := flags.GetStringSlice("with")
		without, _ := flags.GetStringSlice("without")
		ops, _ := flags.GetStringSlice("ops")
		version, _ := flags.GetString("version")
		plural, _ := flags.GetString("plural")
		templateDir, _ := flags.GetString("template-dir")
		dryRun, _ := flags.GetBool("dry-run")
		noInject, _ := flags.GetBool("no-inject")
		skipImports, _ := flags.GetBool("skip-imports")
		force, _ := flags.GetBool("force")

		// 2. Validate resource names
		if len(args) == 0 {
			if g.NonInteractive {
				return errs.New(errs.CodeInvalidArg,
					"add: no resource names provided").
					WithHint("usage: linctl add Post [Comment...] [--with conversion,validation,proto,errno]")
			}
			return errs.New(errs.CodeInvalidArg,
				"add: no resource names provided").
				WithHint("usage: linctl add Post [Comment...] [--with conversion,validation,proto,errno]")
		}

		// 3. Detect project root (working directory)
		cwd, err := os.Getwd()
		if err != nil {
			return errs.Wrap(errs.CodeUnknown, "add: get working directory", err)
		}

		// 4. LoadContext
		ctx, err := scaffold.LoadContext(cwd, scaffold.Flags{
			AppName:     appName,
			Chdir:       g.Chdir,
			TemplateDir: templateDir,
			DryRun:      dryRun,
			Force:       force,
		})
		if err != nil {
			return err
		}

		fmt.Printf("✔ context loaded   module=%s appName=%s storage=%s\n",
			ctx.Module, ctx.AppName, ctx.Storage)

		// 5. Build AddOptions
		opts := scaffold.AddOptions{
			With:        with,
			Without:     without,
			Ops:         ops,
			Version:     version,
			Plural:      plural,
			NoInject:    noInject,
			SkipImports: skipImports,
		}

		// 6. Add each resource
		for _, name := range args {
			if err := scaffold.AddResource(ctx, name, opts); err != nil {
				// Map to user-visible exit code
				code := errs.CodeOf("add", err)
				fmt.Fprintf(cmd.ErrOrStderr(),
					"✗ add %s failed (exit %d): %v\n", name, code, err)
				if he, ok := err.(*errs.Error); ok && he.Hint != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "  Hint: %s\n", he.Hint)
				}
				return err
			}
		}

		return nil
	}
}
