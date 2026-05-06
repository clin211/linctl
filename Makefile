SHELL := /bin/bash

# ===== 元数据 =====
APP        := linctl
PKG        := github.com/clin211/linctl
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# ===== 工具版本 pin（详见 docs/10-tech-stack.md §10.7）=====
GOFUMPT_VERSION    := v0.7.0
GOLANGCI_VERSION   := v1.59.1
MOCKGEN_VERSION    := v0.4.0
GOTESTSUM_VERSION  := v1.12.0

# ===== 路径 =====
OUTPUT_DIR := _output
BIN_DIR    := $(OUTPUT_DIR)/bin
COVER_DIR  := $(OUTPUT_DIR)/coverage

# ===== ldflags =====
# Inject into internal/version package (used by version.Get())
# Also inject into internal/cli package (version.go uses package-level vars)
LDFLAGS := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.BuildDate=$(BUILD_DATE) \
	-X $(PKG)/internal/cli.version=$(VERSION) \
	-X $(PKG)/internal/cli.commit=$(COMMIT) \
	-X $(PKG)/internal/cli.buildTime=$(BUILD_DATE)

# ===== 默认目标 =====
.PHONY: all
all: lint test build

# ===== 构建 =====
.PHONY: build
build: $(BIN_DIR)/$(APP)

$(BIN_DIR)/$(APP): $(shell find . -name '*.go' -not -path './_output/*' 2>/dev/null)
	@mkdir -p $(BIN_DIR)
	@echo "==> building $(APP) ($(VERSION))"
	@go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) .

.PHONY: build-otel
build-otel:
	@mkdir -p $(BIN_DIR)
	@echo "==> building $(APP)-otel (with OpenTelemetry)"
	@go build -trimpath -tags otel -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-otel .

.PHONY: install
install: build
	@cp $(BIN_DIR)/$(APP) $(GOPATH)/bin/$(APP)

# ===== 测试 =====
.PHONY: test
test:
	@mkdir -p $(COVER_DIR)
	@echo "==> running unit tests"
	@go test -race -cover -coverprofile=$(COVER_DIR)/coverage.out ./...
	@go tool cover -func=$(COVER_DIR)/coverage.out | tail -1

.PHONY: test-short
test-short:
	@go test -short ./...

.PHONY: cover
cover: test
	@go tool cover -html=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/coverage.html
	@echo "==> coverage report: $(COVER_DIR)/coverage.html"

# ===== 静态检查 =====
.PHONY: lint
lint:
	@echo "==> running golangci-lint"
	@golangci-lint run --timeout=5m ./...

.PHONY: fmt
fmt:
	@echo "==> formatting code with gofumpt"
	@gofumpt -l -w .

.PHONY: fmt-check
fmt-check:
	@gofumpt -l . | tee /tmp/gofumpt.out
	@test ! -s /tmp/gofumpt.out

.PHONY: vet
vet:
	@go vet ./...

# ===== 工具安装 =====
.PHONY: tools
tools:
	@echo "==> installing pinned dev tools"
	@go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
	@go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	@go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_VERSION)

# ===== E2E 测试 =====
.PHONY: e2e
e2e: build
	@LIN_BIN=$(BIN_DIR)/$(APP) bash tests/e2e/new_test.sh
	@LIN_BIN=$(BIN_DIR)/$(APP) bash tests/e2e/add_test.sh
	@LIN_BIN=$(BIN_DIR)/$(APP) bash tests/e2e/add_idempotent_test.sh
	@LIN_BIN=$(BIN_DIR)/$(APP) bash tests/e2e/lint_test.sh
	@LIN_BIN=$(BIN_DIR)/$(APP) bash tests/e2e/doctor_test.sh

# ===== 清理 =====
.PHONY: clean
clean:
	@rm -rf $(OUTPUT_DIR)

# ===== 帮助 =====
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build        - Build $(APP) binary into $(BIN_DIR)"
	@echo "  make build-otel   - Build $(APP)-otel with OpenTelemetry support"
	@echo "  make test         - Run unit tests with race detector + coverage"
	@echo "  make cover        - Generate HTML coverage report"
	@echo "  make e2e          - Run all E2E tests (new / add / add_idempotent / lint / doctor)"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make fmt          - Format code with gofumpt"
	@echo "  make fmt-check    - Verify code is gofumpt-clean"
	@echo "  make vet          - Run go vet"
	@echo "  make tools        - Install pinned dev tools"
	@echo "  make clean        - Remove $(OUTPUT_DIR)"
	@echo "  make all          - lint + test + build (default)"
