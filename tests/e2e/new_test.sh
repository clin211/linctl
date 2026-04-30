#!/usr/bin/env bash
# tests/e2e/new_test.sh - lin new end-to-end test
set -euo pipefail

LIN_BIN=${LIN_BIN:-$PWD/_output/lin}
if [ ! -x "$LIN_BIN" ]; then
    echo "Building lin binary..."
    go build -o "$LIN_BIN" ./cmd/lin
fi

TMPDIR=$(mktemp -d -t lin-e2e-XXXXXX)
trap "rm -rf $TMPDIR" EXIT

cd "$TMPDIR"

echo "==> lin new --dry-run ..."
"$LIN_BIN" new myblog \
    --module github.com/test/myblog \
    --storage memory \
    --features healthz \
    --yes \
    --non-interactive \
    --dry-run

echo "==> lin new ..."
"$LIN_BIN" new myblog \
    --module github.com/test/myblog \
    --storage memory \
    --features healthz \
    --yes \
    --non-interactive

echo "==> verify directory layout ..."
test -f myblog/go.mod          || (echo "FAIL: missing go.mod"; exit 1)
test -f myblog/Makefile        || (echo "FAIL: missing Makefile"; exit 1)
test -f myblog/cmd/myblog/main.go \
                               || (echo "FAIL: missing main.go"; exit 1)
test -f myblog/internal/myblog/handler/handler.go \
                               || (echo "FAIL: missing handler.go"; exit 1)
test -f myblog/internal/myblog/biz/biz.go \
                               || (echo "FAIL: missing biz.go"; exit 1)
test -f myblog/internal/myblog/store/store.go \
                               || (echo "FAIL: missing store.go"; exit 1)
test -f myblog/internal/pkg/errno/register.go \
                               || (echo "FAIL: missing register.go"; exit 1)

echo "==> verify central files contain expected AST symbols (no anchor comments) ..."
grep -q "type IBiz interface" myblog/internal/myblog/biz/biz.go \
    || (echo "FAIL: biz.go missing IBiz interface"; exit 1)
grep -q "type IStore interface" myblog/internal/myblog/store/store.go \
    || (echo "FAIL: store.go missing IStore interface"; exit 1)
grep -q "func RegisterAll" myblog/internal/pkg/errno/register.go \
    || (echo "FAIL: register.go missing RegisterAll"; exit 1)

echo "==> verify there are NO inject-region anchors anywhere ..."
if grep -r "inject-region" myblog/ 2>/dev/null; then
    echo "FAIL: scaffolded project still contains inject-region anchor comments"
    exit 1
fi

echo "==> go mod tidy + go build ..."
cd myblog
go mod tidy
go build ./...

echo "✔ E2E PASS"
