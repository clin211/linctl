package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/clin211/lin/internal/templatesync"
	"github.com/spf13/cobra"
)

// newInternalTemplatesyncCheckCmd 实现 `linctl internal templatesync check`。
//
// 设计目标：CI 闸门 — 当上游有改动但 lin/templates/web-gin/ 没同步时让 PR 失败。
//
// 退出码：
//   0 = ok（in-sync 且无 unknown-new）
//   1 = 需要 sync（有 create / update / merge-needed / upstream-gone / unknown-new）
//   2 = sync.yaml / lockfile 解析失败等内部问题（一般是 PR 引入 bug）
//
// JSON 输出（与 status 兼容；增加 needsSync 数组便于 GitHub Action 直接消费）：
//
//	{
//	  "ok": false,
//	  "needsSync": [
//	    {"src": "internal/pkg/contextx/contextx.go", "dst": "internal/pkg/contextx/contextx.go.tpl", "kind": "update"},
//	    ...
//	  ],
//	  "unknownNew": ["pkg/cache/redis_cache.go", ...]
//	}
func newInternalTemplatesyncCheckCmd(g *GlobalOptions) *cobra.Command {
	f := &templatesyncFlags{}
	var quiet bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "CI-friendly drift detection (exit non-zero if upstream changes require sync)",
		Long: `Compute the same plan as 'status' / 'plan' but designed for CI gating.

Exit codes:
  0 = clean (no upstream drift, no unknown-new)
  1 = sync required (any create / update / merge-needed / upstream-gone / unknown-new)
  2 = internal failure (sync.yaml or lockfile parse error)

Use -o json to produce a stable JSON report consumable by GitHub Actions / jq:

  linctl internal templatesync check -o json | jq '.needsSync'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := buildRunner(g, f)
			if err != nil {
				return err
			}
			plan, err := r.Plan(cmd.Context())
			if err != nil {
				return err
			}
			return runCheck(cmd.Context(), cmd.OutOrStdout(), g, plan, quiet)
		},
	}
	registerCommonFlags(cmd, f)
	cmd.Flags().BoolVar(&quiet, "quiet", false,
		"Only print exit code; suppress output (useful for shell scripts)")
	return cmd
}

func runCheck(_ context.Context, out io.Writer, g *GlobalOptions, plan *templatesync.Plan, quiet bool) error {
	type drift struct {
		Src   string `json:"src,omitempty"`
		Dst   string `json:"dst"`
		Kind  string `json:"kind"`
		Owner string `json:"owner,omitempty"`
	}
	var needsSync []drift
	for _, a := range plan.Actions {
		switch a.Kind {
		case templatesync.ActionCreate,
			templatesync.ActionUpdate,
			templatesync.ActionMergeNeeded,
			templatesync.ActionConflict,
			templatesync.ActionUpstreamGone:
			needsSync = append(needsSync, drift{
				Src:   a.Src,
				Dst:   a.Dst,
				Kind:  string(a.Kind),
				Owner: string(a.Owner),
			})
		}
	}

	ok := len(needsSync) == 0 && len(plan.UnknownNew) == 0

	if g.Output == "json" {
		dto := struct {
			OK         bool     `json:"ok"`
			NeedsSync  []drift  `json:"needsSync"`
			UnknownNew []string `json:"unknownNew"`
		}{
			OK:         ok,
			NeedsSync:  needsSync,
			UnknownNew: plan.UnknownNew,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(dto); err != nil {
			return err
		}
		if !ok {
			os.Exit(1)
		}
		return nil
	}

	if quiet {
		if !ok {
			os.Exit(1)
		}
		return nil
	}

	if ok {
		fmt.Fprintln(out, "✔ Templates are in sync with miniblog-v4 upstream.")
		return nil
	}

	fmt.Fprintln(out, "⚠ Template upstream sync required:")
	for _, d := range needsSync {
		fmt.Fprintf(out, "  %s %s  (%s)\n", glyph(d.Kind), d.Dst, d.Kind)
	}
	if len(plan.UnknownNew) > 0 {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Unknown new upstream files (run 'templatesync add' to register):")
		for _, src := range plan.UnknownNew {
			fmt.Fprintf(out, "  ? %s\n", src)
		}
	}
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Run locally to fix:")
	fmt.Fprintln(out, "  linctl internal templatesync apply")
	os.Exit(1)
	return nil
}

func glyph(kind string) string {
	switch kind {
	case "create":
		return "+"
	case "update":
		return "~"
	case "merge-needed":
		return "!"
	case "conflict":
		return "✗"
	case "upstream-gone":
		return "?"
	}
	return " "
}
