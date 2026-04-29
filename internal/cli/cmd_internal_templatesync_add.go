package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/templatesync"
	"github.com/spf13/cobra"
)

// newInternalTemplatesyncAddCmd 实现 `linctl internal templatesync add <upstream-path> [...]`。
//
// 把上游某个文件追加到 sync.yaml 的 files: 块。dst 自动按 sync.yaml 的
// addExtension transform 推断；owner 默认 upstream 可被 --owner 覆盖。
//
// 示例：
//
//	# 自动推断 dst（加 .tpl 后缀）
//	linctl internal templatesync add miniblog-v4/pkg/cache/redis_cache.go
//	linctl internal templatesync add internal/pkg/cache/redis_cache.go      # rootPath 相对路径也接受
//
//	# 显式指定 dst（src 与 dst 同名时；如本身就是 .proto）
//	linctl internal templatesync add api/v1/x.proto --dst api/v1/x.proto.tpl
//
//	# 一次加多个文件
//	linctl internal templatesync add pkg/a.go pkg/b.go pkg/c.go --owner shared
func newInternalTemplatesyncAddCmd(g *GlobalOptions) *cobra.Command {
	f := &templatesyncFlags{}
	var (
		dstFlag   string
		ownerFlag string
	)

	cmd := &cobra.Command{
		Use:   "add <upstream-path>...",
		Short: "Add upstream file(s) to sync.yaml",
		Long: `Append entries to sync.yaml's 'files:' block. The dst path is auto-inferred
from the upstream src by appending '.tpl' (per the sync.yaml addExtension transform).

If you pass exactly one src and want a custom dst path, use --dst.
The default owner is "upstream"; use --owner to set "shared" or "linSpecific".

Examples:
  # auto-add new file (single)
  linctl internal templatesync add internal/pkg/cache/redis_cache.go

  # batch add with owner override
  linctl internal templatesync add pkg/a.go pkg/b.go --owner shared

  # explicit dst
  linctl internal templatesync add api/v1/x.proto --dst api/v1/x.proto.tpl`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dstFlag != "" && len(args) != 1 {
				return linctlerr.New(linctlerr.ErrConfigInvalid,
					"--dst can only be used with exactly one upstream-path argument")
			}

			r, err := buildRunner(g, f)
			if err != nil {
				return err
			}
			m := r.Manifest()

			owner, err := parseOwner(ownerFlag)
			if err != nil {
				return err
			}

			upstreamRoot, err := m.AbsUpstreamRoot()
			if err != nil {
				return err
			}

			// 解析 manifest path（buildRunner 已用过；再算一次，避免暴露内部字段）
			manifestPath := f.manifestPath
			if manifestPath == "" {
				linRoot, _ := resolveLinRoot(f.linRoot)
				manifestPath = filepath.Join(linRoot,
					"internal/template/templates/web-gin/.sync.yaml")
			}

			added := 0
			skipped := 0
			for _, raw := range args {
				rel, err := normalizeUpstreamSrc(raw, upstreamRoot)
				if err != nil {
					return err
				}

				if existing := m.FindBySrc(rel); existing != nil {
					fmt.Fprintf(cmd.OutOrStdout(),
						"  = already in sync.yaml: %s -> %s (owner=%s)\n",
						rel, existing.Dst, existing.Owner)
					skipped++
					continue
				}

				dst := dstFlag
				if dst == "" {
					dst = inferDstFromSrc(rel)
				}

				entry := templatesync.FileMapping{
					Src:   rel,
					Dst:   dst,
					Owner: owner,
				}

				if g.DryRun {
					fmt.Fprintf(cmd.OutOrStdout(),
						"  + would add: %s -> %s (owner=%s)\n", rel, dst, owner)
					added++
					continue
				}

				if err := templatesync.AppendFileEntry(manifestPath, entry); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(),
					"  + added: %s -> %s (owner=%s)\n", rel, dst, owner)
				added++
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\n%d added, %d already-present.\n",
				added, skipped)
			if g.DryRun {
				fmt.Fprintln(cmd.OutOrStdout(),
					"(dry-run; sync.yaml not modified)")
			} else if added > 0 {
				fmt.Fprintln(cmd.OutOrStdout(),
					"\nNext: run 'linctl internal templatesync apply' to sync the new entries.")
			}
			return nil
		},
	}
	registerCommonFlags(cmd, f)
	cmd.Flags().StringVar(&dstFlag, "dst", "",
		"Override the inferred dst path (only valid when adding a single file)")
	cmd.Flags().StringVar(&ownerFlag, "owner", "upstream",
		"File ownership: upstream | shared | linSpecific (default: upstream)")
	return cmd
}

// normalizeUpstreamSrc 把用户输入的 src 路径规范化为相对 upstream root 的形式。
//
// 接受多种输入：
//   - 相对 upstream root：internal/pkg/cache/x.go
//   - 含 upstream 名前缀：miniblog-v4/internal/pkg/cache/x.go
//   - 绝对路径：/Users/.../miniblog-v4/internal/pkg/cache/x.go
func normalizeUpstreamSrc(raw, upstreamRoot string) (string, error) {
	if raw == "" {
		return "", linctlerr.New(linctlerr.ErrConfigInvalid,
			"upstream-path argument cannot be empty")
	}

	// 绝对路径 → 直接 Rel
	if filepath.IsAbs(raw) {
		rel, err := filepath.Rel(upstreamRoot, raw)
		if err != nil {
			return "", linctlerr.Wrapf(linctlerr.ErrConfigInvalid, err,
				"path %q is not under upstream root %q", raw, upstreamRoot)
		}
		if strings.HasPrefix(rel, "..") {
			return "", linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"path %q is outside upstream root %q", raw, upstreamRoot)
		}
		return filepath.ToSlash(rel), nil
	}

	cleaned := filepath.ToSlash(filepath.Clean(raw))

	// 含上游目录前缀：miniblog-v4/...
	upstreamBase := filepath.Base(upstreamRoot)
	if strings.HasPrefix(cleaned, upstreamBase+"/") {
		return strings.TrimPrefix(cleaned, upstreamBase+"/"), nil
	}

	// 已是相对 root 形式
	return cleaned, nil
}

// inferDstFromSrc 把 src 推断为对应的 lin 镜像 dst 路径。
//
// 规则：保留路径，加 .tpl 后缀（只对 .go / .proto / .yaml / .yml / .json
// 等文本文件加；其他扩展名原样返回，让用户用 --dst 显式指定）。
func inferDstFromSrc(src string) string {
	tplable := []string{".go", ".proto", ".yaml", ".yml", ".json", ".md", ".sh", ".bash", ".sql"}
	for _, ext := range tplable {
		if strings.HasSuffix(src, ext) {
			return src + ".tpl"
		}
	}
	return src
}

// parseOwner 把用户输入字符串解析为 templatesync.Owner（含校验）。
func parseOwner(s string) (templatesync.Owner, error) {
	switch strings.ToLower(s) {
	case "upstream", "":
		return templatesync.OwnerUpstream, nil
	case "shared":
		return templatesync.OwnerShared, nil
	case "linspecific", "lin-specific", "lin_specific":
		return templatesync.OwnerLinSpecific, nil
	}
	return "", linctlerr.Newf(linctlerr.ErrConfigInvalid,
		"invalid --owner=%q (allowed: upstream/shared/linSpecific)", s)
}
