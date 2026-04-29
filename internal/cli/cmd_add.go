package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	last "github.com/clin211/lin/internal/ast"
	"github.com/clin211/lin/internal/codegen"
	"github.com/clin211/lin/internal/component"
	"github.com/clin211/lin/internal/feature"
	"github.com/clin211/lin/internal/feature/builtin"
	"github.com/clin211/lin/internal/fs"
	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/orchestrator"
	"github.com/clin211/lin/internal/project"
	"github.com/clin211/lin/internal/template"
	"github.com/spf13/cobra"
)

// newAddCmd 实现 `linctl add api <Resource>` 等子命令。
//
// Phase 2 范围：
//   - 生成 biz_<resource>.go / store_<resource>.go / handler_<resource>.go 三个新文件
//   - 通过 AST 注入到 biz/biz.go 与 store/store.go 的 IBiz / IStore 接口
//   - 不更新 router.go（用户手动注册路由，Phase 3 可自动化）
func newAddCmd(g *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new resource / component / feature to an existing linctl project",
	}
	cmd.AddCommand(newAddAPICmd(g))
	return cmd
}

func newAddAPICmd(g *GlobalOptions) *cobra.Command {
	var (
		moduleFlag    string
		componentName string
		framework     string
	)
	cmd := &cobra.Command{
		Use:   "api <ResourceName>",
		Short: "Add a REST resource (biz + store + handler) to an existing component",
		Long: `Add a REST resource. Generates biz/store/handler files and AST-injects
new methods into IBiz and IStore interfaces.

Example:
  linctl add api Post
  linctl add api Comment --component myblog`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			resource := args[0]
			pascal := toPascal(resource)
			lower := strings.ToLower(pascal[:1]) + pascal[1:]

			rootDir, err := filepath.Abs(g.RootDir)
			if err != nil {
				return linctlerr.Wrap(linctlerr.ErrEnvironment, err, "resolve root")
			}

			if componentName == "" {
				// 默认：用根目录名（约定 linctl new <name> 后用户在该目录运行）
				componentName = filepath.Base(rootDir)
			}
			if moduleFlag == "" {
				moduleFlag = "github.com/example/" + componentName
			}

			proj := buildAddProject(componentName, moduleFlag, framework)

			orch, err := buildOrchestrator(rootDir)
			if err != nil {
				return err
			}

			pairs := buildResourcePairs(proj, componentName, pascal, lower)
			plnr, err := codegen.NewPlanner(codegen.PlannerOptions{
				Engine: extractEngineFromOrch(),
				FM:     extractFMFromOrch(rootDir),
			})
			if err != nil {
				return err
			}
			data := buildResourceData(proj, componentName, pascal, lower)

			plan, err := plnr.Plan(ctx, data, pairs)
			if err != nil {
				return err
			}

			rep := orchestrator.NewReporter(cmd.OutOrStdout(), g.NoColor, g.NoEmoji)
			if g.DryRun {
				rep.PrintPlan(plan)
				return nil
			}

			report, err := orch.Apply(ctx, proj, plan, pairs, false)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Resource %s generated (%d files).\n",
				pascal, len(report.Created)+len(report.Updated))

			// AST 注入：在 biz/biz.go 和 store/store.go 中加入新方法
			fm, err := fs.NewFileManager(fs.Options{RootDir: rootDir})
			if err != nil {
				return err
			}
			batch := last.NewBatch(fm)
			mutators := []last.ASTMutator{
				&last.AddInterfaceMethodMutator{
					FilePath:      "internal/" + componentName + "/biz/biz.go",
					InterfaceName: "IBiz",
					MethodName:    pascal + "V1",
					Returns:       pascal + "Biz",
					Doc:           pascal + "V1 返回 " + pascal + " 业务对象（v1 版本）。",
				},
				&last.AddInterfaceMethodMutator{
					FilePath:      "internal/" + componentName + "/store/store.go",
					InterfaceName: "IStore",
					MethodName:    pascal,
					Returns:       pascal + "Store",
					Doc:           pascal + " 返回 " + pascal + " 数据访问对象。",
				},
			}
			batchRes, err := batch.Apply(ctx, mutators)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "AST: modified %d files.\n", len(batchRes.ModifiedFiles))
			for _, f := range batchRes.ModifiedFiles {
				fmt.Fprintf(cmd.OutOrStdout(), "  ~ %s\n", f)
			}
			for _, c := range batchRes.Conflicts {
				fmt.Fprintf(cmd.OutOrStdout(), "  ! conflict: %s\n", c.Error())
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&moduleFlag, "module", "", "Go module path (default: read from linctl.yaml)")
	cmd.Flags().StringVar(&componentName, "component", "", "Component name (default: current directory name)")
	cmd.Flags().StringVar(&framework, "framework", "gin", "Web framework: gin | grpc")
	return cmd
}

func buildAddProject(componentName, modulePath, framework string) *project.Project {
	return &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:   componentName,
			Module: modulePath,
		},
		Spec: project.Spec{
			Components: []project.Component{
				{
					Kind:      "WebServer",
					Name:      componentName,
					Framework: framework,
					Storage:   "memory",
				},
			},
		},
	}
}

func buildResourcePairs(p *project.Project, componentName, pascal, lower string) []codegen.Pair {
	owner := "AddAPI:" + pascal
	data := buildResourceData(p, componentName, pascal, lower)
	return []codegen.Pair{
		{
			Dst:        "internal/" + componentName + "/biz/biz_" + lower + ".go",
			TemplateID: "templates/feature/resource/biz_resource.go.tpl",
			Owner:      owner,
			Data:       data,
		},
		{
			Dst:        "internal/" + componentName + "/store/store_" + lower + ".go",
			TemplateID: "templates/feature/resource/store_resource.go.tpl",
			Owner:      owner,
			Data:       data,
		},
		{
			Dst:        "internal/" + componentName + "/handler/handler_" + lower + ".go",
			TemplateID: "templates/feature/resource/handler_resource.go.tpl",
			Owner:      owner,
			Data:       data,
		},
	}
}

func buildResourceData(p *project.Project, componentName, pascal, lower string) any {
	custom := map[string]any{
		"ResourcePascal": pascal,
		"ResourceLower":  lower,
	}
	return &template.TemplateData{
		Project:    p,
		Component:  p.Spec.Components[0],
		CLIVersion: "dev",
		Custom:     custom,
	}
}

// extractEngineFromOrch / extractFMFromOrch 是 add 命令的临时辅助：
// 这里我们直接重新构造 Engine + FM（与 buildOrchestrator 一致）。
// 后续可以重构为暴露 orch.Engine() / orch.FM() 方法。
func extractEngineFromOrch() *template.Engine {
	eng, _ := template.New()
	return eng
}

func extractFMFromOrch(rootDir string) *fs.FileManager {
	fm, _ := fs.NewFileManager(fs.Options{RootDir: rootDir})
	return fm
}

// toPascal 把 "post" / "user_post" 转成 "Post" / "UserPost"。
func toPascal(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		out += string(runes)
	}
	return out
}

// 防止 osArgs 编译警告
var _ = os.Args
var _ = context.TODO

// 防止 component / feature / builtin / orchestrator 引用未触发（被 buildOrchestrator 使用）
var (
	_ = component.WebServerKind
	_ = feature.NewRegistry
	_ = builtin.NewHealthz
	_ = orchestrator.NewReporter
)
