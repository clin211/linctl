PROJ_ROOT_DIR := $(strip $(abspath $(shell pwd -P)))
OUTPUT_DIR    := $(PROJ_ROOT_DIR)/_output
ROOT_PACKAGE  := {{.Module}}
APIROOT       := $(PROJ_ROOT_DIR)/pkg/api

# ---------------------------------------------------------------------------
# 版本信息

IS_GIT_REPO := $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifeq ($(IS_GIT_REPO),)
    GIT_TREE_STATE := not_a_git_repo
    VERSION        := v0.0.0
    GIT_COMMIT     := unknown
else
    ifeq ($(origin VERSION), undefined)
        VERSION := $(shell git describe --tags --abbrev=0 --match='v*' 2>/dev/null || echo v0.0.0)
    endif
    GIT_TREE_STATE := $(shell git status --porcelain 2>/dev/null | grep -q . && echo dirty || echo clean)
    GIT_COMMIT     := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
endif

ifeq ($(GOOS),windows)
    GO_OUT_EXT := .exe
endif

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# ---------------------------------------------------------------------------
.DEFAULT_GOAL := all

.PHONY: all
all: tidy format build ## 构建项目（默认目标）。

.PHONY: build
build: ## 编译 Go 二进制。
	@echo "===========> Building {{.AppName}}"
	@mkdir -p $(OUTPUT_DIR)
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-o $(OUTPUT_DIR)/{{.AppName}}$(GO_OUT_EXT) \
		$(ROOT_PACKAGE)/cmd/{{.AppName}}

.PHONY: tidy
tidy: ## 同步依赖（go mod tidy）。
	@go mod tidy

.PHONY: format
format: ## 格式化 Go 源文件。
	@gofmt -s -w ./

.PHONY: test
test: ## 运行单元测试。
	@go test -race -cover -timeout=5m ./...

.PHONY: lint
lint: ## 运行 golangci-lint。
	@golangci-lint run ./...

.PHONY: clean
clean: ## 清理构建产物。
	@-rm -vrf $(OUTPUT_DIR)

.PHONY: deps
deps: ## 安装构建工具。
	@echo "===========> Installing dependent tools"
	@go install github.com/google/wire/cmd/wire@latest

.PHONY: protoc
protoc: ## 编译 Protobuf 文件。
	@echo "===========> Generating protobuf files"
	@protoc \
		--proto_path=$(APIROOT) \
		--go_out=paths=source_relative:$(APIROOT) \
		--go-grpc_out=paths=source_relative:$(APIROOT) \
		$(shell find $(APIROOT)/* -name "*.proto" 2>/dev/null)

help: Makefile ## 显示所有可用 target。
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<TARGET>\033[0m\n\n\033[35mTargets:\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
