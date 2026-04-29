# {{ .Component.Name }} 服务配置（开发环境）
#
# 与 miniblog-v4 / blog-apiserver.yaml 对齐的完整配置骨架。
# 同目录下的 {{ .Component.Name }}.docker.yaml 用于 docker-compose 内网部署。
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
  addr: 127.0.0.1:3306
  username: root
  password: ""
  database: {{ snake .Project.Metadata.Name }}
  max-idle-connections: 100 # 连接池最大空闲连接数
  max-open-connections: 100 # 连接池最大打开连接数
  max-connection-life-time: 10s # 单连接最长生命周期
  log-level: 1 # GORM 日志级别：Silent=1 / Error=2 / Warn=3 / Info=4
{{- end }}

{{- if eq .Component.Storage "gorm-postgres" }}
postgresql:
  addr: 127.0.0.1:5432
  username: postgres
  password: postgres
  database: {{ snake .Project.Metadata.Name }}
  max-idle-connections: 100 # 连接池最大空闲连接数
  max-open-connections: 100 # 连接池最大打开连接数
  max-connection-life-time: 10s # 单连接最长生命周期
  log-level: 1 # GORM 日志级别：Silent=1 / Error=2 / Warn=3 / Info=4
{{- end }}

{{- if eq .Component.Storage "mongo" }}
mongo:
  url: mongodb://127.0.0.1:27017
  database: {{ snake .Project.Metadata.Name }}
  collection: default
  username: ""
  password: ""
  timeout: 30s
  tls:
    use-tls: false
{{- end }}

# Redis 缓存与会话存储（默认启用）
redis:
  addr: 127.0.0.1:6379 # Redis 服务地址（host:port）
  username: "" # Redis ACL 用户名（默认用户留空）
  password: "" # Redis 密码（启用了 requirepass 才需要）
  database: 0 # 逻辑库索引
  max-retries: 3 # 命令最大重试次数
  min-idle-conns: 0 # 最小空闲连接数
  dial-timeout: 5s # 连接建立超时
  read-timeout: 3s # 读超时
  write-timeout: 3s # 写超时
  pool-time: 4s # 连接池获取连接超时
  pool-size: 10 # 连接池大小
  enable-trace: false # 是否启用 OTel trace

# Casbin RBAC 配置
casbin:
  model: configs/casbin/model.conf # Casbin 模型文件路径
  policy: configs/casbin/policy.csv # Casbin 策略文件路径（file-adapter 使用）
  auto-load-policy-interval: 10s # 策略自动重载间隔

otel:
  endpoint: 127.0.0.1:4317
  service-name: {{ .Component.Name }}
  output-mode: classic # 可选：otel | console | file | classic | hybrid
  level: debug
  add-source: true
  use-prometheus-endpoint: true
  slog:
    # 仅在 output-mode=classic / hybrid 时生效
    format: json
    time-format: "2006-01-02 15:04:05"
    output: ./_output/{{ .Component.Name }}.log # 支持 stdout / stderr / 文件路径
