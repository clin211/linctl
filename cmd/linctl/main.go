// Package main 是 linctl 的唯一入口。
//
// 设计原则（详见 docs/01-architecture.md §1.7 错误处理统一模式）：
//   - main 严格控制在 50 行以内
//   - 仅做 ctx 准备 + signal 处理 + 调用 cli.Execute
//   - 所有错误通过 linctlerr.LinctlError 由 cli 层返回，main 转换为退出码
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/clin211/lin/internal/cli"
	"github.com/clin211/lin/internal/linctlerr"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := cli.Execute(ctx, os.Args[1:]); err != nil {
		os.Exit(exitCodeFor(err))
	}
}

// exitCodeFor 按 docs/03-cli-design.md §3.6 规定的退出码语义映射 LinctlError。
func exitCodeFor(err error) int {
	if errors.Is(err, context.Canceled) {
		return 130
	}

	var lerr *linctlerr.LinctlError
	if errors.As(err, &lerr) {
		switch lerr.Code {
		case linctlerr.ErrConfigInvalid:
			return 2
		case linctlerr.ErrComponentNotFound:
			return 3
		case linctlerr.ErrFileConflict:
			return 4
		case linctlerr.ErrEnvironment:
			return 5
		case linctlerr.ErrNetwork:
			return 6
		case linctlerr.ErrSecurityPolicy:
			return 7
		}
	}

	fmt.Fprintf(os.Stderr, "linctl: %v\n", err)
	return 1
}
