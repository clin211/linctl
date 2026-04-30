// Package main 是 lin v2 的二进制入口。
//
// 设计来源：lin/docs/features/01-architecture-blueprint.md §3 「L0 入口层」。
//
// 职责（≤ 50 行）：
//   - signal 处理（SIGINT / SIGTERM → cancel ctx）
//   - 调用 cli.Execute 执行命令
//   - 把 error 通过 errs.CodeOf 映射到退出码
//
// 与 cmd/linctl/main.go（v1 入口）并存，互不影响。
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/clin211/lin/internal/cli"
	"github.com/clin211/lin/internal/pkg/errs"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	err := cli.Execute(ctx, os.Args[1:])
	if err == nil {
		os.Exit(0)
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		os.Exit(int(errs.CodeSIGINT))
	}

	cmdName := ""
	if len(os.Args) > 1 {
		cmdName = os.Args[1]
	}
	code := errs.CodeOf(cmdName, err)

	fmt.Fprintf(os.Stderr, "✗ %s\n", err.Error())
	var le *errs.Error
	if errors.As(err, &le) && le.Hint != "" {
		fmt.Fprintf(os.Stderr, "💡 %s\n", le.Hint)
	}

	os.Exit(int(code))
}
