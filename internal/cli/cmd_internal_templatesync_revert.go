package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/spf13/cobra"
)

// newInternalTemplatesyncRevertCmd 实现 `linctl internal templatesync revert <dst>...`。
//
// 把指定 dst 文件还原到 lockfile 中记录的"上次同步状态"内容（从 base cache 取）。
// 用途：维护者实验性修改了某个 .tpl 后想回滚到上次 sync 的纯净版本。
//
// 限制：
//   - dst 必须在 lockfile 中存在；否则报错
//   - lockfile.dstHashAtSync 对应的 base cache 必须仍在；否则报错
//   - 不会更新 lockfile（认为这就是 lockfile 中的状态；revert 后再 status 应是 in-sync）
func newInternalTemplatesyncRevertCmd(g *GlobalOptions) *cobra.Command {
	f := &templatesyncFlags{}
	var allFlag bool

	cmd := &cobra.Command{
		Use:   "revert <dst>...",
		Short: "Revert local edits to the last-synced state recorded in lockfile",
		Long: `Revert one or more dst files in lin/templates/web-gin/ back to the content
recorded in upstream-sync.lock.json (sourced from .linctl/upstream-sync-cache/).

Use --all to revert every file tracked by the lockfile.

Examples:
  # revert a specific file
  linctl internal templatesync revert internal/pkg/contextx/contextx.go.tpl

  # revert several files
  linctl internal templatesync revert pkg/log/log.go.tpl pkg/authn/authn.go.tpl

  # revert everything (use carefully)
  linctl internal templatesync revert --all`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !allFlag && len(args) == 0 {
				return linctlerr.New(linctlerr.ErrConfigInvalid,
					"revert: pass at least one <dst> or use --all")
			}

			r, err := buildRunner(g, f)
			if err != nil {
				return err
			}
			lf := r.Lockfile()
			cache := r.BaseCache()
			templatesRoot := r.LinTemplatesAbsRoot()

			targets := args
			if allFlag {
				targets = lf.AllDsts()
			}
			if len(targets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(),
					"lockfile is empty; nothing to revert.")
				return nil
			}

			reverted := 0
			missing := 0
			notInLock := 0
			for _, dst := range targets {
				rec, ok := lf.Get(dst)
				if !ok {
					fmt.Fprintf(cmd.OutOrStdout(),
						"  ? %s  (not tracked in lockfile; skip)\n", dst)
					notInLock++
					continue
				}
				if rec.DstHashAtSync == "" {
					fmt.Fprintf(cmd.OutOrStdout(),
						"  ? %s  (lockfile entry has no DstHashAtSync; skip)\n", dst)
					missing++
					continue
				}
				content, hit, err := cache.Get(rec.DstHashAtSync)
				if err != nil {
					return err
				}
				if !hit {
					fmt.Fprintf(cmd.OutOrStdout(),
						"  ! %s  (cache miss for %s; cannot revert)\n",
						dst, shortHash(rec.DstHashAtSync))
					missing++
					continue
				}

				dstAbs := filepath.Join(templatesRoot, dst)
				if g.DryRun {
					fmt.Fprintf(cmd.OutOrStdout(),
						"  ~ would revert: %s (%d bytes)\n", dst, len(content))
					reverted++
					continue
				}
				if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
					return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
						"mkdir %s", filepath.Dir(dst))
				}
				tmp := dstAbs + ".tmp"
				if err := os.WriteFile(tmp, content, 0o644); err != nil {
					return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
						"write tmp %s", tmp)
				}
				if err := os.Rename(tmp, dstAbs); err != nil {
					_ = os.Remove(tmp)
					return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
						"rename %s -> %s", tmp, dstAbs)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  ~ reverted: %s\n", dst)
				reverted++
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"\n%d reverted, %d missing-cache/no-hash, %d not-in-lockfile.\n",
				reverted, missing, notInLock)
			if g.DryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "(dry-run; nothing actually written)")
			}
			return nil
		},
	}
	registerCommonFlags(cmd, f)
	cmd.Flags().BoolVar(&allFlag, "all", false,
		"Revert every dst tracked by the lockfile (no positional args required)")
	return cmd
}

// shortHash 把 sha256:abcdef... 截短为 abcdef…（仅展示用）。
func shortHash(h string) string {
	const prefix = "sha256:"
	if len(h) > len(prefix)+8 {
		return h[len(prefix) : len(prefix)+8]
	}
	return h
}
