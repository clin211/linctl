package {{ .Component.Name }}

import (
	"context"
	"log"
	"sync"
)

// Runner aggregates all enabled variants of the {{ .Component.Name }} worker
// and starts them in parallel goroutines.
//
// Each variant lives in its own file in this package:
{{- range .Component.Variants }}
//   - {{ . }}.go
{{- end }}
type Runner struct {
	wg sync.WaitGroup
}

// NewRunner constructs a default Runner.
func NewRunner() *Runner {
	return &Runner{}
}

// Run starts every enabled variant in its own goroutine and blocks until
// the context is canceled. It returns nil after all variants have shut down.
func (r *Runner) Run(ctx context.Context) error {
{{- range .Component.Variants }}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		run{{ . | pascal }}(ctx)
	}()
{{- end }}

	log.Println("all variants started")
	<-ctx.Done()
	log.Println("shutting down...")

	r.wg.Wait()
	return nil
}
