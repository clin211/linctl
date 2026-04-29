package component

import (
	"sort"

	"github.com/clin211/linctl/internal/ast"
	"github.com/clin211/linctl/internal/codegen"
	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/project"
)

// WorkerKind 是 Worker 组件的注册 Kind。
const WorkerKind = "Worker"

// Worker variant 常量。多 variant 可以共存（例如 cron + kafka 在同一 worker 内）。
const (
	WorkerVariantCron       = "cron"
	WorkerVariantKafka      = "kafka"
	WorkerVariantCustomized = "customized"
)

// Worker 是 linctl 内置的后台任务组件实现（Phase 3 Story 3.2）。
//
// 与 WebServer 的差异：
//   - 没有 framework 字段（不绑定 gin / grpc）
//   - 多 variant 共存：spec.Variants 为合法集合（cron / kafka / customized）
//   - 每种 variant 必须提供对应载荷：CronSpec / KafkaSpec / Customized[]
//
// 模板分布：
//   - 通用：cmd/<name>/main.go + internal/<name>/runner.go
//   - 按 variant：internal/<name>/cron.go / kafka.go / customized.go
type Worker struct {
	spec project.Component
}

// NewWorker 通过 project.Component 构造 Worker 实例。
func NewWorker(spec project.Component) *Worker {
	return &Worker{spec: spec}
}

// WorkerFactory 是 Registry 用的 Factory（接受 map[string]any 形式）。
//
// orchestrator 在调度时优先使用 NewWorker；本 Factory 仅满足 Registry 接口。
func WorkerFactory(_ map[string]any) (Component, error) {
	return &Worker{}, nil
}

// Kind 返回 "Worker"。
func (w *Worker) Kind() string { return WorkerKind }

// Name 返回组件实例名。
func (w *Worker) Name() string { return w.spec.Name }

// Validate 校验 Worker 配置合法性。
//
// 规则：
//   - Name 必填
//   - Variants 必填，每个元素 ∈ {cron, kafka, customized}
//   - variants 含 cron       → spec.Cron != nil 且 Jobs ≥ 1
//   - variants 含 kafka      → spec.Kafka != nil 且 Brokers/Topics ≥ 1
//   - variants 含 customized → spec.Customized ≥ 1
func (w *Worker) Validate(_ *project.Project) error {
	if w.spec.Name == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"Worker.Name is required",
			"Set components[].name to a non-empty kebab-case identifier")
	}
	if len(w.spec.Variants) == 0 {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"Worker.Variants is required",
			"Set components[].variants to one or more of: cron, kafka, customized")
	}

	known := map[string]bool{
		WorkerVariantCron:       true,
		WorkerVariantKafka:      true,
		WorkerVariantCustomized: true,
	}
	seen := make(map[string]bool, len(w.spec.Variants))
	for _, v := range w.spec.Variants {
		if !known[v] {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"Worker.Variants contains unknown value %q", v).
				WithHint("Allowed: cron / kafka / customized.")
		}
		if seen[v] {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"Worker.Variants has duplicate %q", v)
		}
		seen[v] = true
	}

	if seen[WorkerVariantCron] {
		if w.spec.Cron == nil || len(w.spec.Cron.Jobs) == 0 {
			return linctlerr.New(linctlerr.ErrConfigInvalid,
				"Worker.Cron.Jobs is required when variants includes cron",
				"Add at least one job under spec.cron.jobs[].name")
		}
	}
	if seen[WorkerVariantKafka] {
		if w.spec.Kafka == nil ||
			len(w.spec.Kafka.Brokers) == 0 ||
			len(w.spec.Kafka.Topics) == 0 {
			return linctlerr.New(linctlerr.ErrConfigInvalid,
				"Worker.Kafka.{Brokers,Topics} are required when variants includes kafka",
				"Add at least one broker and one topic under spec.kafka")
		}
	}
	if seen[WorkerVariantCustomized] {
		if len(w.spec.Customized) == 0 {
			return linctlerr.New(linctlerr.ErrConfigInvalid,
				"Worker.Customized is required when variants includes customized",
				"Add at least one customized job under spec.customized[].name")
		}
	}
	return nil
}

// BasePairs 返回 Worker 自身的骨架文件 Pair 列表。
func (w *Worker) BasePairs(_ *project.Project) []codegen.Pair {
	owner := "Worker:" + w.spec.Name

	pairs := []codegen.Pair{
		{
			Dst:        "cmd/" + w.spec.Name + "/main.go",
			TemplateID: "templates/component/worker/cmd_main.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "internal/" + w.spec.Name + "/runner.go",
			TemplateID: "templates/component/worker/runner.go.tpl",
			Owner:      owner,
		},
		{
			Dst:        "Makefile",
			TemplateID: "templates/project/Makefile.tpl",
			Owner:      owner,
		},
		{
			Dst:        "go.mod",
			TemplateID: "templates/project/go.mod.tpl",
			Owner:      owner,
		},
		{
			Dst:        ".gitignore",
			TemplateID: "templates/project/gitignore.tpl",
			Owner:      owner,
		},
		{
			Dst:        "README.md",
			TemplateID: "templates/project/README.md.tpl",
			Owner:      owner,
		},
	}

	// 按字典序处理 variants，保证 BasePairs 输出稳定（snapshot 友好）。
	variants := append([]string(nil), w.spec.Variants...)
	sort.Strings(variants)
	for _, v := range variants {
		pairs = append(pairs, codegen.Pair{
			Dst:        "internal/" + w.spec.Name + "/" + v + ".go",
			TemplateID: "templates/component/worker/" + v + ".go.tpl",
			Owner:      owner,
		})
	}
	return pairs
}

// BaseMutators 返回组件 AST 修改。Phase 3 第一波不做 AST 注入。
func (w *Worker) BaseMutators(_ *project.Project) []ast.ASTMutator {
	return nil
}

// PostProcess 不需要副作用。
func (w *Worker) PostProcess(_ *project.Project, _ FileSystem) error {
	return nil
}

// SpecComponent 暴露原始 YAML struct（仅供 orchestrator 在调度时用）。
func (w *Worker) SpecComponent() project.Component {
	return w.spec
}
