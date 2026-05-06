{{- /* docker-compose.env.yml: 仅用于本地启动周边依赖服务（数据库、缓存、可观测性等）。 */ -}}
{{- /* 真正的应用容器请使用 build/docker/{{.AppName}}/docker-compose.yml */ -}}
services:
{{- if eq .Storage "gorm-postgres" }}
  postgres:
    image: postgres:16-alpine
    container_name: {{.AppName}}-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: {{.AppName}}
    ports:
      - "5432:5432"
    volumes:
      - ./data/pgdata:/var/lib/postgresql/data
    restart: unless-stopped
{{- end }}
{{- if eq .Storage "gorm-mysql" }}
  mysql:
    image: mysql:8.0
    container_name: {{.AppName}}-mysql
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: {{.AppName}}
    ports:
      - "3306:3306"
    volumes:
      - ./data/mysql:/var/lib/mysql
    restart: unless-stopped
{{- end }}
{{- if eq .Storage "mongo" }}
  mongodb:
    image: mongo:7
    container_name: {{.AppName}}-mongodb
    ports:
      - "27017:27017"
    volumes:
      - ./data/mongodb:/data/db
    restart: unless-stopped
{{- end }}
{{- if eq .Cache "redis" }}
  redis:
    image: redis:7-alpine
    container_name: {{.AppName}}-redis
    ports:
      - "6379:6379"
    volumes:
      - ./data/redisdata:/data
    restart: unless-stopped
{{- end }}
{{- if .Features | Has "otel" }}
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.108.0
    container_name: {{.AppName}}-otel-collector
    command: ["--config=/etc/otel-collector.yaml"]
    volumes:
      - ./otel-collector.yaml:/etc/otel-collector.yaml:ro
    ports:
      - "4327:4327"   # OTLP gRPC
      - "4328:4328"   # OTLP HTTP
    restart: unless-stopped
{{- end }}
