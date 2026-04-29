package main

import (
	"os"

	"{{ .Project.Metadata.Module }}/cmd/{{ .Component.Name }}/app"
)

// main 是 Go 程序的默认入口，仅做最小化的命令分发。
func main() {
	command := app.NewWebServerCommand()

	// 执行命令并处理错误。
	if err := command.Execute(); err != nil {
		// 出错时退出程序，并返回非零退出码，便于 shell 脚本判断进程状态。
		os.Exit(1)
	}
}
