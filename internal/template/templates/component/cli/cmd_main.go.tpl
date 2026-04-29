package main

import (
	"fmt"
	"os"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/cmd"
)

func main() {
	if err := cmd.Root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
