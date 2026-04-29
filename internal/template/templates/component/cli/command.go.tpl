package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// new{{ .CommandName | pascal }}Cmd builds the `{{ .CommandName }}` sub-command.
//
// Edit this stub to wire real business logic. Flags / args / persistence
// live here; shared dependencies belong in internal/{{ .Component.Name }}/.
func new{{ .CommandName | pascal }}Cmd() *cobra.Command {
	return &cobra.Command{
		Use:   "{{ .CommandName }}",
		Short: "{{ .CommandName | pascal }} sub-command (replace with real description)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("{{ .CommandName }}: noop (replace with real implementation)")
			return nil
		},
	}
}
