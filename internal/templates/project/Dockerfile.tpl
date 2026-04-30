# syntax=docker/dockerfile:1.7

ARG GO_VERSION={{.GoVersion}}
ARG OS=linux
ARG ARCH=amd64

# -- Builder stage --
FROM golang:${GO_VERSION} AS builder
ENV GOTOOLCHAIN=auto
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /workspace

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /app/{{.AppName}} {{.Module}}/cmd/{{.AppName}}

# -- Runtime stage --
FROM scratch AS runtime
WORKDIR /app
COPY --from=builder /app/{{.AppName}} /app/{{.AppName}}
USER 10001
EXPOSE 8080

ENTRYPOINT ["/app/{{.AppName}}"]
CMD ["-c", "/app/configs/{{.AppName}}.yaml"]
