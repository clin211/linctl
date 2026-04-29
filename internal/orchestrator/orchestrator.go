// Package orchestrator 是 linctl 的业务编排层（L2）。
//
// 职责：把"用户意图（命令 + 配置）"翻译成"具体的代码生成动作"，并管理生命周期：
// Plan → Apply → Report。
//
// 严格不做：实际的模板渲染（交给 L1 internal/template）、实际 IO（交给 L1 internal/fs）、
// 配置文件解析（交给 L1 internal/project）。
//
// SSOT 锁定（详见 docs/META-fix-decisions-2026-04-25.md §1.2）：
//   - ProjectLoader 物理位置在 internal/project/loader.go，不在本包
//   - 本包仅消费已加载的 *project.Project
package orchestrator

import (
	"context"
	"sort"

	"github.com/clin211/linctl/internal/codegen"
	"github.com/clin211/linctl/internal/component"
	"github.com/clin211/linctl/internal/feature"
	"github.com/clin211/linctl/internal/fs"
	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/project"
	"github.com/clin211/linctl/internal/template"
)

// Orchestrator 编排 Plan → Apply 流程。
//
// 一次实例化对应一次完整命令执行；不持有可变状态。
type Orchestrator struct {
	engine        *template.Engine
	fm            *fs.FileManager
	componentReg  *component.Registry
	featureReg    *feature.Registry
}

// Options 是 Orchestrator 构造选项。
type Options struct {
	Engine          *template.Engine
	FM              *fs.FileManager
	ComponentReg    *component.Registry
	FeatureReg      *feature.Registry
}

// New 构造 Orchestrator。所有依赖必填。
func New(opts Options) (*Orchestrator, error) {
	if opts.Engine == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Orchestrator: Engine is nil")
	}
	if opts.FM == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Orchestrator: FM is nil")
	}
	if opts.ComponentReg == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Orchestrator: ComponentReg is nil")
	}
	if opts.FeatureReg == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Orchestrator: FeatureReg is nil")
	}
	return &Orchestrator{
		engine:       opts.Engine,
		fm:           opts.FM,
		componentReg: opts.ComponentReg,
		featureReg:   opts.FeatureReg,
	}, nil
}

// Plan 计算从当前项目状态到目标 Project 的变更计划。
//
// 流程：
//  1. 对每个 component.spec 实例化 Component（按 Kind 路由）
//  2. 收集 BasePairs + 各 Feature.Apply 的 Pair（按 AppliesTo 过滤）
//  3. 用 PairBuilder 去重
//  4. 通过 codegen.Planner 计算 Plan（diff 磁盘 + 算 hash）
func (o *Orchestrator) Plan(ctx context.Context, p *project.Project) (*codegen.Plan, []codegen.Pair, error) {
	if p == nil {
		return nil, nil, linctlerr.New(linctlerr.ErrInternal, "Plan: project is nil")
	}

	pb := codegen.NewPairBuilder()

	for _, comp := range p.Spec.Components {
		c, err := o.buildComponent(comp)
		if err != nil {
			return nil, nil, err
		}
		if err := c.Validate(p); err != nil {
			return nil, nil, err
		}

		// 给每个 Pair 注入对应的 component-level TemplateData
		compData := &template.TemplateData{
			Project:    p,
			Component:  comp,
			CLIVersion: "dev",
		}
		basePairs := c.BasePairs(p)
		for i := range basePairs {
			if basePairs[i].Data == nil {
				basePairs[i].Data = compData
			}
		}
		pb.AddMany(basePairs)

		// 找出该 Component 适用的所有 Feature
		applicable := o.applicableFeatures(comp)
		ordered, err := o.featureReg.ResolveOrder(applicable)
		if err != nil {
			return nil, nil, err
		}

		for _, f := range ordered {
			pairs, err := f.Apply(ctx, comp)
			if err != nil {
				return nil, nil, linctlerr.Wrapf(linctlerr.ErrInternal, err,
					"feature %s.Apply failed for component %s", f.Name(), comp.Name)
			}
			featData := &template.TemplateData{
				Project:    p,
				Component:  comp,
				Feature:    f.Name(),
				CLIVersion: "dev",
			}
			for i := range pairs {
				if pairs[i].Data == nil {
					pairs[i].Data = featData
				}
			}
			pb.AddMany(pairs)
		}
	}

	pairs := pb.Build()
	plnr, err := codegen.NewPlanner(codegen.PlannerOptions{Engine: o.engine, FM: o.fm})
	if err != nil {
		return nil, nil, err
	}

	data := &template.TemplateData{
		Project:    p,
		CLIVersion: "dev", // TODO: inject from version package
	}

	plan, err := plnr.Plan(ctx, data, pairs)
	if err != nil {
		return nil, nil, err
	}
	return plan, pairs, nil
}

// Apply 执行 plan，产出 Report。
func (o *Orchestrator) Apply(ctx context.Context, p *project.Project, plan *codegen.Plan, pairs []codegen.Pair, dryRun bool) (*codegen.Report, error) {
	if plan == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Apply: plan is nil")
	}

	app, err := codegen.NewApplier(codegen.ApplierOptions{
		Engine: o.engine,
		FM:     o.fm,
		DryRun: dryRun,
	})
	if err != nil {
		return nil, err
	}

	data := &template.TemplateData{
		Project:    p,
		CLIVersion: "dev",
	}

	return app.Apply(ctx, data, pairs, plan)
}

// buildComponent 按 Kind 路由到具体 Component 实现。
//
// 当前支持：WebServer（Phase 1）/ Worker（Phase 3 Story 3.2）/ CLI（Phase 3 Story 3.4）。
func (o *Orchestrator) buildComponent(spec project.Component) (component.Component, error) {
	switch spec.Kind {
	case component.WebServerKind:
		return component.NewWebServer(spec), nil
	case component.WorkerKind:
		return component.NewWorker(spec), nil
	case component.CLIKind:
		return component.NewCLI(spec), nil
	default:
		return nil, linctlerr.Newf(linctlerr.ErrNotImplementedYet,
			"component kind %q not supported", spec.Kind).
			WithHint("Supported: WebServer / Worker / CLI.")
	}
}

// applicableFeatures 返回该 Component 启用的、本注册中心已知的 Feature 名称集合。
func (o *Orchestrator) applicableFeatures(comp project.Component) []string {
	if len(comp.Features) == 0 {
		return nil
	}
	out := make([]string, 0, len(comp.Features))
	for _, name := range comp.Features {
		f, err := o.featureReg.Get(name)
		if err != nil {
			// Phase 1 容忍未知 feature 名（用户可能写了未实现的 feature）
			// 后续可改为严格 fail
			continue
		}
		// 只保留 AppliesTo 列表中含 comp.Kind 的 feature
		for _, k := range f.AppliesTo() {
			if k == comp.Kind {
				out = append(out, name)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}
