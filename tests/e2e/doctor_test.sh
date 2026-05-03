#!/usr/bin/env bash
# tests/e2e/doctor_test.sh - linctl doctor 端到端测试
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

LIN_BIN=${LIN_BIN:-"$REPO_ROOT/_output/bin/linctl"}
case "$LIN_BIN" in /*) ;; *) LIN_BIN="$REPO_ROOT/$LIN_BIN" ;; esac

if [ ! -x "$LIN_BIN" ]; then
    echo "Building linctl binary..."
    mkdir -p "$(dirname "$LIN_BIN")"
    go build -o "$LIN_BIN" ./cmd/linctl
fi
LIN_BIN="$(cd "$(dirname "$LIN_BIN")" && pwd)/$(basename "$LIN_BIN")"

echo "==> linctl doctor --offline (basic, expect exit 0) ..."
"$LIN_BIN" doctor --offline || { echo "FAIL: doctor should not error in offline mode"; exit 1; }

echo "==> linctl doctor --offline --report-format json ..."
JSON=$("$LIN_BIN" doctor --offline --report-format json)
echo "$JSON" | grep -q '"summary"' || { echo "FAIL: json output missing 'summary' field"; exit 1; }
echo "$JSON" | grep -q '"items"' || { echo "FAIL: json output missing 'items' field"; exit 1; }

echo "==> linctl version ..."
VER_LINE=$("$LIN_BIN" version | head -1)
echo "$VER_LINE" | grep -qE '^linctl version .+' || {
    echo "FAIL: expected first line like 'linctl version <semver>', got: $VER_LINE"
    exit 1
}

echo "==> linctl version --short ..."
SHORT=$("$LIN_BIN" version --short)
[ -n "$SHORT" ] || { echo "FAIL: --short version is empty"; exit 1; }

echo "==> linctl completion bash ..."
"$LIN_BIN" completion bash > /tmp/linctl-completion-bash || { echo "FAIL: completion bash error"; exit 1; }
[ -s /tmp/linctl-completion-bash ] || { echo "FAIL: empty completion script"; exit 1; }
rm -f /tmp/linctl-completion-bash

echo "==> linctl completion zsh ..."
"$LIN_BIN" completion zsh > /tmp/linctl-completion-zsh || { echo "FAIL: completion zsh error"; exit 1; }
[ -s /tmp/linctl-completion-zsh ] || { echo "FAIL: empty zsh completion script"; exit 1; }
rm -f /tmp/linctl-completion-zsh

echo "==> linctl doctor --offline --check go/version (subset check) ..."
"$LIN_BIN" doctor --offline --check go/version || { echo "FAIL: doctor go/version check failed"; exit 1; }

echo ""
echo "✔ DOCTOR E2E PASS"
