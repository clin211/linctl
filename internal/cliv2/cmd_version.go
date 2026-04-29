package cliv2

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// 由 ldflags 注入；MVP 阶段使用默认值。
var (
	version   = "2.0.0-rc1"
	commit    = "dev"
	buildTime = "unknown"
)

type versionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Built     string `json:"built"`
	GoVersion string `json:"go"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func newVersionCmd(g *Globals) *cobra.Command {
	var (
		short  bool
		format string
	)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print lin version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := versionInfo{
				Version:   version,
				Commit:    commit,
				Built:     buildTime,
				GoVersion: runtime.Version(),
				OS:        runtime.GOOS,
				Arch:      runtime.GOARCH,
			}

			if short {
				fmt.Fprintln(cmd.OutOrStdout(), info.Version)
				return nil
			}

			switch format {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "lin version %s\n", info.Version)
				fmt.Fprintf(cmd.OutOrStdout(), "  commit:   %s\n", info.Commit)
				fmt.Fprintf(cmd.OutOrStdout(), "  built:    %s\n", info.Built)
				fmt.Fprintf(cmd.OutOrStdout(), "  go:       %s\n", info.GoVersion)
				fmt.Fprintf(cmd.OutOrStdout(), "  os/arch:  %s/%s\n", info.OS, info.Arch)
				return nil
			}
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "Print only semver")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text|json|yaml")

	return cmd
}
