server:
  http:
    addr: 0.0.0.0:8080  # HTTP 监听地址（容器内）

timeout: 30s  # 服务端请求超时
{{- if eq .Storage "gorm-postgres" }}

postgresql:
  addr: postgres:5432
  username: postgres
  password: postgres
  database: {{.AppName}}
  max-idle-connections: 100
  max-open-connections: 1000
  max-connection-life-time: 5m
  log-level: 4
{{- end }}
{{- if eq .Storage "gorm-mysql" }}

mysql:
  addr: mysql:3306
  username: root
  password: root
  database: {{.AppName}}
  max-idle-connections: 100
  max-open-connections: 1000
  max-connection-life-time: 5m
{{- end }}
{{- if eq .Storage "gorm-sqlite" }}

sqlite:
  path: /app/data/{{.AppName}}.db
  database: ""
  max-idle-connections: 2
  max-open-connections: 5
  max-connection-life-time: 30s
{{- end }}
{{- if eq .Storage "mongo" }}

mongo:
  url: mongodb://mongodb:27017
  database: {{.AppName}}
  collection: app
  username: ""
  password: ""
  timeout: 10s
{{- end }}
{{- if eq .Cache "redis" }}

redis:
  addr: redis:6379
  password: ""
  db: 0
  max-retries: 3
  min-idle-conns: 0
  dial-timeout: 5s
  read-timeout: 3s
  write-timeout: 3s
  pool-timeout: 4s
  pool-size: 10
{{- end }}
{{- if .Features | Has "otel" }}

otel:
  endpoint: otel-collector:4327
  service-name: {{.AppName}}
  output-mode: classic
  level: info
  add-source: true
{{- end }}
