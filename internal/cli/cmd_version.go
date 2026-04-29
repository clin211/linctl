package cli

import (
	"encoding/json"
	"fmt"

	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/version"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newVersionCmd 实现 `linctl version`。
//
// 支持 --output text/json/yaml（统一全局 flag），输出 internal/version.Info。
func newVersionCmd(g *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print linctl version information",
		Long:  "Print linctl version, commit, build date, Go version and platform.",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()
			out := g.Output
			if out == "" {
				out = "text"
			}
			w := cmd.OutOrStdout()

			switch out {
			case "text":
				fmt.Fprintln(w, info.String())
				return nil
			case "json":
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			case "yaml":
				return yaml.NewEncoder(w).Encode(info)
			default:
				return linctlerr.Newf(linctlerr.ErrConfigInvalid,
					"unknown --output format: %q", out).
					WithHint("Allowed: text, json, yaml")
			}
		},
	}
	return cmd
}
