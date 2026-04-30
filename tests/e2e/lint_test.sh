#!/usr/bin/env bash
# tests/e2e/lint_test.sh - lin lint end-to-end test
set -euo pipefail

LIN_BIN=${LIN_BIN:-$PWD/_output/lin}
if [ ! -x "$LIN_BIN" ]; then
    echo "Building lin binary..."
    mkdir -p "$(dirname "$LIN_BIN")"
    go build -o "$LIN_BIN" ./cmd/lin
fi

TMPDIR=$(mktemp -d -t lin-lint-XXXXXX)
trap "rm -rf $TMPDIR" EXIT
cd "$TMPDIR"

echo "==> lin new myblog ..."
"$LIN_BIN" new myblog \
    --module github.com/test/myblog \
    --storage memory \
    --yes \
    --non-interactive

cd myblog

echo "==> lin add Post ..."
"$LIN_BIN" add Post --yes --non-interactive

echo "==> lin lint (clean project, expect pass) ..."
"$LIN_BIN" lint || { echo "FAIL: lint reported issues on clean project"; exit 1; }

echo "==> simulate registration drift: create a stray biz/v1/orphan/ ..."
mkdir -p internal/myblog/biz/v1/orphan
cat > internal/myblog/biz/v1/orphan/orphan.go <<'GO'
package orphan

func Orphan() {}
GO

echo "==> lin lint (must report register/biz-impl warning) ..."
OUT=$("$LIN_BIN" lint --report-format text || true)
echo "$OUT" | grep -q "register/biz-impl" || {
    echo "FAIL: lint did not surface register/biz-impl warning"
    echo "$OUT"
    exit 1
}
echo "  [ok] lint detected stray resource directory"

echo "==> lin lint --report-format json ..."
JSON=$("$LIN_BIN" lint --report-format json || true)
echo "$JSON" | grep -q '"summary"' || { echo "FAIL: json report missing 'summary' field"; exit 1; }
echo "$JSON" | grep -q '"items"' || { echo "FAIL: json report missing 'items' field"; exit 1; }

echo ""
echo "✔ LINT E2E PASS"
