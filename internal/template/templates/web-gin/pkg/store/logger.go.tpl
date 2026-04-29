package store

import (
	"context"
)

// Logger 用于记录 store 层错误日志的接口。
type Logger interface {
	// Error 记录一条带上下文的错误日志。
	Error(ctx context.Context, err error, message string, kvs ...any)
}
