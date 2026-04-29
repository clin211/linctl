package empty

import "context"

// emptyLogger 是 store.Logger 的空实现，所有方法都不输出。
type emptyLogger struct{}

// NewLogger 构造一个 *emptyLogger 实例。
func NewLogger() *emptyLogger {
	return &emptyLogger{}
}

// Error 是 store.Logger.Error 的空实现，不做任何操作。
func (l *emptyLogger) Error(ctx context.Context, err error, msg string, kvs ...any) {
	// 故意为空。
}
