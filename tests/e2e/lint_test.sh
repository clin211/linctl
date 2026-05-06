#!/usr/bin/env bash
# tests/e2e/lint_test.sh - linctl lint end-to-end test
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

LIN_BIN=${LIN_BIN:-"$REPO_ROOT/_output/bin/linctl"}
case "$LIN_BIN" in /*) ;; *) LIN_BIN="$REPO_ROOT/$LIN_BIN" ;; esac

if [ ! -x "$LIN_BIN" ]; then
    echo "Building linctl binary..."
    mkdir -p "$(dirname "$LIN_BIN")"
    go build -o "$LIN_BIN" .
fi
LIN_BIN="$(cd "$(dirname "$LIN_BIN")" && pwd)/$(basename "$LIN_BIN")"

TMPDIR=$(mktemp -d -t lin-lint-XXXXXX)
trap "rm -rf $TMPDIR" EXIT

E2E_COMMON="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"
# shellcheck source=/dev/null
source "$E2E_COMMON"

cd "$TMPDIR"
linctl_e2e_link_linhub_replace "$TMPDIR" "$REPO_ROOT"

echo "==> linctl new myblog ..."
"$LIN_BIN" new myblog \
    --module github.com/test/myblog \
    --storage memory \
    --yes \
    --non-interactive

cd myblog

echo "==> linctl add Post ..."
"$LIN_BIN" add Post --yes --non-interactive

echo "==> linctl lint (clean project, expect pass) ..."
"$LIN_BIN" lint || { echo "FAIL: lint reported issues on clean project"; exit 1; }

echo "==> simulate registration drift: create a stray biz/v1/orphan/ ..."
mkdir -p internal/myblog/biz/v1/orphan
cat > internal/myblog/biz/v1/orphan/orphan.go <<'GO'
package orphan

func Orphan() {}
GO

echo "==> linctl lint (must report register/biz-impl warning) ..."
OUT=$("$LIN_BIN" lint --report-format text || true)
echo "$OUT" | grep -q "register/biz-impl" || {
    echo "FAIL: lint did not surface register/biz-impl warning"
    echo "$OUT"
    exit 1
}
echo "  [ok] lint detected stray resource directory"

echo "==> linctl lint --report-format json ..."
JSON=$("$LIN_BIN" lint --report-format json || true)
echo "$JSON" | grep -q '"summary"' || { echo "FAIL: json report missing 'summary' field"; exit 1; }
echo "$JSON" | grep -q '"items"' || { echo "FAIL: json report missing 'items' field"; exit 1; }

echo ""
echo "✔ LINT E2E PASS"
