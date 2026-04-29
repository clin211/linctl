package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics 聚合应用级 OpenTelemetry meter 与计数器。
type Metrics struct {
	Meter                     metric.Meter
	RESTResourceCreateCounter metric.Int64Counter
	RESTResourceGetCounter    metric.Int64Counter
}

// M 是初始化后的 *Metrics 全局句柄，业务代码通过 metrics.M 直接累加。
var M *Metrics

// Initialize 在服务启动时创建 meter 与计数器。
//
// 计数器命名遵循 Prometheus 规范：{subsystem}_{object}_{action}_{unit}。
func Initialize(ctx context.Context, scope string) error {
	meter := otel.Meter(scope + ".metrics")

	createCounter, _ := meter.Int64Counter(scope+"_resource_create_total",
		metric.WithDescription("REST 资源创建请求总数"))
	getCounter, _ := meter.Int64Counter(scope+"_resource_get_total",
		metric.WithDescription("REST 资源查询请求总数"))

	M = &Metrics{
		Meter:                     meter,
		RESTResourceCreateCounter: createCounter,
		RESTResourceGetCounter:    getCounter,
	}
	return nil
}
