package orchestrator

import (
	"fmt"
	"io"

	"github.com/clin211/lin/internal/codegen"
)

// Reporter 把 codegen.Report / Plan 输出给用户。
//
// Phase 1 仅文本输出；Phase 4 引入 colour + emoji + table。
type Reporter struct {
	out     io.Writer
	noColor bool
	noEmoji bool
}

// NewReporter 构造 Reporter。
func NewReporter(out io.Writer, noColor, noEmoji bool) *Reporter {
	return &Reporter{out: out, noColor: noColor, noEmoji: noEmoji}
}

// PrintPlan 打印 Plan 概要（Phase 1 仅 text 格式；JSON/YAML 由调用方处理）。
func (r *Reporter) PrintPlan(plan *codegen.Plan) {
	if plan == nil {
		return
	}
	fmt.Fprintf(r.out, "Plan: %d files (create=%d update=%d skip=%d)\n",
		plan.Stats.Total, plan.Stats.Create, plan.Stats.Update, plan.Stats.Skip)
	fmt.Fprintf(r.out, "Digest: %s\n", plan.Digest)

	for _, action := range plan.Actions {
		fmt.Fprintf(r.out, "  %-7s %s\n", action.Kind, action.Dst)
	}
}

// PrintReport 打印 Apply 结果。
func (r *Reporter) PrintReport(rep *codegen.Report) {
	if rep == nil {
		return
	}
	if rep.DryRun {
		fmt.Fprintf(r.out, "[dry-run] would: create=%d update=%d skip=%d\n",
			len(rep.Created), len(rep.Updated), len(rep.Skipped))
		return
	}
	fmt.Fprintf(r.out, "Done. created=%d updated=%d skipped=%d\n",
		len(rep.Created), len(rep.Updated), len(rep.Skipped))

	for _, f := range rep.Created {
		fmt.Fprintf(r.out, "  + %s\n", f)
	}
	for _, f := range rep.Updated {
		fmt.Fprintf(r.out, "  ~ %s\n", f)
	}
}
