package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runner := {{ .Component.Name }}.NewRunner()

	log.Printf("worker {{ .Component.Name }} starting")
	if err := runner.Run(ctx); err != nil {
		log.Fatalf("worker exited with error: %v", err)
	}
	log.Println("bye")
}
