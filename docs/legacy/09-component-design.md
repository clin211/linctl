# 09. Component 抽象与内置组件设计

> 本文档定义 [Component](./99-glossary.md#component) 抽象的接口契约、生命周期、以及三种内置实现（WebServer / Worker / CLI）的详细设计。Component 是 linctl 的"组件级"建模单位，与 [Feature](./99-glossary.md#feature)（横切能力）正交。

## 9.1 为什么要 Component 抽象

### 9.1.1 osbuilder 现状的问题

osbuilder 把组件类型硬编码为 4 个 struct（`WebServer` / `JobServer` / `MQServer` / `CLITool`），每个 struct 都有自己的 `Pairs() map[string]string` 方法 + 大量 if-else 分支。这导致：

1. **重复代码**：JobServer 和 MQServer 的 90% 逻辑相同（同样的 `cmd/<name>/main.go`、Wire、配置加载），但被拆成两个 struct。
2. **新增组件类型代价大**：要做"声明式 API server"（DeclServer）就得新加一整套 struct + Pairs() + Complete()。
3. **配置语法不一致**：JobServer 用 `topics` 字段，MQServer 用 `subscribes`，但其实是同一个抽象（消息消费）。
4. **组件间通信不清晰**：当 WebServer 和 Worker 都想用同一个 Storage 时，配置上没有"共享存储"的表达方式。

### 9.1.2 linctl 的解法

提取 `Component` 接口 + 三种内置实现：

| 内置组件 | 对应 osbuilder | 职责 |
| --- | --- | --- |
| `WebServer` | WebServer | HTTP/gRPC 服务 |
| `Worker` | JobServer + MQServer + 自定义 watcher | 后台任务（cron + MQ + custom） |
| `CLI` | CLITool | 命令行工具 |

> **决策原因**：见 [ADR-005](./adr/005-feature-as-first-class.md) 中关于 Component 与 Feature 分离的讨论。

## 9.2 Component 接口契约

```go
// internal/component/component.go
package component

import (
    "github.com/<org>/linctl/internal/ast"
    "github.com/<org>/linctl/internal/codegen"
    "github.com/<org>/linctl/internal/project"
)

// Component 是 linctl 中"可独立编译/运行的产物"的抽象。
// 实现了本接口的类型可被注册到 component.Registry，并参与 plan/apply 流水线。
type Component interface {
    // ============= 元数据 =============

    // Kind 返回组件类型名（如 "WebServer" / "Worker" / "CLI"）。
    // 必须与 project.Component.Kind 字段值一致，用于路由。
    Kind() string

    // Name 返回组件实例名（如 "mb-apiserver"），项目内唯一。
    Name() string

    // Validate 校验该组件的配置合法性。
    // 例：WebServer 要求 framework 不为空；Worker 要求 variants 不为空。
    Validate(p *project.Project) error

    // ============= 文件生成（纯计算）=============

    // BasePairs 返回组件自身（不含 Feature）需要生成的文件对。
    // 这些是组件的"骨架"：cmd/<name>/main.go, internal/<name>/server.go 等。
    //
    // 契约：纯函数，不做任何 IO（不读文件、不写文件、不调外部命令）。
    // 仅根据 *project.Project 计算并返回 []Pair。
    BasePairs(p *project.Project) []codegen.Pair

    // ============= AST 修改（纯计算）=============

    // BaseMutators 返回组件自身需要执行的 AST 修改（不含 Feature 贡献）。
    // 例：CLI 组件需要在 cmd/all.go 中加 `_ "..."` import。
    //
    // 契约：返回 mutator 描述（也是纯函数）；mutator 的实际 Apply 由
    // ast.Injector 在 Applier 阶段统一调度（届时才接触磁盘）。
    BaseMutators(p *project.Project) []ast.ASTMutator

    // ============= 显式声明的副作用扩展点 =============

    // PostProcess 是组件**显式声明**的"无法用 Pair/Mutator 表达"的副作用钩子。
    //
    // 关键约束（与 BasePairs 的纯计算契约不冲突）：
    //   1. 仅在所有 Pair 已写入磁盘 + AST 注入完成 + Hook 之前调用。
    //   2. 必须通过 fm (FileSystem 抽象) 访问磁盘，禁止直接 os.OpenFile。
    //   3. **必须由 Applier 统一调度**（同一把项目锁 + 同一份 plan/digest）。
    //      绝对禁止在 Plan 阶段调用 PostProcess。
    //   4. 99% 场景不应使用；新增组件如果要用，必须在 PR 中说明无法用 Pair/Mutator
    //      表达的原因。
    //
    // 例：CLI 组件可在此重新生成 root.go 的子命令注册（罕见场景）。
    PostProcess(p *project.Project, fm FileSystem) error
}

// FileSystem 是组件对文件系统的访问抽象（来自 internal/fs.FileManager）。
// PostProcess 必须通过此接口访问磁盘，禁止直接使用 os.* / ioutil.*。
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, content []byte) error
    Exists(path string) (bool, error)
}
```

### 9.2.1 接口设计原则

| 原则 | 解释 |
| --- | --- |
| **接口 ≤ 6 个方法** | 抽象简单，便于第三方实现 |
| **BasePairs / BaseMutators 纯计算** | 严禁 IO、严禁随机性，便于单测与缓存；所有 IO 由 Applier (L0) 统一调度 |
| **PostProcess 是显式声明的副作用扩展点** | 不破坏 BasePairs 的纯计算契约；必须通过 `FileSystem` 抽象访问磁盘；必须由 Applier 在持锁阶段统一调度 |
| **Pair 自带 Owner** | 便于 plan 报告中追溯"这个文件是哪个 Component 贡献的" |
| **PostProcess 是 escape hatch** | 99% 场景不需要；仅用于无法用 Pair/Mutator 表达的极端情况 |

### 9.2.2 Component 与 Feature 的关系

```mermaid
flowchart LR
    Project --> Components
    Components --> WS["WebServer"]
    Components --> WK["Worker"]
    Components --> CL["CLI"]

    Features --> Healthz
    Features --> OTel
    Features --> User
    Features --> WS

    WS -->|BasePairs<br/>BaseMutators| PB[PairBuilder]
    Healthz -.AppliesTo: WebServer.->|Apply<br/>Mutators| PB
    OTel -.AppliesTo: WebServer/Worker.->|Apply<br/>Mutators| PB

    PB --> Plan
```

**关键不变量**：

1. Component **拥有**生命周期（每个 Component 独立 plan/apply）。
2. Feature **横切**多个 Component（一个 Feature 可作用于多种 Kind）。
3. Pair 的 `Owner` 字段标识贡献者，便于多人协作时定位。
4. 若 Component 与 Feature 贡献了相同 `Dst`，按"后写覆盖"语义合并；同时记录 warning。

## 9.3 Component Registry

```go
// internal/component/registry.go
package component

import (
    "fmt"
    "sync"
)

type Constructor func(c project.Component) (Component, error)

type Registry struct {
    mu     sync.RWMutex
    ctors  map[string]Constructor
}

func NewRegistry() *Registry {
    return &Registry{ctors: make(map[string]Constructor)}
}

func (r *Registry) Register(kind string, ctor Constructor) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    if _, exists := r.ctors[kind]; exists {
        return fmt.Errorf("component kind %q already registered", kind)
    }
    r.ctors[kind] = ctor
    return nil
}

func (r *Registry) Build(spec project.Component) (Component, error) {
    r.mu.RLock()
    ctor, ok := r.ctors[spec.Kind]
    r.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("unknown component kind %q. Available: %v", spec.Kind, r.list())
    }
    return ctor(spec)
}

func (r *Registry) list() []string {
    out := make([]string, 0, len(r.ctors))
    for k := range r.ctors {
        out = append(out, k)
    }
    return out
}

// Default 全局注册表，由 init() 注册三种内置组件
var Default = NewRegistry()

func MustRegister(kind string, ctor Constructor) {
    if err := Default.Register(kind, ctor); err != nil {
        panic(err)
    }
}
```

内置组件注册：

```go
// internal/component/webserver.go
func init() {
    MustRegister("WebServer", func(spec project.Component) (Component, error) {
        return NewWebServer(spec)
    })
}

// internal/component/worker.go
func init() {
    MustRegister("Worker", func(spec project.Component) (Component, error) {
        return NewWorker(spec)
    })
}

// internal/component/cli.go
func init() {
    MustRegister("CLI", func(spec project.Component) (Component, error) {
        return NewCLI(spec)
    })
}
```

## 9.4 内置组件 1：WebServer

### 9.4.1 定位

`WebServer` 是 linctl 最常用的组件类型，对应"对外提供 HTTP / gRPC 接口的服务"。

支持两种 framework：

- `gin` —— 纯 HTTP/JSON
- `grpc` —— gRPC（可选 grpc-gateway 提供 HTTP/JSON 代理）

> Phase 5 通过插件机制支持 `kratos` / `go-zero` / `kitex` 等。

### 9.4.2 配置示例

```yaml
- kind: WebServer
  name: mb-apiserver
  framework: gin
  storage: gorm-postgres
  features: [healthz, opentelemetry, user]
  port: 5555
  registry: nacos
  clients: [fake, oss]
  resources:
    - name: post
    - name: comment
```

### 9.4.3 实现

```go
// internal/component/webserver.go
package component

import (
    "fmt"
    "path/filepath"

    "github.com/<org>/linctl/internal/ast"
    "github.com/<org>/linctl/internal/codegen"
    "github.com/<org>/linctl/internal/project"
)

type WebServer struct {
    spec project.Component
}

func NewWebServer(spec project.Component) (*WebServer, error) {
    return &WebServer{spec: spec}, nil
}

func (w *WebServer) Kind() string { return "WebServer" }
func (w *WebServer) Name() string { return w.spec.Name }

func (w *WebServer) Validate(p *project.Project) error {
    if w.spec.Framework == "" {
        return fmt.Errorf("webserver %q: framework is required", w.spec.Name)
    }
    if w.spec.Framework == "grpc" && w.spec.GRPCPort == 0 {
        return fmt.Errorf("webserver %q: grpcPort is required when framework=grpc", w.spec.Name)
    }
    if w.spec.Framework == "gin" && w.spec.GRPCPort != 0 {
        return fmt.Errorf("webserver %q: grpcPort cannot be set when framework=gin", w.spec.Name)
    }
    return nil
}

func (w *WebServer) BasePairs(p *project.Project) []codegen.Pair {
    pairs := []codegen.Pair{
        // === cmd 入口 ===
        w.tplPair("templates/component/webserver/cmd/main.go.tpl",
            filepath.Join("cmd", w.spec.Name, "main.go")),
        w.tplPair("templates/component/webserver/cmd/options.go.tpl",
            filepath.Join("cmd", w.spec.Name, "app/options.go")),
        w.tplPair("templates/component/webserver/cmd/server.go.tpl",
            filepath.Join("cmd", w.spec.Name, "app/server.go")),

        // === internal 骨架 ===
        w.tplPair("templates/component/webserver/internal/server.go.tpl",
            filepath.Join("internal", w.spec.Name, "server.go")),
        w.tplPair("templates/component/webserver/internal/wire.go.tpl",
            filepath.Join("internal", w.spec.Name, "wire.go")),

        // === framework 特定 ===
        w.tplPair(
            fmt.Sprintf("templates/framework/%s/server.go.tpl", w.spec.Framework),
            filepath.Join("internal", w.spec.Name, "framework_server.go"),
        ),
        w.tplPair(
            fmt.Sprintf("templates/framework/%s/handler/handler.go.tpl", w.spec.Framework),
            filepath.Join("internal", w.spec.Name, "handler/handler.go"),
        ),

        // === storage 特定 ===
        w.tplPair(
            fmt.Sprintf("templates/storage/%s/store/store.go.tpl", w.spec.Storage),
            filepath.Join("internal", w.spec.Name, "store/store.go"),
        ),

        // === 配置文件 ===
        w.tplPair("templates/component/webserver/configs/server.yaml.tpl",
            filepath.Join("configs", w.spec.Name+".yaml")),
    }

    // === 每个 Resource 一组文件（handler/biz/store/proto/model）===
    for _, r := range w.spec.Resources {
        pairs = append(pairs, w.resourcePairs(p, r)...)
    }

    return pairs
}

func (w *WebServer) BaseMutators(p *project.Project) []ast.ASTMutator {
    // WebServer 自身的 base 阶段无 AST 修改。
    // Resource 的 AST 注入由 add api 流程触发，不在 BasePairs 范围内。
    return nil
}

func (w *WebServer) PostProcess(p *project.Project, fm FileSystem) error {
    // 通常无需 post-process
    return nil
}

// 辅助方法
func (w *WebServer) tplPair(tplID, dst string) codegen.Pair {
    return codegen.Pair{
        Dst:        dst,
        TemplateID: tplID,
        Mode:       codegen.WriteCreate,
        Owner:      "WebServer:" + w.spec.Name,
    }
}

func (w *WebServer) resourcePairs(p *project.Project, r project.Resource) []codegen.Pair {
    apiVersion := p.Spec.Defaults.ProtoVersion   // 字段已重命名，详见 META §1.18
    base := []codegen.Pair{
        // proto
        {
            Dst:        filepath.Join("pkg/api", w.spec.Name, apiVersion, r.Name+".proto"),
            TemplateID: "templates/component/webserver/api/resource.proto.tpl",
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
        // handler
        {
            Dst:        filepath.Join("internal", w.spec.Name, "handler", w.spec.Framework, r.Name+".go"),
            TemplateID: fmt.Sprintf("templates/framework/%s/handler/api/resource.go.tpl", w.spec.Framework),
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
        // biz
        {
            Dst:        filepath.Join("internal", w.spec.Name, "biz", apiVersion, r.Name, r.Name+".go"),
            TemplateID: "templates/component/webserver/biz/resource.go.tpl",
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
        // store
        {
            Dst:        filepath.Join("internal", w.spec.Name, "store", r.Name+".go"),
            TemplateID: fmt.Sprintf("templates/storage/%s/store/resource.go.tpl", w.spec.Storage),
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
        // model
        {
            Dst:        filepath.Join("internal", w.spec.Name, "model", r.Name+".gen.go"),
            TemplateID: fmt.Sprintf("templates/storage/%s/model/resource.go.tpl", w.spec.Storage),
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
        // validation
        {
            Dst:        filepath.Join("internal", w.spec.Name, "pkg/validation", r.Name+".go"),
            TemplateID: "templates/component/webserver/validation/resource.go.tpl",
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
        // conversion
        {
            Dst:        filepath.Join("internal", w.spec.Name, "pkg/conversion", r.Name+".go"),
            TemplateID: "templates/component/webserver/conversion/resource.go.tpl",
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
        // errno
        {
            Dst:        filepath.Join("internal/pkg/errno", r.Name+".go"),
            TemplateID: "templates/component/webserver/errno/resource.go.tpl",
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("WebServer:%s:%s", w.spec.Name, r.Name),
        },
    }
    return base
}
```

### 9.4.4 add api 子流程

`linctl add api` 命令的核心逻辑（不在 `BasePairs` 中）：

1. 把 `--kinds` 中的资源加入 `c.Resources`。
2. 调用 `WebServer.resourcePairs(p, newResource)` 生成新资源的 8 个文件 Pair。
3. 调用 **AST mutators**：
   - `AddInterfaceMethodMutator` 给 `internal/<name>/biz/biz.go` 的 `IBiz` 加 `<Resource>V1() resource.<Resource>Biz`
   - `AddStructMethodMutator` 给 biz struct 加实现
   - `AddImportMutator` 加新 import
   - 同样的 4 个 mutator 操作 `internal/<name>/store/store.go`
   - `AddProtoRPCMutator` 给 `pkg/api/<name>/<v>/<name>.proto` 加 5 个 RPC method
4. 持久化：把新 Resource 写入 `linctl.yaml` 的 `c.resources` 列表。

> 详见 [03-cli-design.md §3.3.2](./03-cli-design.md) 与 [07-ast-injection.md §7.5](./07-ast-injection.md)。

## 9.5 内置组件 2：Worker

### 9.5.1 定位

`Worker` 是合并后的"后台任务"组件，统一表达 osbuilder 的 JobServer + MQServer + 自定义 watcher。

支持三种 variant，可同时启用：

- `cron` —— 定时任务（基于 robfig/cron）
- `kafka` —— Kafka 消费者
- `customized` —— 自定义 watcher（如 LLM 训练监听器）

### 9.5.2 配置示例

```yaml
- kind: Worker
  name: mb-worker
  storage: gorm-postgres
  features: [opentelemetry, preloader]
  variants: [cron, kafka, customized]
  cron:
    jobs:
      - name: dailyReport
      - name: dataSync
  kafka:
    brokers: ["kafka:9092"]
    topics:
      - name: order.created
      - name: user.registered
  customized:
    - name: llmtrain
```

### 9.5.3 实现

```go
// internal/component/worker.go
package component

import (
    "fmt"
    "path/filepath"

    "github.com/<org>/linctl/internal/ast"
    "github.com/<org>/linctl/internal/codegen"
    "github.com/<org>/linctl/internal/project"
)

type Worker struct {
    spec project.Component
}

func NewWorker(spec project.Component) (*Worker, error) {
    return &Worker{spec: spec}, nil
}

func (w *Worker) Kind() string { return "Worker" }
func (w *Worker) Name() string { return w.spec.Name }

func (w *Worker) Validate(p *project.Project) error {
    if len(w.spec.Variants) == 0 {
        return fmt.Errorf("worker %q: at least one variant required", w.spec.Name)
    }
    for _, v := range w.spec.Variants {
        switch v {
        case "cron":
            if w.spec.Cron == nil || len(w.spec.Cron.Jobs) == 0 {
                return fmt.Errorf("worker %q: cron variant requires cron.jobs", w.spec.Name)
            }
        case "kafka":
            if w.spec.Kafka == nil || len(w.spec.Kafka.Topics) == 0 {
                return fmt.Errorf("worker %q: kafka variant requires kafka.topics", w.spec.Name)
            }
            if len(w.spec.Kafka.Brokers) == 0 {
                return fmt.Errorf("worker %q: kafka variant requires kafka.brokers", w.spec.Name)
            }
        case "customized":
            if len(w.spec.Customized) == 0 {
                return fmt.Errorf("worker %q: customized variant requires customized list", w.spec.Name)
            }
        default:
            return fmt.Errorf("worker %q: unknown variant %q", w.spec.Name, v)
        }
    }
    return nil
}

func (w *Worker) BasePairs(p *project.Project) []codegen.Pair {
    pairs := []codegen.Pair{
        // 通用入口
        w.tplPair("templates/component/worker/cmd/main.go.tpl",
            filepath.Join("cmd", w.spec.Name, "main.go")),
        w.tplPair("templates/component/worker/cmd/options.go.tpl",
            filepath.Join("cmd", w.spec.Name, "app/options.go")),
        w.tplPair("templates/component/worker/internal/scheduler.go.tpl",
            filepath.Join("internal", w.spec.Name, "scheduler.go")),

        // 配置
        w.tplPair("templates/component/worker/configs/worker.yaml.tpl",
            filepath.Join("configs", w.spec.Name+".yaml")),

        // storage 共用
        w.tplPair(
            fmt.Sprintf("templates/storage/%s/store/store.go.tpl", w.spec.Storage),
            filepath.Join("internal", w.spec.Name, "store/store.go"),
        ),
    }

    // === Variant: cron ===
    if w.hasVariant("cron") {
        pairs = append(pairs,
            w.tplPair("templates/component/worker/cron/registry.go.tpl",
                filepath.Join("internal", w.spec.Name, "cron/registry.go")),
        )
        for _, job := range w.spec.Cron.Jobs {
            pairs = append(pairs, codegen.Pair{
                Dst:        filepath.Join("internal", w.spec.Name, "cron", job.Name+".go"),
                TemplateID: "templates/component/worker/cron/job.go.tpl",
                Mode:       codegen.WriteCreate,
                Owner:      fmt.Sprintf("Worker:%s:cron:%s", w.spec.Name, job.Name),
            })
        }
    }

    // === Variant: kafka ===
    if w.hasVariant("kafka") {
        pairs = append(pairs,
            w.tplPair("templates/component/worker/kafka/consumer.go.tpl",
                filepath.Join("internal", w.spec.Name, "kafka/consumer.go")),
        )
        for _, t := range w.spec.Kafka.Topics {
            pairs = append(pairs, codegen.Pair{
                Dst:        filepath.Join("internal", w.spec.Name, "kafka/handlers", t.Name+".go"),
                TemplateID: "templates/component/worker/kafka/handler.go.tpl",
                Mode:       codegen.WriteCreate,
                Owner:      fmt.Sprintf("Worker:%s:kafka:%s", w.spec.Name, t.Name),
            })
        }
    }

    // === Variant: customized ===
    if w.hasVariant("customized") {
        pairs = append(pairs,
            w.tplPair("templates/component/worker/customized/registry.go.tpl",
                filepath.Join("internal", w.spec.Name, "customized/registry.go")),
        )
        for _, c := range w.spec.Customized {
            pairs = append(pairs, codegen.Pair{
                Dst:        filepath.Join("internal", w.spec.Name, "customized", c.Name+".go"),
                TemplateID: "templates/component/worker/customized/watcher.go.tpl",
                Mode:       codegen.WriteCreate,
                Owner:      fmt.Sprintf("Worker:%s:customized:%s", w.spec.Name, c.Name),
            })
        }
    }

    return pairs
}

func (w *Worker) BaseMutators(p *project.Project) []ast.ASTMutator {
    return nil
}

func (w *Worker) PostProcess(p *project.Project, fm FileSystem) error {
    return nil
}

func (w *Worker) hasVariant(v string) bool {
    for _, x := range w.spec.Variants {
        if x == v {
            return true
        }
    }
    return false
}

func (w *Worker) tplPair(tplID, dst string) codegen.Pair {
    return codegen.Pair{
        Dst:        dst,
        TemplateID: tplID,
        Mode:       codegen.WriteCreate,
        Owner:      "Worker:" + w.spec.Name,
    }
}
```

### 9.5.4 add worker 子流程

`linctl add worker <Name> --variant cron|kafka|customized` 的实现：

1. 把新条目加入 `c.Cron.Jobs` / `c.Kafka.Topics` / `c.Customized`。
2. 仅生成新条目的单个文件（如 `cron/<NewJob>.go`）。
3. **不需要 AST 注入**（registry 模式：在生成的 `registry.go` 里通过 `init()` 自动注册）。

## 9.6 内置组件 3：CLI

### 9.6.1 定位

`CLI` 是命令行工具组件，对应 osbuilder 的 CLITool。

典型用例：`mbctl get post`、`kubectl` 风格的运维工具。

### 9.6.2 配置示例

```yaml
- kind: CLI
  name: mbctl
  commands:
    - name: get
    - name: create
    - name: describe
    - name: version
```

### 9.6.3 实现

```go
// internal/component/cli.go
package component

import (
    "fmt"
    "path/filepath"

    "github.com/<org>/linctl/internal/ast"
    "github.com/<org>/linctl/internal/codegen"
    "github.com/<org>/linctl/internal/project"
)

type CLI struct {
    spec project.Component
}

func NewCLI(spec project.Component) (*CLI, error) {
    return &CLI{spec: spec}, nil
}

func (c *CLI) Kind() string { return "CLI" }
func (c *CLI) Name() string { return c.spec.Name }

func (c *CLI) Validate(p *project.Project) error {
    if len(c.spec.Commands) == 0 {
        return fmt.Errorf("cli %q: at least one command required", c.spec.Name)
    }
    return nil
}

func (c *CLI) BasePairs(p *project.Project) []codegen.Pair {
    pairs := []codegen.Pair{
        c.tplPair("templates/component/cli/cmd/main.go.tpl",
            filepath.Join("cmd", c.spec.Name, "main.go")),
        c.tplPair("templates/component/cli/internal/root.go.tpl",
            filepath.Join("internal", c.spec.Name, "cmd/root.go")),
        c.tplPair("templates/component/cli/internal/all.go.tpl",
            filepath.Join("internal", c.spec.Name, "cmd/all.go")),
    }
    for _, cmd := range c.spec.Commands {
        pairs = append(pairs, codegen.Pair{
            Dst:        filepath.Join("internal", c.spec.Name, "cmd", cmd.Name, cmd.Name+".go"),
            TemplateID: "templates/component/cli/internal/subcmd.go.tpl",
            Mode:       codegen.WriteCreate,
            Owner:      fmt.Sprintf("CLI:%s:%s", c.spec.Name, cmd.Name),
        })
    }
    return pairs
}

func (c *CLI) BaseMutators(p *project.Project) []ast.ASTMutator {
    // add cli 时给 all.go 加 _ "..."
    // 但 BaseMutators 在 new 流程也调用，需要幂等
    var muts []ast.ASTMutator
    allGoPath := filepath.Join("internal", c.spec.Name, "cmd/all.go")
    for _, cmd := range c.spec.Commands {
        muts = append(muts, &ast.AddImportMutator{
            File:       allGoPath,
            ImportPath: fmt.Sprintf("%s/internal/%s/cmd/%s", p.Metadata.Module, c.spec.Name, cmd.Name),
            Anonymous:  true,
        })
    }
    return muts
}

func (c *CLI) PostProcess(p *project.Project, fm FileSystem) error {
    return nil
}

func (c *CLI) tplPair(tplID, dst string) codegen.Pair {
    return codegen.Pair{
        Dst:        dst,
        TemplateID: tplID,
        Mode:       codegen.WriteCreate,
        Owner:      "CLI:" + c.spec.Name,
    }
}
```

### 9.6.4 add cli 子流程

`linctl add cli <name>` 实现：

1. 把新 command 加入 `c.Commands`。
2. 生成 `cmd/<name>/<name>.go` 单个文件。
3. AST 注入：在 `all.go` 加 `_ "<module>/internal/<cli>/cmd/<name>"`（幂等检测重复）。

## 9.7 多 Component 协作

### 9.7.1 共享存储

```yaml
spec:
  defaults:
    storage: gorm-postgres  # 项目级默认

  components:
    - kind: WebServer
      name: mb-apiserver
      # 继承 defaults.storage = gorm-postgres
    - kind: Worker
      name: mb-worker
      storage: gorm-postgres  # 显式声明
```

linctl 会确保这两个组件共享同一份 `internal/pkg/storage/` 模板（去重）。

### 9.7.2 共享 internal/pkg/

下列文件被多个 Component **共享**，去重逻辑由 PairBuilder 自动处理：

| 共享路径 | 贡献者 |
| --- | --- |
| `internal/pkg/errno/` | 所有组件 |
| `internal/pkg/storage/` | 所有用 storage 的组件 |
| `internal/pkg/log/` | 所有组件 |
| `internal/pkg/observability/` | 所有启用 OTel feature 的组件 |
| `internal/pkg/middleware/` | 所有用 user feature 的组件 |
| `pkg/api/` | 所有 WebServer |

冲突解决：

- 完全相同的 Pair（dst + tplID）→ 去重。
- dst 相同但 tplID 不同 → 后写覆盖 + warning。

### 9.7.3 跨组件命名约定

```
<module>/cmd/<component>/...                    # 每组件入口隔离
<module>/internal/<component>/...               # 每组件业务隔离
<module>/internal/pkg/...                       # 项目级共享
<module>/pkg/api/<webserver>/v1/...             # API 仅 WebServer 有
```

## 9.8 Component 测试策略

### 9.8.1 单测：每个组件覆盖

```go
// internal/component/webserver_test.go
func TestWebServer_BasePairs(t *testing.T) {
    cases := []struct {
        name      string
        spec      project.Component
        wantCount int
        contains  []string
    }{
        {
            name: "minimal gin",
            spec: project.Component{
                Kind: "WebServer", Name: "myblog",
                Framework: "gin", Storage: "memory",
            },
            wantCount: 8,
            contains: []string{
                "cmd/myblog/main.go",
                "internal/myblog/server.go",
                "internal/myblog/store/store.go",
            },
        },
        {
            name: "grpc with resources",
            spec: project.Component{
                Kind: "WebServer", Name: "myblog",
                Framework: "grpc", Storage: "gorm-postgres",
                Resources: []project.Resource{{Name: "post"}},
            },
            wantCount: 16, // 8 base + 8 per resource
            contains: []string{
                "pkg/api/myblog/v1/post.proto",
                "internal/myblog/handler/grpc/post.go",
                "internal/myblog/biz/v1/post/post.go",
            },
        },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ws, err := NewWebServer(tc.spec)
            require.NoError(t, err)
            pairs := ws.BasePairs(testProject(tc.spec))
            require.Len(t, pairs, tc.wantCount)
            paths := make([]string, len(pairs))
            for i, p := range pairs {
                paths[i] = p.Dst
            }
            for _, c := range tc.contains {
                require.Contains(t, paths, c)
            }
        })
    }
}
```

### 9.8.2 集成：与 Feature 协作

```go
// tests/integration/webserver_with_features_test.go
func TestWebServerWithFeatures(t *testing.T) {
    proj := loadProject(t, "fixtures/webserver-with-features.yaml")
    plan, err := orchestrator.Plan(context.Background(), proj)
    require.NoError(t, err)

    // 验证 healthz 贡献了 healthz.proto
    assert.True(t, planContains(plan, "pkg/api/myblog/v1/healthz.proto"))
    // 验证 user feature 贡献了 user middleware
    assert.True(t, planContains(plan, "internal/pkg/middleware/gin/authn.go"))
    // 验证 OTel mutator 给 server.go 加了 import
    assert.True(t, planHasMutator(plan, "internal/myblog/server.go", "AddImportMutator"))
}
```

### 9.8.3 E2E：真实 go build

```go
// tests/e2e/new_webserver_test.go
func TestNewWebServerCanBuild(t *testing.T) {
    tmpDir := t.TempDir()
    cmd := exec.Command("./linctl", "new", "myblog",
        "--module", "github.com/example/myblog",
        "--framework", "gin",
        "--storage", "memory",
    )
    cmd.Dir = tmpDir
    out, err := cmd.CombinedOutput()
    require.NoError(t, err, "linctl new failed: %s", out)

    // 真实 go build
    buildCmd := exec.Command("go", "build", "./...")
    buildCmd.Dir = filepath.Join(tmpDir, "myblog")
    out, err = buildCmd.CombinedOutput()
    require.NoError(t, err, "go build failed: %s", out)
}
```

## 9.9 第三方 Component（远期）

Phase 5+ 通过插件机制支持第三方 Component（如 `linctl-plugin-kratos` 提供 `KratosWebServer` 组件）。设计原则：

1. 插件通过 `linctl-plugin-*` 二进制 + stdio JSON-RPC 注册。
2. 插件 Component 必须遵循同样的 5 段式接口契约。
3. `BasePairs` / `BaseMutators` 通过 RPC 序列化为 `[]Pair` / `[]SerializedMutator`。
4. 用户在 `linctl.yaml` 中 `kind: Kratos` 即可使用。

> 详见 [08-feature-system.md §8.7](./08-feature-system.md)（插件协议）+ Phase 5 实施计划。

## 9.10 关键设计决策

| 决策 | 原因 | ADR |
| --- | --- | --- |
| 合并 JobServer + MQServer 为 Worker | 90% 共用代码；减少抽象数量 | 本文档 §9.1.2 |
| Component 接口仅 6 个方法 | 易于实现 + 易于插件化 | 本文档 §9.2.1 |
| Pair 携带 Owner 字段 | plan 报告可追溯，团队协作便于定位 | [ADR-004](./adr/004-plan-apply-pattern.md) |
| BasePairs/BaseMutators 是纯函数 | 单测友好，IO 由 L0 隔离 | 本文档 §9.2.1 |
| 用 `init()` + Registry 注册组件 | 与 Feature 一致 + 便于插件扩展 | [ADR-005](./adr/005-feature-as-first-class.md) |

## 9.11 Open Questions

| 问题 | 待决议 |
| --- | --- |
| 是否引入 `Component.Dependencies() []ComponentRef`，让组件之间显式声明依赖（如 Worker 依赖 WebServer 的 store）？ | Phase 3 评估 |
| `customized` variant 是否应升级为独立的 `Watcher` 组件？ | Phase 4 评估，目前合并为 Worker 简化心智 |
| 是否引入 `Singleton` 概念，限制某些 Component 全项目只能有一个（如 cli 组件）？ | Phase 2 评估 |
| 是否支持组件级 `enabled: false`，便于临时禁用？ | Phase 1 引入即可 |

---

## 修订记录

| 日期 | 版本 | 变更 |
| --- | --- | --- |
| 2026-04-25 | 0.1 | 初始版本 |

---

下一步阅读：[10-tech-stack.md](./10-tech-stack.md) → [11-implementation-plan.md](./11-implementation-plan.md)

_Last reviewed: 2026-04-25_
