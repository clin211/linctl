package {{ .Component.Name }}

import (
	"context"
	"log"
	"sync"
)

// runCustomized 启动所有 customized 任务，每个任务用独立 goroutine。
//
// Customized variant 适合「事件驱动 / 长连接 / 自定义触发」类场景：
{{- range .Component.Customized }}
//   - {{ .Name }}
{{- end }}
func runCustomized(ctx context.Context) {
	var wg sync.WaitGroup
{{- range .Component.Customized }}
	wg.Add(1)
	go func() {
		defer wg.Done()
		run{{ .Name | pascal }}(ctx)
	}()
{{- end }}
	wg.Wait()
	log.Println("customized: all tasks finished")
}

{{ range .Component.Customized }}
// run{{ .Name | pascal }} 是 customized 任务 {{ .Name }} 的入口。
func run{{ .Name | pascal }}(ctx context.Context) {
	// TODO: implement {{ .Name }} business logic.
	<-ctx.Done()
	log.Println("customized[{{ .Name }}]: stopping")
}

{{ end }}
