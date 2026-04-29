{{- $project := .Project.Metadata.Name -}}
{{- $storage := (index .Project.Spec.Components 0).Storage -}}
# 本地开发依赖（postgres / redis / otel-collector）的 Compose 编排。
#
# 使用方法：
#   docker compose -f docker-compose.env.yml up -d
#   docker compose -f docker-compose.env.yml down
#
# 端口约定：业务端口尽量贴近 miniblog 习惯（54321/56379/4327/4328）以与
# configs/{{ (index .Project.Spec.Components 0).Name }}.yaml 默认值匹配；
# 如需改为标准 5432/6379/4317/4318，请同步修改 configs 与 networks。
services:
  postgres:
    image: postgres:16-alpine
    container_name: {{ $project }}-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: {{ $project }}
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      PGDATA: /var/lib/postgresql/data/pgdata
    ports:
      - "54321:5432"   # 宿主机 54321 → 容器 5432，避免与本地 PG 冲突
    volumes:
      - ./data/pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d {{ $project }}"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks:
      - {{ $project }}_net
    security_opt:
      - no-new-privileges:true
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

  redis:
    image: redis:7-alpine
    container_name: {{ $project }}-redis
    restart: unless-stopped
    command:
      - "redis-server"
      - "--appendonly"
      - "yes"
      - "--requirepass"
      - "redis_dev_password"   # 仅供本地开发；生产请改为强密码 + secret
    environment:
      REDIS_PASSWORD: redis_dev_password
    ports:
      - "56379:6379"   # 宿主机 56379 → 容器 6379
    volumes:
      - ./data/redisdata:/data
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a redis_dev_password ping | grep PONG"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks:
      - {{ $project }}_net
    security_opt:
      - no-new-privileges:true
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.97.0
    container_name: {{ $project }}-otel-collector
    restart: unless-stopped
    command: ["--config=/etc/otelcol/config.yaml"]
    ports:
      - "4327:4327"   # OTLP gRPC（与 app 的 telemetry.endpoint 配置一致）
      - "4328:4328"   # OTLP HTTP（可选）
      - "13133:13133" # Collector 健康检查
    volumes:
      - ./otel-collector.yaml:/etc/otelcol/config.yaml:ro
    healthcheck:
      test: ["CMD-SHELL", "wget --quiet --tries=1 --spider http://localhost:13133 || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks:
      - {{ $project }}_net
    security_opt:
      - no-new-privileges:true
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

networks:
  {{ $project }}_net:
    driver: bridge

volumes:
  pgdata:
  redisdata:
{{- if eq $storage "mongo" }}

# 提示：当前组件 storage=mongo，但本 compose 仍提供 postgres/redis 以方便切换。
# 如需 mongo，可参考下方 stub（默认未启用）：
#
#  mongo:
#    image: mongo:7
#    container_name: {{ $project }}-mongo
#    restart: unless-stopped
#    ports:
#      - "27017:27017"
#    volumes:
#      - ./data/mongodata:/data/db
#    networks:
#      - {{ $project }}_net
{{- end }}
