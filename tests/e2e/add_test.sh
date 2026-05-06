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

TMPDIR=$(mktemp -d -t lin-add-XXXXXX)
trap "rm -rf $TMPDIR" EXIT

E2E_COMMON="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"
# shellcheck source=/dev/null
source "$E2E_COMMON"

cd "$TMPDIR"
linctl_e2e_link_linhub_replace "$TMPDIR" "$REPO_ROOT"

echo "==> bootstrap project ..."
"$LIN_BIN" new myblog \
    --module github.com/test/myblog \
    --storage memory \
    --features healthz \
    --yes \
    --non-interactive
cd myblog

echo "==> linctl add Post ..."
"$LIN_BIN" add Post --yes --non-interactive

echo "==> verify Post resource files (13) ..."
for f in \
    internal/myblog/handler/post.go \
    internal/myblog/biz/v1/post/post.go \
    internal/myblog/biz/v1/post/create.go \
    internal/myblog/biz/v1/post/update.go \
    internal/myblog/biz/v1/post/delete.go \
    internal/myblog/biz/v1/post/get.go \
    internal/myblog/biz/v1/post/list.go \
    internal/myblog/store/post.go \
    internal/myblog/model/post.gen.go \
    internal/myblog/pkg/conversion/post.go \
    internal/myblog/pkg/validation/post.go \
    internal/pkg/errno/post.go \
    pkg/api/myblog/v1/post.proto; do
    test -f "$f" || (echo "MISSING: $f"; exit 1)
done

echo "==> verify AST injections (4) ..."
grep -q "PostV1() postv1.PostBiz" internal/myblog/biz/biz.go \
    || (echo "biz.go missing PostV1"; exit 1)
grep -q "Posts() PostStore" internal/myblog/store/store.go \
    || (echo "store.go missing Posts"; exit 1)
grep -q 'import "post.proto"' pkg/api/myblog/v1/myblog.proto \
    || (echo "myblog.proto missing post import"; exit 1)
grep -q "RegisterErrors(PostErrors()...)" internal/pkg/errno/register.go \
    || (echo "register.go missing PostErrors call"; exit 1)

echo "==> go mod tidy + go build ..."
go mod tidy
go build ./...

echo "✔ ADD E2E PASS"
