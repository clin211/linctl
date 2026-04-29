# {{ .Component.Name }} 服务配置（Docker Compose 环境）
#
# 与 {{ .Component.Name }}.yaml 同结构，但所有外部依赖替换为 docker-compose
# 内网中的服务名，假设服务与 app 在同一 compose network 下。
#
# 与 Stream D（docker-compose 模板）的契约：
#   - 数据库服务名：postgres（gorm-postgres）/ mysql（gorm-mysql）/ mongo（mongo）
#   - 缓存服务名：redis
#   - 链路收集器服务名：otel-collector
#   - compose 网络名：{{ snake .Project.Metadata.Name }}_net
#
# 端口为容器内端口（非宿主机映射端口）。
http:
  network: tcp
  addr: 0.0.0.0:{{ default 8080 .Component.Port }} # 服务监听地址
  timeout: 30s # 服务端超时

tls:
  use-tls: false
  insecure-skip-verify: false
  ca-cert: ""
  cert: ""
  key: ""

jwt-key: Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5 # JWT 签名密钥（长度 >= 6）
expiration: 2h # JWT Token 过期时长

{{- if eq .Component.Storage "gorm-mysql" }}
mysql:
  addr: mysql:3306 # docker-compose 内网服务名
  username: root
  password: ""
  database: {{ snake .Project.Metadata.Name }}
  max-idle-connections: 100
  max-open-connections: 100
  max-connection-life-time: 10s
  log-level: 1
{{- end }}

{{- if eq .Component.Storage "gorm-postgres" }}
postgresql:
  addr: postgres:5432 # docker-compose 内网服务名
  username: postgres
  password: postgres
  database: {{ snake .Project.Metadata.Name }}
  max-idle-connections: 100
  max-open-connections: 100
  max-connection-life-time: 10s
  log-level: 1
{{- end }}

{{- if eq .Component.Storage "mongo" }}
mongo:
  url: mongodb://mongo:27017 # docker-compose 内网服务名
  database: {{ snake .Project.Metadata.Name }}
  collection: default
  username: ""
  password: ""
  timeout: 30s
  tls:
    use-tls: false
{{- end }}

redis:
  addr: redis:6379 # docker-compose 内网服务名
  username: ""
  password: ""
  database: 0
  max-retries: 3
  min-idle-conns: 0
  dial-timeout: 5s
  read-timeout: 3s
  write-timeout: 3s
  pool-time: 4s
  pool-size: 10
  enable-trace: false

casbin:
  model: configs/casbin/model.conf
  policy: configs/casbin/policy.csv
  auto-load-policy-interval: 10s

otel:
  endpoint: otel-collector:4317 # docker-compose 内网服务名
  service-name: {{ .Component.Name }}
  output-mode: otel # docker 环境默认走 OTLP，方便聚合
  level: info
  add-source: true
  use-prometheus-endpoint: true
  slog:
    format: text
    time-format: "2006-01-02 15:04:05"
    output: stdout # 容器中默认输出到 stdout，由 docker logs 收集
