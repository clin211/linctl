package main

import (
	"os"

	"{{.Module}}/cmd/{{.AppName}}/app"
)

// main is the default entry point of the application.
func main() {
	command := app.NewWebServerCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
