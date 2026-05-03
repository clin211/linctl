server:
  http:
    addr: 0.0.0.0:8080  # HTTP listen address

timeout: 30s  # Server request timeout
{{- if eq .Storage "gorm-postgres"}}

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
{{- if eq .Storage "gorm-sqlite"}}

sqlite:
  path: ./data/{{.AppName}}.db
  database: ""
  max-idle-connections: 2
  max-open-connections: 5
  max-connection-life-time: 30s
{{- end}}
{{- if eq .Storage "mongo"}}

mongo:
  url: mongodb://127.0.0.1:27017
  database: {{.AppName}}
  collection: app
  username: ""
  password: ""
  timeout: 10s
{{- end}}
{{- if eq .Cache "redis"}}

redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
  max-retries: 3
  min-idle-conns: 0
  dial-timeout: 5s
  read-timeout: 3s
  write-timeout: 3s
  pool-timeout: 4s
  pool-size: 10
{{- end}}
{{- if eq .Cache "bigcache"}}

bigcache:
  shards: 1024
  life-window: 10m
  clean-window: 5m
  max-entries-in-window: 1000000
  max-entry-size: 500
  verbose: false
  hard-max-cache-size-mb: 0  # 0 = no hard limit
{{- end}}
{{- if .Features | Has "otel"}}

otel:
  endpoint: 127.0.0.1:4317
  service-name: {{.AppName}}
  output-mode: classic
  level: info
  add-source: true
{{- end}}
