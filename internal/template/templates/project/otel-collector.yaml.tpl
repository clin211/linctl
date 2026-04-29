{{- $project := .Project.Metadata.Name -}}
# OpenTelemetry Collector 配置（{{ $project }}）。
#
# - 接收：OTLP gRPC :4327 / HTTP :4328（与 docker-compose.env.yml 暴露端口一致）
# - 处理：batch（默认；如需 attributes / filter 等可按需扩展）
# - 导出：本地 logging（开发环境直接打印；生产请改 otlp/远端 backend）
# - 健康检查：:13133
#
# 切换到生产 backend（示例：Tempo / Loki / Prometheus / Jaeger / Datadog 等）：
#   exporters:
#     otlp/tempo:
#       endpoint: tempo:4317
#       tls:
#         insecure: true
#   service.pipelines.traces.exporters: [ otlp/tempo ]
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4327
      http:
        endpoint: 0.0.0.0:4328

processors:
  batch:

exporters:
  logging:
    loglevel: info

extensions:
  health_check:

service:
  extensions: [ health_check ]
  pipelines:
    traces:
      receivers: [ otlp ]
      processors: [ batch ]
      exporters: [ logging ]
    metrics:
      receivers: [ otlp ]
      processors: [ batch ]
      exporters: [ logging ]
    logs:
      receivers: [ otlp ]
      processors: [ batch ]
      exporters: [ logging ]
