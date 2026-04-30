server:
  http:
    addr: 0.0.0.0:8080  # HTTP listen address

timeout: 30s  # Server request timeout
{{- if ne .Storage "memory"}}

postgresql:
  addr: 127.0.0.1:5432
  username: postgres
  password: postgres
  database: {{.AppName}}
  max-idle-connections: 100
  max-open-connections: 1000
  max-connection-life-time: 5m
  log-level: 4  # 1=silent 2=error 3=warn 4=info
{{- end}}
{{- if eq .Storage "gorm-mysql"}}

mysql:
  addr: 127.0.0.1:3306
  username: root
  password: root
  database: {{.AppName}}
  max-idle-connections: 100
  max-open-connections: 1000
  max-connection-life-time: 5m
{{- end}}
{{- if .Features | Has "otel"}}

otel:
  endpoint: 127.0.0.1:4317
  service-name: {{.AppName}}
  output-mode: classic
  level: info
  add-source: true
{{- end}}
