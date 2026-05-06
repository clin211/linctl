# ==============================================================================
# 全局变量

COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
# 项目根目录
PROJ_ROOT_DIR := $(strip $(abspath $(shell cd $(COMMON_SELF_DIR)/ && pwd -P)))
# 构建产物与临时文件目录
OUTPUT_DIR := $(PROJ_ROOT_DIR)/_output
# Protobuf 文件路径
APIROOT=$(PROJ_ROOT_DIR)/pkg/api
ROOT_PACKAGE={{.Module}}
DOCKERFILE_DIR=$(PROJ_ROOT_DIR)/build/docker
REGISTRY_PREFIX ?= {{.AppName}}

# 单元测试覆盖率阈值（演示值；生产环境建议设置为 60+）。
COVERAGE := 1

# ==============================================================================
# 版本信息

# 版本变量将通过 -ldflags -X 注入到 linhub/version 包。
VERSION_PACKAGE=github.com/clin211/linhub/version

# 检查当前目录是否是 Git 仓库
IS_GIT_REPO := $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifeq ($(IS_GIT_REPO),)
    # 非 Git 仓库使用回退值
    GIT_TREE_STATE := "not_a_git_repo"
    VERSION := v0.0.0
    GIT_COMMIT := "unknown"
else
    # 若未显式指定，则从 git tag 获取语义化版本
    ifeq ($(origin VERSION), undefined)
        VERSION := $(shell git describe --tags --abbrev=0 --match='v*' 2>/dev/null || echo v0.0.0)
    endif

    # 检查工作区是否干净
    GIT_TREE_STATE := "dirty"
    ifeq (, $(shell git status --porcelain 2>/dev/null))
        GIT_TREE_STATE := "clean"
    endif

    GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
endif

GO_LDFLAGS += \
    -X $(VERSION_PACKAGE).gitVersion=$(VERSION) \
    -X $(VERSION_PACKAGE).gitCommit=$(GIT_COMMIT) \
    -X $(VERSION_PACKAGE).gitTreeState=$(GIT_TREE_STATE) \
    -X $(VERSION_PACKAGE).buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
    -X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn

ifeq ($(GOOS),windows)
    GO_OUT_EXT := .exe
endif

GO_BUILD_FLAGS += -ldflags "$(GO_LDFLAGS)"
# 仅构建主二进制，排除工具命令（如 gen-gorm-model）
COMMANDS ?= $(filter-out $(PROJ_ROOT_DIR)/cmd/gen-gorm-model, $(filter-out %.md, $(wildcard $(PROJ_ROOT_DIR)/cmd/*)))
BINS ?= $(foreach cmd,${COMMANDS},$(notdir $(cmd)))
IMAGES ?= $(filter-out tools, $(foreach dir, $(COMMANDS), $(notdir $(if $(wildcard $(dir)/*.go), $(dir),))))

# 支持的目标平台
PLATFORMS ?= darwin_amd64 darwin_arm64 windows_amd64 windows_arm64 linux_amd64 linux_arm64

# 默认平台（取自当前主机环境）
ifeq ($(origin PLATFORM), undefined)
    ifeq ($(origin GOOS), undefined)
        GOOS := $(shell go env GOOS)
    endif
    ifeq ($(origin GOARCH), undefined)
        GOARCH := $(shell go env GOARCH)
    endif
    PLATFORM := $(GOOS)_$(GOARCH)
    # 构建镜像默认使用 linux 作为操作系统
    IMAGE_PLAT := linux_$(GOARCH)
else
    GOOS := $(word 1, $(subst _, ,$(PLATFORM)))
    GOARCH := $(word 2, $(subst _, ,$(PLATFORM)))
    IMAGE_PLAT := $(PLATFORM)
endif

# ==============================================================================
# 默认 target

.DEFAULT_GOAL := all

.PHONY: all
all: tidy format build ## 构建项目（默认 target，依赖 tidy / format / build）。

# ==============================================================================
# 使用说明

define USAGE_OPTIONS

Options:
  BINS             需要构建的二进制名（默认 cmd/ 下所有可执行项目）。
                   配合 make build 使用：make build BINS=<binaryName>
  IMAGES           需要构建的镜像名（默认 cmd/ 下所有可执行项目）。
                   配合 make image 使用：make image IMAGES=<binaryName>
  VERSION          注入二进制的版本号。
  V                设置为 1 开启 verbose 构建输出，默认 0。
endef
export USAGE_OPTIONS

# ==============================================================================
# 构建相关 target

.PHONY: build.multiarch
build.multiarch: $(foreach p,$(PLATFORMS),$(addprefix build., $(addprefix $(p)., $(BINS)))) ## 多架构构建所有二进制。

.PHONY: build
build: $(addprefix build., $(addprefix $(PLATFORM)., $(BINS))) ## 构建当前平台所有二进制。

build.%: ## 编译 Go 源码。
	$(eval COMMAND := $(word 2,$(subst ., ,$*)))
	$(eval PLATFORM := $(word 1,$(subst ., ,$*)))
	$(eval OS := $(word 1,$(subst _, ,$(PLATFORM))))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLATFORM))))
	@echo "===========> Building binary $(COMMAND) $(VERSION) for $(OS) $(ARCH)"
	@mkdir -p $(OUTPUT_DIR)/platforms/$(OS)/$(ARCH)
	@CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/platforms/$(OS)/$(ARCH)/$(COMMAND)$(GO_OUT_EXT) \
		$(ROOT_PACKAGE)/cmd/$(COMMAND)

.PHONY: image
image: $(addprefix image., $(addprefix $(PLATFORM)., $(IMAGES))) ## 构建当前平台的所有 Docker 镜像。

image.%: build.% ## 构建指定的 Docker 镜像。
	$(eval IMAGE := $(word 2,$(subst ., ,$*)))
	$(eval IMAGE_TAG := $(subst +,-,$(VERSION)))
	@echo "===========> Building docker image $(IMAGE) $(IMAGE_TAG) for $(IMAGE_PLAT)"
	$(eval PLATFORM := $(word 1,$(subst ., ,$*)))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLATFORM))))
	$(eval OS := $(word 1,$(subst _, ,$(PLATFORM))))
	$(eval DOCKERFILE := Dockerfile)
	@DOCKER_BUILDKIT=1 docker build \
		--build-arg OS=$(OS) \
		--build-arg ARCH=$(ARCH) \
		--file $(DOCKERFILE_DIR)/$(IMAGE)/$(DOCKERFILE) \
		--tag $(REGISTRY_PREFIX)/$(IMAGE):$(IMAGE_TAG) \
		$(PROJ_ROOT_DIR)

# ==============================================================================
# 代码质量与依赖

.PHONY: format
format: ## 用 gofmt 格式化源码（-s -w）。
	@gofmt -s -w ./

.PHONY: add-copyright
add-copyright: ## 给所有源文件添加版权头（跳过 third_party / vendor / _output）。
	@addlicense -v -f $(PROJ_ROOT_DIR)/scripts/boilerplate.txt $(PROJ_ROOT_DIR) --skip-dirs=third_party,vendor,$(OUTPUT_DIR)

.PHONY: tidy
tidy: ## 同步依赖并更新 go.mod / go.sum（go mod tidy）。
	@go mod tidy

.PHONY: clean
clean: ## 清理构建产物与临时文件（_output/）。
	@-rm -vrf $(OUTPUT_DIR)

.PHONY: protoc
protoc: ## 编译 Protobuf 文件。
	@echo "===========> Generate protobuf files"
	@mkdir -p $(PROJ_ROOT_DIR)/api/openapi
	@protoc \
		--proto_path=$(APIROOT) \
		--proto_path=$(PROJ_ROOT_DIR)/third_party/protobuf \
		--go_out=paths=source_relative:$(APIROOT) \
		--go-grpc_out=paths=source_relative:$(APIROOT) \
		--grpc-gateway_out=allow_delete_body=true,paths=source_relative:$(APIROOT) \
		--openapiv2_out=$(PROJ_ROOT_DIR)/api/openapi \
		--openapiv2_opt=allow_delete_body=true,logtostderr=true \
		--defaults_out=paths=source_relative:$(APIROOT) \
		$(shell find $(APIROOT)/* -name *.proto 2>/dev/null)
	@find $(APIROOT)/* -name "*.pb.go" -exec protoc-go-inject-tag -input={} \; 2>/dev/null || true

.PHONY: protoc.%
protoc.%: ## 编译指定子模块的 Protobuf 文件。
	@echo "===========> Generate protobuf files"
	@mkdir -p $(PROJ_ROOT_DIR)/api/openapi
	@protoc \
		--proto_path=$(APIROOT) \
		--proto_path=$(PROJ_ROOT_DIR)/third_party/protobuf \
		--go_out=paths=source_relative:$(APIROOT) \
		--go-grpc_out=paths=source_relative:$(APIROOT) \
		--openapiv2_out=$(PROJ_ROOT_DIR)/api/openapi \
		--openapiv2_opt=allow_delete_body=true,logtostderr=true \
		--defaults_out=paths=source_relative:$(APIROOT) \
		$(shell find $(APIROOT)/$* -name *.proto)
	@find $(APIROOT)/$* -name "*.pb.go" -exec protoc-go-inject-tag -input={} \;

.PHONY: test
test: ## 运行单元测试（-race -cover -shuffle -short）。
	@echo "===========> Running unit tests"
	@mkdir -p $(OUTPUT_DIR)
	@go test -race -cover \
		-coverprofile=$(OUTPUT_DIR)/coverage.out \
		-timeout=10m -shuffle=on -short \
		-v `go list ./... | grep -Ev 'tools|vendor|third_party'`

.PHONY: cover
cover: test ## 运行测试并校验覆盖率阈值（COVERAGE%）。
	@echo "===========> Running code coverage tests"
	@go tool cover -func=$(OUTPUT_DIR)/coverage.out | awk -v target=$(COVERAGE) -f $(PROJ_ROOT_DIR)/scripts/coverage.awk

.PHONY: lint
lint: ## 用 golangci-lint 做静态检查（基于 .golangci.yaml）。
	@echo "===========> Running golangci to lint source codes"
	@golangci-lint run -c $(PROJ_ROOT_DIR)/.golangci.yaml $(PROJ_ROOT_DIR)/...

.PHONY: deps
deps: ## 安装构建与代码生成所需工具。
	@echo "===========> Installing dependent tools, may take a while ..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@go install github.com/onexstack/protoc-gen-defaults@latest
	@go install github.com/favadi/protoc-go-inject-tag@latest
	@go install github.com/google/wire/cmd/wire@latest
	@go install github.com/onexstack/addlicense@v0.0.3
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | \
			sh -s -- -b "$$(go env GOPATH)/bin" v2.4.0 ; \
	fi

.PHONY: generate
generate: ## 用 go:generate 指令为所有包生成代码（递归 ./...）。
	@go generate ./...

help: Makefile ## 显示所有可用的 target 与用法。
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<TARGETS> <OPTIONS>\033[0m\n\n\033[35mTargets:\033[0m\n"} /^[0-9A-Za-z._-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 } /^##@/{ printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' Makefile
	@echo "$$USAGE_OPTIONS"
