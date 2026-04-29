package tracing

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Exporter 是 sdktrace.SpanExporter 的空实现。
type Exporter struct{}

// 编译期断言：确保 *Exporter 实现了 sdktrace.SpanExporter。
var _ sdktrace.SpanExporter = (*Exporter)(nil)

// ExportSpans 是 SpanExporter.ExportSpans 的空实现，丢弃所有 span。
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

// Shutdown 是 SpanExporter.Shutdown 的空实现。
func (e *Exporter) Shutdown(ctx context.Context) error {
	return nil
}

// NewEmptyExporter 构造一个不做任何输出 / 存储 / 转发的 *Exporter，
// 仅用来满足 OTel SDK 对 SpanExporter 的接口要求。
func NewEmptyExporter() *Exporter {
	return &Exporter{}
}
