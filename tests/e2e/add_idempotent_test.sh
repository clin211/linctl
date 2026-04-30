#!/usr/bin/env bash
set -euo pipefail
LIN_BIN=${LIN_BIN:-$PWD/_output/lin}
[ -x "$LIN_BIN" ] || go build -o "$LIN_BIN" ./cmd/lin

TMPDIR=$(mktemp -d -t lin-idempotent-XXXXXX)
trap "rm -rf $TMPDIR" EXIT
cd "$TMPDIR"

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
