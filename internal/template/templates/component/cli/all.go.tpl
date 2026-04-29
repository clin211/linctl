package cmd

import (
	"github.com/spf13/cobra"
)

// Root returns the root cobra command for {{ .Component.Name }}.
//
// Sub-commands are registered below; `linctl add cli {{ .Component.Name }} <name>`
// will append a new line here (Phase 3 follow-up; for now edit manually).
func Root() *cobra.Command {
	root := &cobra.Command{
		Use:   "{{ .Component.Name }}",
		Short: "{{ .Component.Name }} CLI",
	}

{{- range .Component.Commands }}
	root.AddCommand(new{{ .Name | pascal }}Cmd())
{{- end }}

	return root
}
