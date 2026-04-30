services:
  {{.AppName}}:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: {{.AppName}}
    environment:
      - TZ=Asia/Shanghai
    volumes:
      - ./configs/{{.AppName}}.yaml:/app/configs/{{.AppName}}.yaml:ro
    ports:
      - "8080:8080"
    restart: unless-stopped
    extra_hosts:
      - "host.docker.internal:host-gateway"
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
{{- if ne .Storage "memory"}}

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
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  postgres_data:
{{- end}}
