{{- $app := .Component.Name -}}
# syntax=docker/dockerfile:1.7
#
# 组件级 Dockerfile：仅构建并发布 {{ $app }} 这一个二进制。
#
# - 与项目根 Dockerfile 的区别：
#     根 Dockerfile：通用，通过 ARG BIN 切换构建目标
#     本文件：专属 {{ $app }}，BIN 已固化，便于在 CI 中直接 docker build
#
# - 与 docker-compose.yml 的搭配：
#     build:
#       context: ../../..
#       dockerfile: build/docker/{{ $app }}/Dockerfile

ARG BUILDER_IMAGE=golang:1.25.3
ARG RUNTIME_IMAGE=gcr.io/distroless/base-debian12:nonroot

# 1) Builder
FROM ${BUILDER_IMAGE} AS builder
ENV GOTOOLCHAIN=auto
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=sum.golang.org
ARG OS=linux
ARG ARCH=amd64
WORKDIR /workspace

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} GO111MODULE=on \
    GOCACHE=/root/.cache/go-build GOMODCACHE=/go/pkg/mod

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build BINS={{ $app }}

RUN mkdir -p /app && cp -v _output/platforms/${OS}/${ARCH}/{{ $app }} /app/{{ $app }}

# 2) Runtime（scratch；如需 distroless / alpine 见根 Dockerfile 的备选）
FROM scratch AS runtime
WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/{{ $app }} /app/{{ $app }}

USER 10001
{{- if .Component.Port }}
EXPOSE {{ .Component.Port }}
{{- else }}
EXPOSE 8080
{{- end }}

ENTRYPOINT ["/app/{{ $app }}"]
CMD ["-c", "/app/configs/{{ $app }}.yaml"]
