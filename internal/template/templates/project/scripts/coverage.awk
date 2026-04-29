#!/usr/bin/env awk

# coverage.awk
#
# 解析 `go tool cover -func` 的输出，针对 total 行做质量门限检查。
# 用法：
#     go tool cover -func=coverage.out | awk -v target=80 -f scripts/coverage.awk
#
# 行为：
#   - 透传所有输入行
#   - 命中 "total:" 行时打印实际覆盖率与目标覆盖率
#   - 实际 < target 时输出错误并以 exit 1 终止（CI 中触发失败）

{
  print $0
  if (match($0, /^total:/)) {
    sub(/%/, "", $NF);
    printf("test coverage is %s% (quality gate is %s%)\n", $NF, target)
    if (strtonum($NF) < target) {
      printf("test coverage does not meet expectations: %d%, please add test cases!\n", target)
      exit 1;
    }
  }
}
