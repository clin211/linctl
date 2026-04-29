{{- $app := .Component.Name -}}
{{- $project := .Project.Metadata.Name -}}
{{- $registry := .Project.Spec.Defaults.Image.RegistryPrefix | default $project -}}
{{- $port := .Component.Port -}}
{{- if not $port }}{{ $port = 8080 }}{{ end }}
# 开发环境的组件级 Compose（仅 {{ $app }} 自身）。
#
# 使用方法：
#   cd build/docker/{{ $app }}
#   docker compose up -d --build
#
# 同一项目下的依赖（postgres / redis / otel）请用项目根的
# docker-compose.env.yml 启动。
services:
  {{ $app }}:
    image: {{ $registry }}/{{ $app }}:dev
    build:
      # 相对路径指向项目根（从 build/docker/{{ $app }}/ 向上三级）
      context: ../../..
      dockerfile: build/docker/{{ $app }}/Dockerfile
      args:
        OS: linux
        ARCH: amd64
        GOPROXY: https://goproxy.cn,direct
    container_name: {{ $app }}
    hostname: {{ $app }}

    environment:
      - TZ=Asia/Shanghai

    # 配置文件挂载（相对项目根）
    volumes:
      - ../../../configs/{{ $app }}.docker.yaml:/app/configs/{{ $app }}.yaml:ro
      - /etc/localtime:/etc/localtime:ro

    ports:
      - "{{ $port }}:{{ $port }}"

    restart: unless-stopped

    command: ["-c", "/app/configs/{{ $app }}.yaml"]

    # 让容器能访问宿主机服务（Linux 上 host.docker.internal 默认不可达）
    extra_hosts:
      - "host.docker.internal:host-gateway"

    security_opt:
      - no-new-privileges:true

    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

    # scratch 镜像无 shell，无法在容器内做命令式健康检查；
    # 生产建议用外部 LB / 监控系统的 HTTP 探活。
    healthcheck:
      test: ["CMD-SHELL", "exit 0"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
