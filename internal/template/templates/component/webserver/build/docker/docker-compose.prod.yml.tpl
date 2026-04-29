{{- $app := .Component.Name -}}
{{- $project := .Project.Metadata.Name -}}
{{- $registry := .Project.Spec.Defaults.Image.RegistryPrefix | default "ghcr.io/example" -}}
{{- $port := .Component.Port -}}
{{- if not $port }}{{ $port = 8080 }}{{ end }}
# 生产环境 Docker Compose（{{ $app }}）。
#
# 使用方法：
#   1. 在 CI 中构建并推送镜像：
#        make image PLATFORM=linux_amd64 VERSION=vX.Y.Z IMAGES={{ $app }}
#        docker push {{ $registry }}/{{ $app }}:vX.Y.Z
#   2. 准备配置文件：configs/{{ $app }}.prod.yaml
#   3. 启动服务（VERSION 从环境变量注入）：
#        VERSION=vX.Y.Z docker compose -f docker-compose.prod.yml up -d
#
# 跨服务器部署：
#   - 数据库 / Redis / OTEL 在其他主机：直接在配置文件中填 IP:Port
#   - 同宿主机：用 host.docker.internal:Port（已配 extra_hosts）
#   - 同 Docker 网络：取消注释 networks 段并使用服务名
services:
  {{ $app }}:
    image: {{ $registry }}/{{ $app }}:${VERSION:-latest}

    container_name: {{ $app }}
    hostname: {{ $app }}

    environment:
      - TZ=Asia/Shanghai
      # 业务可选环境变量（如果应用支持）
      # - LOG_LEVEL=info
      # - DATABASE_HOST=192.168.1.100

    volumes:
      - ./configs/{{ $app }}.prod.yaml:/app/configs/{{ $app }}.yaml:ro
      # 持久化日志（output-mode=file 时启用）
      # - ./logs:/app/logs

    ports:
      - "{{ $port }}:{{ $port }}"

    restart: always

    command: ["-c", "/app/configs/{{ $app }}.yaml"]

    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M

    extra_hosts:
      - "host.docker.internal:host-gateway"

    security_opt:
      - no-new-privileges:true
    read_only: false

    logging:
      driver: json-file
      options:
        max-size: "50m"
        max-file: "5"
        compress: "true"

    healthcheck:
      test: ["CMD-SHELL", "exit 0"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 15s

    # 同 Docker 网络部署时取消注释：
    # networks:
    #   - {{ $project }}_net
    # depends_on:
    #   postgres:
    #     condition: service_healthy
    #   redis:
    #     condition: service_healthy
    #   otel-collector:
    #     condition: service_healthy

# networks:
#   {{ $project }}_net:
#     external: true   # 与 docker-compose.env.yml 共用同一外部网络
