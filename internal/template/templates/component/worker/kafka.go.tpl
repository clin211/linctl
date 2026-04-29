package {{ .Component.Name }}

import (
	"context"
	"log"
)

// kafkaBrokers / kafkaTopics 是 linctl.yaml 中声明的连接信息。
// 实际消费需要安装 kafka 客户端（建议 `segmentio/kafka-go`）；
// MVP 先以纯日志桩占位，确保 `go build ./...` 在零依赖下可通过。
var (
	kafkaBrokers = []string{
{{- range .Component.Kafka.Brokers }}
		{{ printf "%q" . }},
{{- end }}
	}
	kafkaTopics = []string{
{{- range .Component.Kafka.Topics }}
		{{ printf "%q" .Name }},
{{- end }}
	}
)

// runKafka 启动 kafka consumer 占位。
//
// Wiring real consumption（推荐流程）：
//  1. go get github.com/segmentio/kafka-go
//  2. 用 kafka.NewReader(...) 在 handle{{ pascal "Topic" }}() 中读消息
//  3. 用 ctx 控制 graceful shutdown
func runKafka(ctx context.Context) {
	log.Printf("kafka: brokers=%v topics=%v (placeholder, no real consumer)", kafkaBrokers, kafkaTopics)
{{- range .Component.Kafka.Topics }}
	go handle{{ .Name | pascal }}(ctx)
{{- end }}
	<-ctx.Done()
	log.Println("kafka: stopping")
}

{{ range .Component.Kafka.Topics }}
// handle{{ .Name | pascal }} consumes messages from the {{ .Name }} topic.
func handle{{ .Name | pascal }}(ctx context.Context) {
	// TODO: replace with real kafka consumer (see segmentio/kafka-go).
	<-ctx.Done()
}

{{ end }}
