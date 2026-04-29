{{- $app := (index .Project.Spec.Components 0).Name -}}
# syntax=docker/dockerfile:1.7
#
# 顶层 Dockerfile：项目根目录的多阶段构建定义。
#
# - builder 阶段：基于 golang:1.25.3 编译指定二进制（默认 {{ $app }}）
# - runtime 阶段：默认 scratch（最小镜像），如需运行时调试可改为
#   gcr.io/distroless/base-debian12:nonroot 或 alpine
#
# CI 中可通过 `--build-arg BIN=other-app` 切换构建目标，无需改 Dockerfile。

# 0) Build args (overridable in CI)
ARG BUILDER_IMAGE=golang:1.25.3
ARG RUNTIME_IMAGE=gcr.io/distroless/base-debian12:nonroot
ARG UID=65532
ARG GID=65532
ARG BIN={{ $app }}

# 1) Builder stage
FROM ${BUILDER_IMAGE} AS builder
ENV GOTOOLCHAIN=auto
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=sum.golang.org
ARG OS=linux
ARG ARCH=amd64
ARG BIN
WORKDIR /workspace

# 利用 docker layer cache：先拷 go.mod / go.sum 下载依赖
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# 拷贝源码并构建
COPY . .

ENV CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} GO111MODULE=on \
    GOCACHE=/root/.cache/go-build GOMODCACHE=/go/pkg/mod

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build BINS=${BIN}

# 把 Makefile 的产物复制到一个固定路径，方便 runtime 阶段 COPY
RUN mkdir -p /app && cp -v _output/platforms/${OS}/${ARCH}/${BIN} /app/${BIN}

# 2) Runtime stage：默认 scratch（最小镜像）
#
# 提示：scratch 镜像没有 shell / curl / busybox，无法在容器内做健康检查；
# 如需 exec 进入或脚本探活，请改为：
#     FROM ${RUNTIME_IMAGE} AS runtime
# 并启用上方的 distroless / alpine 选项。
FROM scratch AS runtime
ARG BIN
WORKDIR /app

# 拷贝 CA 证书（HTTPS 出站需要；不需要可注释掉）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# 仅复制二进制
COPY --from=builder /app/${BIN} /app/{{ $app }}

# 使用非 root（数值 UID 即可）
USER 10001
{{- if (index .Project.Spec.Components 0).Port }}
EXPOSE {{ (index .Project.Spec.Components 0).Port }}
{{- else }}
EXPOSE 8080
{{- end }}

ENTRYPOINT ["/app/{{ $app }}"]
# 默认通过 Compose 挂载配置文件
CMD ["-c", "/app/configs/{{ $app }}.yaml"]
