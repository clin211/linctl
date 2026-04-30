#!/usr/bin/env bash
# boot.sh — quick local bootstrap for {{.ProjectName}}
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${PROJECT_ROOT}/_output/{{.AppName}}"
CONFIG="${PROJECT_ROOT}/configs/{{.AppName}}.yaml"

echo "==> Building {{.AppName}} ..."
(cd "${PROJECT_ROOT}" && go build -o "${BINARY}" ./cmd/{{.AppName}})

echo "==> Running {{.AppName}} ..."
exec "${BINARY}" -c "${CONFIG}"
