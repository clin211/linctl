{{- $app := (index .Project.Spec.Components 0).Name -}}
{{- $port := (index .Project.Spec.Components 0).Port -}}
{{- if not $port }}{{ $port = 8080 }}{{ end }}
#!/bin/bash
#
# startup-test.sh —— 通用 REST 资源烟测脚本。
#
# 用法：
#   ./scripts/startup-test.sh <resource> {create|get|list} [args]
#
# 示例：
#   ./scripts/startup-test.sh post create '{"title":"hello"}'
#   ./scripts/startup-test.sh post get 1
#   ./scripts/startup-test.sh post list
#
# 默认假设 API 监听在 127.0.0.1:{{ $port }}（与 {{ $app }} 默认配置匹配）；
# 如需调整，覆盖环境变量 API_BASE。

set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:{{ $port }}/v1}"
CT="Content-Type: application/json"

res="${1:-}"
action="${2:-}"
arg="${3:-}"

if [ -z "$res" ] || [ -z "$action" ]; then
  echo "Usage: $0 <resource> {create|get|list} [args]"
  exit 1
fi

url="$API_BASE/$res"

# Common helper: 发起请求并提取 X-Trace-Id（OTel 串联）
call_api() {
  local method="$1"
  local full_url="$2"
  local data="$3"

  response=$(curl -s -D - -X "$method" "$full_url" -H "$CT" -d "$data")
  trace_id=$(echo "$response" | grep -i '^X-Trace-Id:' | awk '{print $2}' | tr -d '\r')
  body=$(echo "$response" | sed -n '/^\r$/,$p' | tail -n +2)

  echo "X-Trace-Id: ${trace_id:-<none>}"
  echo "-----------------------------"
  echo "$body" | jq .
}

case "$action" in
  create)
    if [ -z "$arg" ]; then
      echo "Example: $0 $res create '{\"name\":\"my-job\"}'"
      exit 1
    fi
    call_api POST "$url" "$arg"
    ;;
  get)
    if [ -z "$arg" ]; then
      echo "Example: $0 $res get 123"
      exit 1
    fi
    call_api GET "$url/$arg" ""
    ;;
  list)
    call_api GET "$url" ""
    ;;
  *)
    echo "Unsupported action: $action (supported: create|get|list)"
    exit 1
    ;;
esac
