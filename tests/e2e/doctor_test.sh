#!/usr/bin/env bash
# tests/e2e/doctor_test.sh - lin doctor 端到端测试
set -euo pipefail

LIN_BIN=${LIN_BIN:-$PWD/_output/lin}
if [ ! -x "$LIN_BIN" ]; then
    echo "Building lin binary..."
    mkdir -p "$(dirname "$LIN_BIN")"
    go build -o "$LIN_BIN" ./cmd/lin
fi

echo "==> lin doctor --offline (basic, expect exit 0) ..."
"$LIN_BIN" doctor --offline || { echo "FAIL: doctor should not error in offline mode"; exit 1; }

echo "==> lin doctor --offline --report-format json ..."
JSON=$("$LIN_BIN" doctor --offline --report-format json)
echo "$JSON" | grep -q '"summary"' || { echo "FAIL: json output missing 'summary' field"; exit 1; }
echo "$JSON" | grep -q '"items"' || { echo "FAIL: json output missing 'items' field"; exit 1; }

echo "==> lin version ..."
"$LIN_BIN" version | grep -q "lin version 2.0.0-rc1" || { echo "FAIL: wrong version output"; exit 1; }

echo "==> lin version --short ..."
SHORT=$("$LIN_BIN" version --short)
[ "$SHORT" = "2.0.0-rc1" ] || { echo "FAIL: --short version expected '2.0.0-rc1', got '$SHORT'"; exit 1; }

echo "==> lin completion bash ..."
"$LIN_BIN" completion bash > /tmp/lin-completion-bash || { echo "FAIL: completion bash error"; exit 1; }
[ -s /tmp/lin-completion-bash ] || { echo "FAIL: empty completion script"; exit 1; }
rm -f /tmp/lin-completion-bash

echo "==> lin completion zsh ..."
"$LIN_BIN" completion zsh > /tmp/lin-completion-zsh || { echo "FAIL: completion zsh error"; exit 1; }
[ -s /tmp/lin-completion-zsh ] || { echo "FAIL: empty zsh completion script"; exit 1; }
rm -f /tmp/lin-completion-zsh

echo "==> lin doctor --offline --check go/version (subset check) ..."
"$LIN_BIN" doctor --offline --check go/version || { echo "FAIL: doctor go/version check failed"; exit 1; }

echo ""
echo "✔ DOCTOR E2E PASS"
