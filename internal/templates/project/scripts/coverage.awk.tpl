# coverage.awk —— 比对 go test -cover 的输出与目标阈值，未达到则非零退出。
# 用法：go tool cover -func=coverage.out | awk -v target=60 -f scripts/coverage.awk
END {
    pct = $NF
    gsub("%", "", pct)
    if (pct + 0 < target + 0) {
        printf "Coverage %s%% is below target %s%%\n", pct, target
        exit 1
    }
    printf "Coverage %s%% meets target %s%%\n", pct, target
}
