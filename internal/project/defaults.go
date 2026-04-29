package project

// 默认值常量（与 docs/04-config-schema.md §4.3.3 表格保持一致）。
const (
	defaultFramework      = "gin"
	defaultStorage        = "memory"
	defaultDeploy         = "docker"
	defaultMakefile       = "unstructured"
	defaultProtoVersion   = "v1"
	defaultDockerfileMode = "multi-stage"
	defaultDistrolessMode = "always"
	defaultDocsLanguage   = "zh-CN"
	defaultLogging        = "slog"
	defaultMetrics        = "prometheus"
	defaultTracing        = "otlp"
)

// ApplyDefaults 把 spec.defaults 中的默认值写入未填的字段，并把 defaults
// 继承到每个 Component（仅 Framework / Storage 两个会被组件覆盖默认）。
//
// 调用时机：在 Loader 解码完成之后、validator 强校验之前。
//
// 边界：
//   - p == nil → no-op（不 panic）
//   - 已显式声明的字段保持原值不动
//   - Defaults 内嵌的 *ImageDefaults / *DocsDefaults / *TelemetryDefaults
//     若为 nil，会被实例化为零值后再填默认（避免后续 nil-deref）
//
// 该函数是幂等的：连续调用多次结果相同。
func ApplyDefaults(p *Project) {
	if p == nil {
		return
	}

	if p.APIVersion == "" {
		p.APIVersion = APIVersionV1
	}
	if p.Kind == "" {
		p.Kind = KindProject
	}

	applySpecDefaults(&p.Spec.Defaults)
	inheritDefaultsToComponents(&p.Spec)
}

// applySpecDefaults 设置 spec.defaults 内未填的标量字段，并在嵌套块为 nil 时实例化。
func applySpecDefaults(d *Defaults) {
	if d == nil {
		return
	}

	if d.Framework == "" {
		d.Framework = defaultFramework
	}
	if d.Storage == "" {
		d.Storage = defaultStorage
	}
	if d.Deploy == "" {
		d.Deploy = defaultDeploy
	}
	if d.Makefile == "" {
		d.Makefile = defaultMakefile
	}
	if d.ProtoVersion == "" {
		d.ProtoVersion = defaultProtoVersion
	}

	if d.Image == nil {
		d.Image = &ImageDefaults{}
	}
	if d.Image.DockerfileMode == "" {
		d.Image.DockerfileMode = defaultDockerfileMode
	}
	if d.Image.DistrolessMode == "" {
		d.Image.DistrolessMode = defaultDistrolessMode
	}

	if d.Docs == nil {
		d.Docs = &DocsDefaults{}
	}
	if len(d.Docs.Languages) == 0 {
		d.Docs.Languages = []string{defaultDocsLanguage}
	}

	if d.Telemetry == nil {
		d.Telemetry = &TelemetryDefaults{}
	}
	if d.Telemetry.Logging == "" {
		d.Telemetry.Logging = defaultLogging
	}
	if d.Telemetry.Metrics == "" {
		d.Telemetry.Metrics = defaultMetrics
	}
	if d.Telemetry.Tracing == "" {
		d.Telemetry.Tracing = defaultTracing
	}
}

// inheritDefaultsToComponents 把 spec.defaults 的 Framework / Storage 注入到
// 那些未显式声明的 Component。
//
// 仅这两个字段会被 component 默认继承——其他 defaults 字段（image / docs / telemetry / ...）
// 是项目级的，不属于组件维度。
func inheritDefaultsToComponents(spec *Spec) {
	if spec == nil {
		return
	}
	d := spec.Defaults
	for i := range spec.Components {
		c := &spec.Components[i]
		if c.Framework == "" {
			c.Framework = d.Framework
		}
		if c.Storage == "" {
			c.Storage = d.Storage
		}
	}
}
