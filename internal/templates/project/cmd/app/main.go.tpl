package main

import (
	"os"

	"{{.Module}}/cmd/{{.AppName}}/app"
)

// main 是应用程序的默认入口。
func main() {
	command := app.NewWebServerCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
