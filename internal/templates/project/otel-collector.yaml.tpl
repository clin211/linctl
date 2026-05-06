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
