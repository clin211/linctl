package {{ .Component.Name }}

import (
	"context"
	"log"
	"time"
)

// runCron drives all configured cron jobs.
//
// MVP uses a simple time.Ticker (1-minute resolution) to keep the
// generated project free of external dependencies. Replace with
// `github.com/robfig/cron/v3` when you need crontab-spec support.
//
// Configured jobs (see linctl.yaml spec.cron.jobs):
{{- range .Component.Cron.Jobs }}
//   - {{ .Name }}
{{- end }}
func runCron(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Printf("cron: tick every 1m")
	for {
		select {
		case <-ctx.Done():
			log.Println("cron: stopping")
			return
		case t := <-ticker.C:
{{- range .Component.Cron.Jobs }}
			tick{{ .Name | pascal }}(ctx, t)
{{- end }}
		}
	}
}

{{ range .Component.Cron.Jobs }}
// tick{{ .Name | pascal }} runs the {{ .Name }} cron job for one tick.
func tick{{ .Name | pascal }}(_ context.Context, _ time.Time) {
	// TODO: implement {{ .Name }} business logic.
	log.Printf("cron[{{ .Name }}]: noop")
}

{{ end }}
