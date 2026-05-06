#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

LIN_BIN=${LIN_BIN:-"$REPO_ROOT/_output/bin/linctl"}
case "$LIN_BIN" in /*) ;; *) LIN_BIN="$REPO_ROOT/$LIN_BIN" ;; esac

if [ ! -x "$LIN_BIN" ]; then
    mkdir -p "$(dirname "$LIN_BIN")"
    go build -o "$LIN_BIN" .
fi
LIN_BIN="$(cd "$(dirname "$LIN_BIN")" && pwd)/$(basename "$LIN_BIN")"

TMPDIR=$(mktemp -d -t lin-idempotent-XXXXXX)
trap "rm -rf $TMPDIR" EXIT

E2E_COMMON="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"
# shellcheck source=/dev/null
source "$E2E_COMMON"

cd "$TMPDIR"
linctl_e2e_link_linhub_replace "$TMPDIR" "$REPO_ROOT"

"$LIN_BIN" new myblog \
    --module github.com/test/myblog \
    --storage memory \
    --yes \
    --non-interactive
cd myblog

echo "==> first add ..."
"$LIN_BIN" add Post --yes --non-interactive

echo "==> snapshot file hashes ..."
HASHES=$(find internal pkg -name "*.go" -o -name "*.proto" | sort | xargs shasum -a 256 | sort)

echo "==> second add (must be idempotent) ..."
"$LIN_BIN" add Post --yes --non-interactive

echo "==> verify hashes unchanged ..."
HASHES_AFTER=$(find internal pkg -name "*.go" -o -name "*.proto" | sort | xargs shasum -a 256 | sort)
if [ "$HASHES" != "$HASHES_AFTER" ]; then
    echo "FAIL: file hashes changed after idempotent add"
    diff <(echo "$HASHES") <(echo "$HASHES_AFTER")
    exit 1
fi

echo "==> go mod tidy + build still works ..."
go mod tidy
go build ./...

echo "✔ IDEMPOTENT E2E PASS"
