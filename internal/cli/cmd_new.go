package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// newNewCmd 实现 `linctl new <name>`：从命令行参数构造一个 *project.Project，然后 plan + apply。
//
// Phase 1 退化策略（详见 SSOT §1.6）：
//   - 内部仍走 Plan → Apply 五段式
//   - 用户视角：默认 --auto-approve，无需手动看 plan
//   - --dry-run 仍然有效（仅打印 plan、不写盘）
func newNewCmd(g *GlobalOptions) *cobra.Command {
	var (
		moduleFlag   string
		kind         string
		framework    string
		storage      string
		featuresCSV  string
		variantsCSV  string
		commandsCSV  string
		port         int
		grpcPort     int
		grpcGateway  bool
	)

	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new Go microservice project from scratch",
		Long: `Create a new Go microservice project scaffold with linctl.

The Gin scaffold follows the miniblog-v4 enterprise architecture:
Cobra+Viper, Wire DI, Casbin authz, JWT, OpenTelemetry, Prometheus, GORM (or MongoDB).

Examples:
  # Gin + PostgreSQL (default)
  linctl new myblog --module github.com/foo/myblog --framework gin --storage gorm-postgres

  # Gin + MySQL
  linctl new myblog --module github.com/foo/myblog --framework gin --storage gorm-mysql

  # Gin + MongoDB
  linctl new myblog --module github.com/foo/myblog --framework gin --storage mongo

  # gRPC project
  linctl new grpcdemo --module github.com/foo/grpcdemo --framework grpc

  # gRPC + grpc-gateway (HTTP/JSON) — --port is HTTP, --grpc-port is gRPC
  linctl new apigw --module github.com/foo/apigw --framework grpc --grpc-gateway --port 8080 --grpc-port 9090

  # Worker project
  linctl new reporter --module github.com/foo/reporter --kind Worker --variants cron`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			if moduleFlag == "" {
				return linctlerr.New(linctlerr.ErrConfigInvalid,
					"--module is required",
					"Example: --module github.com/foo/"+name)
			}

			features := splitCSV(featuresCSV)
			variants := splitCSV(variantsCSV)
			commands := splitCSV(commandsCSV)
			proj := buildProjectFromFlags(buildProjectInputs{
				Name:         name,
				Module:       moduleFlag,
				Kind:         kind,
				Framework:     framework,
				Storage:      storage,
				Features:     features,
				Variants:     variants,
				Commands:     commands,
				Port:         port,
				GRPCPort:     grpcPort,
				GrpcGateway:  grpcGateway,
			})
			// 模板（如 deploy.yml.tpl / Dockerfile.tpl / PROJECT.tpl）依赖
			// Spec.Defaults.Image / Author / Docs 等嵌套块；这里统一补齐默认值，
			// 避免 missingkey=error 严格模式下 nil pointer 渲染失败。
			project.ApplyDefaults(proj)

			outDir, err := filepath.Abs(filepath.Join(g.RootDir, name))
			if err != nil {
				return linctlerr.Wrap(linctlerr.ErrEnvironment, err, "resolve output dir")
			}
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return linctlerr.Wrap(linctlerr.ErrEnvironment, err, "mkdir output")
			}

			orch, err := buildOrchestrator(outDir)
			if err != nil {
				return err
			}

			plan, pairs, err := orch.Plan(ctx, proj)
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

			fmt.Fprintf(cmd.OutOrStdout(), "Project layout generated at %s (%d files).\n",
				outDir, len(report.Created)+len(report.Updated))
			rep.PrintReport(report)

			fmt.Fprintf(cmd.OutOrStdout(), "\nNext steps:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  cd %s\n", filepath.Base(outDir))
			fmt.Fprintf(cmd.OutOrStdout(), "  go mod tidy\n")
			if isGRPCProject(proj) {
				fmt.Fprintf(cmd.OutOrStdout(), "  make protoc\n")
			} else {
				// Wire 生成 wire_gen.go 是 Gin 项目的强制前置步骤。
				fmt.Fprintf(cmd.OutOrStdout(), "  make wire   # generate internal/<app>/wire_gen.go\n")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  make build\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&moduleFlag, "module", "", "Go module path (required, e.g. github.com/foo/bar)")
	cmd.Flags().StringVar(&kind, "kind", "WebServer", "Component kind: WebServer | Worker | CLI")
	cmd.Flags().StringVar(&framework, "framework", "gin", "Web framework (WebServer only): gin | grpc")
	cmd.Flags().StringVar(&storage, "storage", "gorm-postgres", "Storage backend (WebServer only): gorm-mysql | gorm-postgres | mongo")
	cmd.Flags().StringVar(&featuresCSV, "features", "healthz", "Comma-separated features to enable")
	cmd.Flags().StringVar(&variantsCSV, "variants", "cron", "Worker variants (Worker only): cron,kafka,customized")
	cmd.Flags().StringVar(&commandsCSV, "commands", "version", "CLI sub-commands (CLI only): comma-separated, e.g. dump,migrate")
	cmd.Flags().IntVar(&port, "port", 8080, "HTTP port (WebServer+gin) or gRPC port default mapping (WebServer+grpc; use with --grpc-gateway for HTTP only)")
	cmd.Flags().IntVar(&grpcPort, "grpc-port", 0, "Native gRPC listen port (WebServer+grpc). If 0: same as --port, but 8080 is mapped to 9090; with --grpc-gateway, defaults to 9090")
	cmd.Flags().BoolVar(&grpcGateway, "grpc-gateway", false, "Enable grpc-gateway (WebServer+grpc only; --port=HTTP, --grpc-port=gRPC)")

	return cmd
}

func isGRPCProject(proj *project.Project) bool {
	if len(proj.Spec.Components) == 0 {
		return false
	}
	return proj.Spec.Components[0].Kind == "WebServer" && proj.Spec.Components[0].Framework == "grpc"
}

// buildProjectInputs 把 cmd_new flag 聚合为构造 Project 的输入；保持 buildProjectFromFlags 签名稳定。
type buildProjectInputs struct {
	Name        string
	Module      string
	Kind        string
	Framework   string
	Storage     string
	Features    []string
	Variants    []string
	Commands    []string
	Port        int
	GRPCPort    int
	GrpcGateway bool
}

func buildProjectFromFlags(in buildProjectInputs) *project.Project {
	switch in.Kind {
	case "Worker":
		return buildWorkerProject(in)
	case "CLI":
		return buildCLIProject(in)
	default: // WebServer 是默认
		return buildWebServerProject(in)
	}
}

func buildWebServerProject(in buildProjectInputs) *project.Project {
	// gin: Port=HTTP. grpc: GRPCPort=原生 gRPC；若启用 grpcGateway，则 Port=HTTP 网关、GRPCPort=gRPC。
	httpPort := in.Port
	grpcPort := 0
	grpcGateway := false
	if in.Framework == "grpc" {
		grpcGateway = in.GrpcGateway
		if grpcGateway {
			httpPort = in.Port
			if in.GRPCPort != 0 {
				grpcPort = in.GRPCPort
			} else {
				grpcPort = 9090
			}
		} else {
			grpcPort = in.GRPCPort
			if grpcPort == 0 {
				grpcPort = in.Port
				if grpcPort == 8080 {
					grpcPort = 9090 // 习惯端口
				}
			}
			httpPort = 0
		}
	}

	comp := project.Component{
		Kind:        "WebServer",
		Name:        in.Name,
		Framework:   in.Framework,
		Storage:     in.Storage,
		Features:    in.Features,
		Port:        httpPort,
		GRPCPort:    grpcPort,
		GrpcGateway: grpcGateway,
	}

	return &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:   in.Name,
			Module: in.Module,
		},
		Spec: project.Spec{
			Defaults: project.Defaults{
				Framework: in.Framework,
				Storage:   in.Storage,
			},
			Components: []project.Component{comp},
		},
	}
}

// buildCLIProject 构造 CLI 项目。
//
// 简化策略：
//   - 每个 --commands 中的元素生成一个 NamedSpec
//   - 默认 commands=version 让单条命令也能跑通
func buildCLIProject(in buildProjectInputs) *project.Project {
	cmds := in.Commands
	if len(cmds) == 0 {
		cmds = []string{"version"}
	}
	commands := make([]project.NamedSpec, 0, len(cmds))
	for _, name := range cmds {
		commands = append(commands, project.NamedSpec{Name: name})
	}
	return &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:   in.Name,
			Module: in.Module,
		},
		Spec: project.Spec{
			Components: []project.Component{
				{
					Kind:     "CLI",
					Name:     in.Name,
					Commands: commands,
				},
			},
		},
	}
}

// buildWorkerProject 构造 Worker 项目。
//
// 简化策略：
//   - 每个 variant 默认生成一个同名占位 spec（cron→Jobs[default]、kafka→Topics[events]、customized→Tasks[task]）
//   - 用户可在生成后编辑 linctl.yaml 添加更多 jobs / topics / tasks
func buildWorkerProject(in buildProjectInputs) *project.Project {
	if len(in.Variants) == 0 {
		in.Variants = []string{"cron"}
	}
	comp := project.Component{
		Kind:     "Worker",
		Name:     in.Name,
		Variants: in.Variants,
		Features: in.Features,
	}
	for _, v := range in.Variants {
		switch v {
		case "cron":
			comp.Cron = &project.CronSpec{
				Jobs: []project.NamedSpec{{Name: "default"}},
			}
		case "kafka":
			comp.Kafka = &project.KafkaSpec{
				Brokers: []string{"localhost:9092"},
				Topics:  []project.NamedSpec{{Name: "events"}},
			}
		case "customized":
			comp.Customized = []project.NamedSpec{{Name: "task"}}
		}
	}
	return &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:   in.Name,
			Module: in.Module,
		},
		Spec: project.Spec{
			Components: []project.Component{comp},
		},
	}
}

func buildOrchestrator(rootDir string) (*orchestrator.Orchestrator, error) {
	eng, err := template.New()
	if err != nil {
		return nil, err
	}
	fm, err := fs.NewFileManager(fs.Options{RootDir: rootDir})
	if err != nil {
		return nil, err
	}

	compReg := component.NewRegistry()
	if err := compReg.Register(component.WebServerKind, component.WebServerFactory); err != nil {
		return nil, err
	}
	if err := compReg.Register(component.WorkerKind, component.WorkerFactory); err != nil {
		return nil, err
	}
	if err := compReg.Register(component.CLIKind, component.CLIFactory); err != nil {
		return nil, err
	}

	featReg := feature.NewRegistry()
	if err := featReg.Register(builtin.NewHealthz()); err != nil {
		return nil, err
	}

	return orchestrator.New(orchestrator.Options{
		Engine:       eng,
		FM:           fm,
		ComponentReg: compReg,
		FeatureReg:   featReg,
	})
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// 防止 context 未使用的告警（cmd 里需要 ctx 但未来可能改 Builder 接受 ctx）
var _ = context.TODO
