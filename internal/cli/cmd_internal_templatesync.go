package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/templatesync"
	"github.com/clin211/lin/internal/version"
	"github.com/spf13/cobra"
)

// newInternalCmd 构造 `linctl internal` 父命令组（隐藏在面向用户的 help 中）。
//
// 此命令族**仅供 lin 仓库维护者使用**，不属于终端用户可见的 CLI 表面。
// 详见 docs/META-template-upstream-sync-2026-04-28.md。
func newInternalCmd(g *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "internal",
		Short:  "Maintainer-only internal commands (not for end users)",
		Hidden: true,
		Long: `linctl internal is a hidden command group reserved for lin repository
maintainers. It exposes utilities that operate on the lin repository itself
(e.g. syncing upstream demo project changes into the embedded templates),
not on user-generated projects.`,
	}
	cmd.AddCommand(newInternalTemplatesyncCmd(g))
	return cmd
}

// newInternalTemplatesyncCmd 构造 `linctl internal templatesync` 父命令。
func newInternalTemplatesyncCmd(g *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templatesync",
		Short: "Sync upstream demo project (e.g. miniblog-v4) into lin templates",
		Long: `Sync changes from an upstream demo project (declared in sync.yaml)
into lin/internal/template/templates/web-gin/.

This command operates on the lin repository itself, not on user projects.
It reads sync.yaml + upstream-sync.lock.json, applies declarative
transforms (rewriteImports, replaceLiteral, ...), and produces a Plan
or applies it.

Subcommands:
  status   show current sync status (changes since last sync, drift, unknowns)
  plan     compute a sync plan (no writes)
  apply    apply the plan (--strategy=force|abort|ours|theirs|ask)
  add      register new upstream file(s) into sync.yaml
  revert   restore one or more dst files to the lock-recorded state
  check    CI-friendly drift detection (exits non-zero when sync is required)
`,
	}
	cmd.AddCommand(newInternalTemplatesyncStatusCmd(g))
	cmd.AddCommand(newInternalTemplatesyncPlanCmd(g))
	cmd.AddCommand(newInternalTemplatesyncApplyCmd(g))
	cmd.AddCommand(newInternalTemplatesyncAddCmd(g))
	cmd.AddCommand(newInternalTemplatesyncRevertCmd(g))
	cmd.AddCommand(newInternalTemplatesyncCheckCmd(g))
	return cmd
}

// commonFlags 是 status/plan/apply 共享的 flag 集合。
type templatesyncFlags struct {
	linRoot      string
	manifestPath string
	lockPath     string
}

func registerCommonFlags(cmd *cobra.Command, f *templatesyncFlags) {
	cmd.Flags().StringVar(&f.linRoot, "lin-root", "",
		"Path to lin repository root (default: auto-detect from CWD by walking up to dir containing go.mod)")
	cmd.Flags().StringVar(&f.manifestPath, "manifest", "",
		"Path to sync.yaml (default: <lin-root>/internal/template/templates/web-gin/.sync.yaml)")
	cmd.Flags().StringVar(&f.lockPath, "lockfile", "",
		"Path to upstream-sync.lock.json (default: <lin-root>/.linctl/upstream-sync.lock.json)")
}

func resolveLinRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
				"resolve --lin-root %s", explicit)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", linctlerr.Wrap(linctlerr.ErrEnvironment, err, "getwd")
	}
	dir := cwd
	for {
		if isLinRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", linctlerr.New(linctlerr.ErrEnvironment,
		"could not auto-detect lin repository root from CWD",
		"Run from inside the lin/ directory, or pass --lin-root explicitly")
}

// isLinRoot 判断给定目录是否是 lin/ 根（含 go.mod 且 module path 为 linctl）。
func isLinRoot(dir string) bool {
	gomod := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(gomod)
	if err != nil {
		return false
	}
	// 简单判断：第一行 "module github.com/clin211/lin"
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			break
		}
	}
	return string(data[:7]) == "module " && contains(data, "/linctl")
}

func contains(haystack []byte, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func buildRunner(g *GlobalOptions, f *templatesyncFlags) (*templatesync.Runner, error) {
	linRoot, err := resolveLinRoot(f.linRoot)
	if err != nil {
		return nil, err
	}
	return templatesync.NewRunner(templatesync.RunnerOptions{
		LinRoot:       linRoot,
		ManifestPath:  f.manifestPath,
		LockPath:      f.lockPath,
		LinctlVersion: version.Get().Version,
	})
}

// ----------------------------------------------------------------------------
// status
// ----------------------------------------------------------------------------

func newInternalTemplatesyncStatusCmd(g *GlobalOptions) *cobra.Command {
	f := &templatesyncFlags{}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current upstream sync status",
		Long: `Show what would happen if you ran 'linctl internal templatesync apply'
right now: how many files are in-sync, need update, need merge, or are unknown
(present upstream but missing from sync.yaml).

Exit codes:
  0 = fully in sync (nothing to do)
  1 = updates pending (run 'apply')
  2 = conflicts / merge-needed pending (review required)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd.Context(), cmd.OutOrStdout(), g, f)
		},
	}
	registerCommonFlags(cmd, f)
	return cmd
}

func runStatus(ctx context.Context, out io.Writer, g *GlobalOptions, f *templatesyncFlags) error {
	r, err := buildRunner(g, f)
	if err != nil {
		return err
	}
	plan, err := r.Plan(ctx)
	if err != nil {
		return err
	}

	stats := computeStats(plan)
	switch g.Output {
	case "json":
		return printStatusJSON(out, plan, stats)
	default:
		printStatusText(out, plan, stats)
	}

	switch {
	case stats.MergeNeeded > 0 || stats.Conflict > 0:
		os.Exit(2)
	case stats.Create+stats.Update > 0 || len(plan.UnknownNew) > 0 || stats.UpstreamGone > 0:
		os.Exit(1)
	}
	return nil
}

type planStats struct {
	Create       int
	Update       int
	Skip         int
	MergeNeeded  int
	Conflict     int
	Preserve     int
	UpstreamGone int
	Unknown      int
}

func computeStats(plan *templatesync.Plan) planStats {
	s := planStats{Unknown: len(plan.UnknownNew)}
	for _, a := range plan.Actions {
		switch a.Kind {
		case templatesync.ActionCreate:
			s.Create++
		case templatesync.ActionUpdate:
			s.Update++
		case templatesync.ActionSkip:
			s.Skip++
		case templatesync.ActionMergeNeeded:
			s.MergeNeeded++
		case templatesync.ActionConflict:
			s.Conflict++
		case templatesync.ActionPreserve:
			s.Preserve++
		case templatesync.ActionUpstreamGone:
			s.UpstreamGone++
		}
	}
	return s
}

func printStatusText(out io.Writer, plan *templatesync.Plan, s planStats) {
	fmt.Fprintln(out, "linctl internal templatesync — status")
	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "  In-sync (skip):  %d\n", s.Skip)
	fmt.Fprintf(out, "  Will create:     %d\n", s.Create)
	fmt.Fprintf(out, "  Will update:     %d\n", s.Update)
	fmt.Fprintf(out, "  Need merge:      %d\n", s.MergeNeeded)
	fmt.Fprintf(out, "  Conflict:        %d\n", s.Conflict)
	fmt.Fprintf(out, "  Preserved:       %d\n", s.Preserve)
	fmt.Fprintf(out, "  Upstream gone:   %d\n", s.UpstreamGone)
	fmt.Fprintf(out, "  Unknown new:     %d\n", s.Unknown)
	fmt.Fprintln(out, "")

	if s.Create+s.Update+s.MergeNeeded+s.Conflict+s.UpstreamGone == 0 && s.Unknown == 0 {
		fmt.Fprintln(out, "✔ Everything is in sync.")
		return
	}

	for _, a := range plan.Actions {
		switch a.Kind {
		case templatesync.ActionCreate:
			fmt.Fprintf(out, "  + %s  (create; from %s)\n", a.Dst, a.Src)
		case templatesync.ActionUpdate:
			fmt.Fprintf(out, "  ~ %s  (update; %s)\n", a.Dst, a.Reason)
		case templatesync.ActionMergeNeeded:
			fmt.Fprintf(out, "  ! %s  (merge-needed; %s)\n", a.Dst, a.Reason)
		case templatesync.ActionConflict:
			fmt.Fprintf(out, "  ✗ %s  (conflict; %s)\n", a.Dst, a.Reason)
		case templatesync.ActionUpstreamGone:
			fmt.Fprintf(out, "  ? %s  (upstream-gone; src=%s)\n", a.Dst, a.Src)
		}
	}

	if len(plan.UnknownNew) > 0 {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Unknown new upstream files (not declared in sync.yaml):")
		for _, src := range plan.UnknownNew {
			fmt.Fprintf(out, "  ? %s\n", src)
		}
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Add to sync.yaml manually, or (in U2) run: linctl internal templatesync add <src>")
	}
}

func printStatusJSON(out io.Writer, plan *templatesync.Plan, s planStats) error {
	type actionDTO struct {
		Kind   string `json:"kind"`
		Src    string `json:"src,omitempty"`
		Dst    string `json:"dst"`
		Owner  string `json:"owner,omitempty"`
		Reason string `json:"reason,omitempty"`
	}
	dto := struct {
		Stats      planStats   `json:"stats"`
		Actions    []actionDTO `json:"actions"`
		UnknownNew []string    `json:"unknownNew"`
	}{Stats: s, UnknownNew: plan.UnknownNew}
	for _, a := range plan.Actions {
		dto.Actions = append(dto.Actions, actionDTO{
			Kind:   string(a.Kind),
			Src:    a.Src,
			Dst:    a.Dst,
			Owner:  string(a.Owner),
			Reason: a.Reason,
		})
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(dto)
}

// ----------------------------------------------------------------------------
// plan
// ----------------------------------------------------------------------------

func newInternalTemplatesyncPlanCmd(g *GlobalOptions) *cobra.Command {
	f := &templatesyncFlags{}
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Compute a sync plan (no writes)",
		Long: `Compute the same plan as 'apply' would execute, but never write to disk.
Useful for inspection or piping into 'jq' (combined with -o json).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := buildRunner(g, f)
			if err != nil {
				return err
			}
			plan, err := r.Plan(cmd.Context())
			if err != nil {
				return err
			}
			s := computeStats(plan)
			if g.Output == "json" {
				return printStatusJSON(cmd.OutOrStdout(), plan, s)
			}
			printStatusText(cmd.OutOrStdout(), plan, s)
			return nil
		},
	}
	registerCommonFlags(cmd, f)
	return cmd
}

// ----------------------------------------------------------------------------
// apply
// ----------------------------------------------------------------------------

func newInternalTemplatesyncApplyCmd(g *GlobalOptions) *cobra.Command {
	f := &templatesyncFlags{}
	var strategy string

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply the sync plan to lin/templates/web-gin/",
		Long: `Apply the sync plan to lin/templates/web-gin/. Strategies:
  --strategy=force  : overwrite all updates (default; lin-side changes lost for owner=shared)
  --strategy=abort  : abort if any merge-needed; do not write
  --strategy=ours   : keep lin-side changes; skip upstream updates
  --strategy=theirs : overwrite with upstream (3-way merge but conflicts resolved as theirs)
  --strategy=ask    : run 3-way merge via 'git merge-file'; write conflict markers
                      into the target files for the maintainer to resolve manually.
                      Conflicted files are reported but the apply does not fail.

Use --dry-run to print what would be written without touching the disk.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := buildRunner(g, f)
			if err != nil {
				return err
			}
			plan, err := r.Plan(cmd.Context())
			if err != nil {
				return err
			}

			strat, err := parseStrategy(strategy)
			if err != nil {
				return err
			}

			rep, err := r.Apply(cmd.Context(), plan, strat, g.DryRun)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "linctl internal templatesync — apply")
			if g.DryRun {
				fmt.Fprintln(out, "  (dry-run; no files written)")
			}
			fmt.Fprintf(out, "  Created:   %d\n", len(rep.Created))
			fmt.Fprintf(out, "  Updated:   %d\n", len(rep.Updated))
			fmt.Fprintf(out, "  Merged:    %d\n", len(rep.Merged))
			fmt.Fprintf(out, "  Skipped:   %d\n", len(rep.Skipped))
			fmt.Fprintf(out, "  Preserved: %d\n", len(rep.Preserved))
			if len(rep.Conflict) > 0 {
				fmt.Fprintf(out, "  Conflicts: %d\n", len(rep.Conflict))
				for _, c := range rep.Conflict {
					fmt.Fprintf(out, "    ✗ %s\n", c)
				}
			}
			return nil
		},
	}
	registerCommonFlags(cmd, f)
	cmd.Flags().StringVar(&strategy, "strategy", "force",
		"Conflict strategy: force | abort | ours | theirs (U2 will add: ask)")
	return cmd
}

func parseStrategy(s string) (templatesync.Strategy, error) {
	switch s {
	case "force":
		return templatesync.StrategyForce, nil
	case "abort":
		return templatesync.StrategyAbort, nil
	case "ours":
		return templatesync.StrategyOurs, nil
	case "theirs":
		return templatesync.StrategyTheirs, nil
	case "ask":
		// U2: 走 git merge-file 写入 conflict markers；
		// "交互式 prompt 每个文件" 留待 U3，当前 ask 等价于"自动写 markers 后退出非零"。
		return templatesync.StrategyAsk, nil
	default:
		return "", linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"unknown --strategy=%q (allowed: force/abort/ours/theirs/ask)", s)
	}
}
