# 04. 模板系统设计

> **前置阅读**：[01-architecture-blueprint.md](./01-architecture-blueprint.md)、[03-resource-scaffold.md](./03-resource-scaffold.md)
>
> 本文档定义重构后**简化版模板系统**：embed.FS 默认 + `--template-dir` 外部目录覆盖。

---

## 1. 设计目标

| 目标 | 实现 |
| --- | --- |
| 零配置开箱即用 | embed.FS 内置 miniblog-v4 风格全套模板 |
| 团队定制 | `--template-dir <dir>` 完全替换或部分覆盖 |
| 简单 | 仅 `text/template`，不引入 jinja/handlebars 等异构语法 |
| 可调试 | 模板渲染失败时打印**完整上下文**（模板、行号、变量） |
| 跨平台路径 | 模板内禁用 `\`，统一 `filepath.ToSlash` |

明确**不做**：

- ❌ 模板上游同步 / 远程拉取
- ❌ Manifest / Lockfile / Cache
- ❌ 模板版本号声明 / 兼容性矩阵
- ❌ 模板继承（block/extend）— 简单复用即可

---

## 2. 模板查找优先级

```
┌──────────────────────────────────────────────────────────┐
│  1. 命令行 --template-dir <abs-path>                      │ ← 最高优先级
└────────────────────────────┬─────────────────────────────┘
                             │ 未传或文件不存在 ↓
┌──────────────────────────────────────────────────────────┐
│  2. 项目根目录 ./.lin/templates/                           │ ← 项目级覆盖
└────────────────────────────┬─────────────────────────────┘
                             │ 未存在 ↓
┌──────────────────────────────────────────────────────────┐
│  3. 用户目录 ~/.lin/templates/                            │ ← 用户级覆盖
└────────────────────────────┬─────────────────────────────┘
                             │ 未存在 ↓
┌──────────────────────────────────────────────────────────┐
│  4. embed.FS（lin 二进制内置）                             │ ← 默认
└──────────────────────────────────────────────────────────┘
```

**规则**：

1. 按上述优先级从高到低查找；**只要某层存在该模板**，使用该层并停止。
2. **不会跨层合并**（如不会"上层只 override 一个文件，其他从下层取"）。如需局部覆盖，请整层复制后修改。
3. 模板路径必须**完全一致**（如 `resource/handler.go.tpl`）；命名错误会被忽略。

> **设计取舍**：放弃"细粒度合并覆盖"以简化心智模型。团队定制就完整 fork 一份模板目录。

---

## 3. 模板目录约定

### 3.1 标准结构（embed 与外部目录都遵守）

```
templates/                                # 根
├── project/                               # lin new 使用
│   ├── go.mod.tpl
│   ├── Makefile.tpl
│   ├── README.md.tpl
│   ├── .gitignore.tpl
│   ├── .golangci.yaml.tpl
│   ├── Dockerfile.tpl
│   ├── docker-compose.yml.tpl
│   ├── cmd/
│   │   ├── app/
│   │   │   ├── main.go.tpl
│   │   │   └── app/
│   │   │       └── server.go.tpl
│   │   └── gen-gorm-model/                # 数据库模型反推工具
│   │       └── gen_gorm_model.go.tpl
│   ├── internal/
│   │   ├── app/
│   │   │   ├── handler/
│   │   │   │   ├── handler.go.tpl
│   │   │   │   └── healthz.go.tpl
│   │   │   ├── biz/
│   │   │   │   └── biz.go.tpl           # 仅初始接口（不含资源方法）
│   │   │   ├── store/
│   │   │   │   └── store.go.tpl         # 仅初始接口
│   │   │   └── pkg/
│   │   │       ├── conversion/
│   │   │       │   └── .gitkeep.tpl
│   │   │       └── validation/
│   │   │           └── validator.go.tpl
│   │   └── pkg/
│   │       ├── errno/
│   │       │   ├── errno.go.tpl
│   │       │   └── register.go.tpl
│   │       ├── middleware/
│   │       │   └── middleware.go.tpl
│   │       ├── contextx/
│   │       │   └── contextx.go.tpl
│   │       └── known/
│   │           └── const.go.tpl
│   ├── pkg/
│   │   ├── api/
│   │   │   └── app/
│   │   │       └── v1/
│   │   │           └── app.proto.tpl
│   │   └── db/                            # 数据库连接器（PostgreSQL/MySQL/...）
│   │       ├── postgres.go.tpl
│   │       └── mysql.go.tpl
│   ├── configs/
│   │   └── app.yaml.tpl                   # 单一应用配置；含 db section 供 gen-gorm-model 复用
│   ├── scripts/
│   │   └── boot.sh.tpl
│   └── docs/
│       └── README.md.tpl
│
└── resource/                              # lin add 使用
    ├── handler.go.tpl
    ├── biz/
    │   ├── biz_iface.go.tpl              # 资源接口部分
    │   ├── biz_struct.go.tpl             # 资源工厂部分
    │   ├── verb_create.go.tpl
    │   ├── verb_update.go.tpl
    │   ├── verb_delete.go.tpl
    │   ├── verb_get.go.tpl
    │   └── verb_list.go.tpl
    ├── store.go.tpl
    ├── model.gen.go.tpl
    ├── conversion.go.tpl
    ├── validation.go.tpl
    ├── errno.go.tpl
    └── proto.tpl
```

### 3.2 模板路径转换规则

模板路径中**仅有一个固定占位符**，其它路径段由 `scaffold/render.go` 的 `Plan` 计算时拼出：

| 占位符 | 替换时机 | 替换为 |
| --- | --- | --- |
| `app/` 段 | `lin new` / `lin add` 渲染时 | `<AppName>/`（如 `myblog/`） |

资源路径**不**使用占位符；由代码根据 `Plan.FileSpec.DestPath` 直接拼接：

```
模板路径                                         输出路径
──────────────────────────────────────────       ──────────────────────────────────
templates/project/cmd/app/main.go.tpl       →   cmd/myblog/main.go
templates/project/internal/app/biz/biz.go.tpl → internal/myblog/biz/biz.go
templates/resource/biz/verb_create.go.tpl   →   internal/myblog/biz/v1/post/create.go
                                                                          ^^^^
                                                            由 Plan 拼接，不在模板路径中
```

**约定**：

1. 模板路径只能包含 `app/` 一种占位符（指向 `<AppName>/`）。
2. 资源相关路径段（如 `post/`、`post.go`）由 `scaffold.BuildPlan` 计算 `DestPath` 时拼接，**不在模板文件名中出现**。
3. 模板内**禁止使用 `{{.AppName}}` 在路径上下文**（即模板内容仍可用，但代码处理路径时不使用 Go template 引擎）。

> **设计取舍**：路径替换由代码处理而非 template 引擎，逻辑显式且更易测试。

---

## 4. 模板变量定义

### 4.1 项目级变量（`lin new` 时可用）

```go
type ProjectVars struct {
    // 项目身份
    Module    string  // github.com/foo/myblog
    AppName   string  // myblog
    AppNameU  string  // MYBLOG（uppercase 用于环境变量名）

    // 配置
    Framework string  // gin (MVP only)
    Storage   string  // memory | gorm-postgres | gorm-mysql | gorm-sqlite | mongo
    Features  []string // healthz, otel, user, swagger
    WithGRPC  bool
    WithDocker bool
    WithK8s   bool

    // 元数据
    Author    string
    Email     string
    Year      int     // 当前年份（License）
    GoVersion string  // 1.22

    // lin 自身
    LinVersion string
}
```

### 4.2 资源级变量（`lin add` 时可用）

```go
type ResourceVars struct {
    // 继承项目变量
    ProjectVars

    // 资源身份
    Resource     string  // Post（PascalCase）
    ResourceCamel string // post（camelCase = lowercase here）
    ResourceLower string // post
    ResourceSnake string // post（snake_case）
    ResourceKebab string // post
    ResourcePlural string // posts
    ResourcePluralKebab string // posts

    // 控制
    With []string  // [conversion, validation, proto]
    Ops  []string  // [create, update, delete, get, list]
}
```

---

## 5. FuncMap（模板可用函数）

```go
package tpl

import (
    "strings"
    "text/template"

    "github.com/iancoleman/strcase"
    "github.com/jinzhu/inflection"
)

func DefaultFuncs() template.FuncMap {
    return template.FuncMap{
        // 大小写转换
        "Pascal":      strcase.ToPascal,        // post → Post
        "Camel":       strcase.ToCamel,          // post → post
        "LowerCamel":  strcase.ToLowerCamel,     // Post → post
        "Snake":       strcase.ToSnake,          // PostItem → post_item
        "Kebab":       strcase.ToKebab,          // PostItem → post-item
        "Lower":       strings.ToLower,
        "Upper":       strings.ToUpper,
        "Title":       strings.Title,
        // 单复数
        "Plural":      inflection.Plural,         // post → posts
        "Singular":    inflection.Singular,       // posts → post
        // 字符串
        "Quote":       strconv.Quote,
        "TrimPrefix":  strings.TrimPrefix,
        "TrimSuffix":  strings.TrimSuffix,
        "Replace":     strings.ReplaceAll,
        // 集合
        // 注意：参数顺序为 (needle, slice)，匹配 Go template pipeline:
        //   {{ .Features | Has "user" }} → Has("user", .Features)
        "Has":         func(needle string, slice []string) bool {
                           for _, s := range slice {
                               if s == needle { return true }
                           }
                           return false
                      },
        "Join":        strings.Join,
        // 时间
        "Now":         func() string { return time.Now().Format(time.RFC3339) },
        "Year":        func() int { return time.Now().Year() },
        // 条件
        "Default":     func(d, v string) string { if v == "" { return d }; return v },
    }
}
```

> **不允许**模板内执行外部命令、读取环境变量、访问文件系统（安全考虑）。

---

## 6. 模板渲染示例

### 6.1 `internal/<app>/handler/healthz.go.tpl`

```go.tpl
package handler

import (
    "github.com/gin-gonic/gin"
    "{{ .Module }}/pkg/core"
)

func init() {
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        v1.GET("/healthz", h.Healthz)
    })
}

// Healthz 健康检查.
func (h *Handler) Healthz(c *gin.Context) {
    core.WriteResponse(c, nil, gin.H{
        "status":  "ok",
        "service": "{{ .AppName }}",
    })
}
```

### 6.2 `internal/<app>/store/store.go.tpl`（项目骨架阶段）

```go.tpl
package store

import (
    "context"
    "sync"

    "github.com/google/wire"
    "{{ .Module }}/pkg/store/where"
    "gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewStore, wire.Bind(new(IStore), new(*datastore)))

var (
    once sync.Once
    S    *datastore
)

// IStore 定义存储层方法集合.
// lin: inject-region:store-interface (do not remove this comment)
type IStore interface {
    DB(ctx context.Context, wheres ...where.Where) *gorm.DB
    TX(ctx context.Context, fn func(ctx context.Context) error) error
    {{- if .Features | Has "user" }}
    User() UserStore
    {{- end }}
}
// lin: inject-region-end

// datastore 是 IStore 的具体实现.
type datastore struct {
    core *gorm.DB
}

var _ IStore = (*datastore)(nil)

func NewStore(db *gorm.DB) *datastore {
    once.Do(func() { S = &datastore{db} })
    return S
}

func (s *datastore) DB(ctx context.Context, wheres ...where.Where) *gorm.DB {
    db := s.core
    for _, w := range wheres {
        db = w.Where(db)
    }
    return db
}

func (s *datastore) TX(ctx context.Context, fn func(ctx context.Context) error) error {
    return s.core.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        return fn(context.WithValue(ctx, transactionKey{}, tx))
    })
}

type transactionKey struct{}

// lin: inject-region:store-impl (do not remove this comment)
{{- if .Features | Has "user" }}
func (s *datastore) User() UserStore { return newUserStore(s) }
{{- end }}
// lin: inject-region-end
```

> **关键点**：`// lin: inject-region:` 锚点注释由 AST 注入识别（详见 [05-registration-strategy.md](./05-registration-strategy.md) §3.1）；模板初始化时根据 features 决定是否包含 `User()`。

---

## 7. 外部模板目录使用示例

### 7.1 完整 fork 模板（MVP 推荐方式）

MVP 不提供 `dump` 子命令；用户通过以下任一方式获取模板基线：

```bash
# 方式 A：从 lin 源码 clone 后复制
git clone https://github.com/<org>/lin.git /tmp/lin
cp -r /tmp/lin/internal/templates ./my-templates

# 方式 B：直接 sparse-checkout 仅取 templates 目录
git clone --depth=1 --filter=blob:none --sparse https://github.com/<org>/lin.git
cd lin && git sparse-checkout set internal/templates
mv internal/templates ../my-templates && cd ..

# 修改 my-templates/...

# 使用自定义模板
lin new myblog --module github.com/foo/myblog --template-dir ./my-templates
```

> **设计取舍**：MVP 不内置 `dump` 子命令，避免命令膨胀。如未来证实有强需求，作为 Phase 2+ 增量评估。

### 7.2 项目级覆盖

```bash
# 在项目根目录建一个 .lin/templates/ 子目录覆盖
mkdir -p .lin/templates/resource
cp ~/lin-source/internal/templates/resource/handler.go.tpl \
   .lin/templates/resource/handler.go.tpl
# 修改后...
lin add Post   # 自动使用 .lin/templates/resource/handler.go.tpl
```

### 7.3 用户级覆盖

```bash
# 全局覆盖（所有项目）
mkdir -p ~/.lin/templates
# 放入团队风格的整套模板
lin new myblog --module github.com/foo/myblog
```

---

## 8. 渲染失败时的诊断

模板渲染失败时，错误信息必须包含：

```
✗ Template render failed
  Template:  resource/biz/verb_create.go.tpl
  Output:    internal/myblog/biz/v1/post/create.go
  Error:     template: verb_create.go.tpl:14:18:
             executing "verb_create.go.tpl" at <.Resource | UndefinedFn>:
             function "UndefinedFn" not defined

  Variables (excerpt):
    Module:   github.com/foo/myblog
    AppName:  myblog
    Resource: Post
    With:     [conversion, validation, proto]

  Hint: Available functions: Pascal, Camel, Snake, Kebab, Lower, Upper, Plural, ...
        See: lin/docs/features/04-template-system.md §5
```

---

## 9. 模板加载实现（伪代码）

### 9.1 embed 路径约定

模板源在仓库内位于 `lin/internal/templates/`，因此 `embed` 指令位于 `lin/internal/templates/embed.go`：

```go
// 文件：lin/internal/templates/embed.go
package templates

import "embed"

// FS 内嵌的模板文件系统。
// 注意：embed 路径相对于本 .go 文件所在目录，不含 internal/ 前缀。
//go:embed all:project all:resource
var FS embed.FS
```

### 9.2 Loader 实现

```go
// 文件：lin/internal/pkg/tpl/loader.go
package tpl

import (
    "embed"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "text/template"

    "github.com/<org>/lin/internal/templates"
    "github.com/<org>/lin/internal/pkg/fsx"
)

type Loader struct {
    overrideDirs []string  // 按优先级排序（高优先级在前）
}

func NewLoader(opts Options) (*Loader, error) {
    var dirs []string

    // 1. 命令行 --template-dir
    if opts.TemplateDir != "" {
        abs, err := filepath.Abs(opts.TemplateDir)
        if err != nil {
            return nil, fmt.Errorf("template-dir abs: %w", err)
        }
        if info, err := os.Stat(abs); err == nil && info.IsDir() {
            dirs = append(dirs, abs)
        }
    }

    // 2. 项目级覆盖：./.lin/templates/
    if opts.ProjectRoot != "" {
        p := filepath.Join(opts.ProjectRoot, ".lin", "templates")
        if info, err := os.Stat(p); err == nil && info.IsDir() {
            dirs = append(dirs, p)
        }
    }

    // 3. 用户级覆盖：~/.lin/templates/
    if home, err := os.UserHomeDir(); err == nil {
        p := filepath.Join(home, ".lin", "templates")
        if info, err := os.Stat(p); err == nil && info.IsDir() {
            dirs = append(dirs, p)
        }
    }

    return &Loader{overrideDirs: dirs}, nil
}

// Load 按优先级查找并解析模板。relPath 形如 "project/cmd/app/main.go.tpl".
func (l *Loader) Load(relPath string) (*template.Template, error) {
    // 关键安全：规范化路径并防御穿越
    rel := filepath.ToSlash(filepath.Clean(relPath))
    if strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
        return nil, fmt.Errorf("template path traversal: %s", relPath)
    }

    for _, dir := range l.overrideDirs {
        full, err := fsx.SafeJoin(dir, rel)
        if err != nil { continue }
        if info, err := os.Stat(full); err == nil && !info.IsDir() {
            return parseFromFile(full)
        }
    }
    return parseFromEmbed(rel)
}

func parseFromFile(path string) (*template.Template, error) {
    raw, err := os.ReadFile(path)
    if err != nil { return nil, err }
    return template.New(filepath.Base(path)).Funcs(DefaultFuncs()).Parse(string(raw))
}

func parseFromEmbed(relPath string) (*template.Template, error) {
    raw, err := fs.ReadFile(templates.FS, relPath)
    if err != nil { return nil, fmt.Errorf("embed template not found: %s", relPath) }
    return template.New(filepath.Base(relPath)).Funcs(DefaultFuncs()).Parse(string(raw))
}
```

### 9.3 关键约束

| 约束 | 实现 |
| --- | --- |
| `embed` 路径正确 | `//go:embed all:project all:resource`（相对模板包） |
| 路径穿越防护 | `Load` 入口 `..` 检查；`fsx.SafeJoin` 二次校验 |
| 模板包独立 | `internal/templates/` 仅存模板 + `embed.go`，不含业务代码 |
| 渲染产物路径校验 | 由 `scaffold/render.go` 在写入前调用 `fsx.SafeJoin(rootDir, dest)`，详见 [05 §2.5](./05-registration-strategy.md) |

---

## 10. 模板编写规范

| # | 规则 | 例 |
| --- | --- | --- |
| 1 | 文件后缀统一 `.tpl`，最终输出去掉 `.tpl` | `handler.go.tpl` → `handler.go` |
| 2 | 路径中不使用 `{{ }}`，由 lin 代码替换 | `cmd/app/main.go.tpl` → `cmd/myblog/main.go` |
| 3 | Feature 控制用 `{{- if .Features \| Has "user" }}` | 不要拼接字符串 |
| 4 | 锚点注释统一格式：`// lin: inject-region:<name>` | 见 §6.2 |
| 5 | 不允许模板内 `os.Exec` / 读环境变量 | 安全 |
| 6 | 不允许包含作者本机路径 | 跨用户复用 |
| 7 | 模板必须能在 macOS / Linux / Windows 渲染出一致结果 | 路径分隔符用 `filepath.ToSlash` |
| 8 | 模板内**禁止** `os.Open` / `os.ReadFile` / `exec.Command` | 仅纯字符串操作 |
| 9 | FuncMap 仅暴露纯函数，不允许 IO | 见 §5 funcMap 清单 |

### 10.1 路径穿越（Path Traversal）防护

由于 `--template-dir` 与 `~/.lin/templates/` 来自用户控制，**必须**强制验证：

| 校验点 | 实现 | 失败行为 |
| --- | --- | --- |
| 模板查找路径 | `Load(relPath)` 内 `..` 检查 + `fsx.SafeJoin` | `error` exit 43 |
| 渲染产物路径 | `Render(dest)` 前 `fsx.SafeJoin(rootDir, dest)` | `error` exit 43 |
| 模板名包含 `/` 反斜杠 | 不允许；统一 `filepath.ToSlash` | `error` exit 41 |
| 渲染产物路径绝对路径 | 不允许；必须相对 `rootDir` | `error` exit 43 |

```go
// fsx/safe_path.go 关键实现
func SafeJoin(root, rel string) (string, error) {
    abs, err := filepath.Abs(filepath.Join(root, rel))
    if err != nil { return "", err }

    rootAbs, err := filepath.Abs(root)
    if err != nil { return "", err }

    if !strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) && abs != rootAbs {
        return "", &errs.Error{
            Code:    errs.CodeTplPathTraversal,
            Message: fmt.Sprintf("path %q escapes root %q", rel, root),
        }
    }
    return abs, nil
}
```

### 10.2 模板内容审计

CI 阶段对内置模板（`internal/templates/`）做静态扫描，禁止以下模式：

```
禁止 pattern                                  允许
─────────────────────────────────────         ────────────────────────
{{ env "..." }}                               {{ .Module }}
{{ exec "..." }}                              {{ Pascal .Resource }}
/Users/...                                   {{ .Author }}
$HOME / ${HOME}                              ~/.bashrc（仅文档/注释）
```

---

## 11. 模板生命周期

| 阶段 | 操作 |
| --- | --- |
| 修改模板 | 直接编辑 `internal/templates/...` |
| 测试模板 | `go test ./internal/scaffold/...`（含模板渲染断言） |
| 升级模板 | 修改后发布新版 lin；**不影响已生成项目** |
| 团队定制 | fork lin 仓库或仅 fork templates 目录 |

> **关键**：lin 不维护"模板版本号"、不跟踪"已生成项目用了哪个版本"；模板升级与已生成项目**完全解耦**。

---

## 12. wire 在生成项目中的角色

### 12.1 wire 仅在**生成项目**中存在，不在 lin 自身依赖

| 维度 | lin 工具 | 生成的项目（如 `myblog`） |
| --- | --- | --- |
| 是否依赖 wire | ❌ 不依赖 | ✅ 依赖 `github.com/google/wire` |
| 用途 | n/a | 生成 `wire_gen.go` 完成 DI 装配 |
| 依赖入口 | n/a | 生成项目的 `go.mod` 含 wire；`internal/<app>/biz/biz.go` 等含 `var ProviderSet = wire.NewSet(...)` |

`lin` 自身的依赖清单（≤ 8 个）见 [01 §9 关键非功能需求](./01-architecture-blueprint.md#9-关键非功能需求)；**wire 不在其中**。

### 12.2 模板中 `wire.NewSet` 的用法

`internal/<app>/biz/biz.go.tpl` 与 `store.go.tpl` 默认包含 `ProviderSet`：

```go.tpl
var ProviderSet = wire.NewSet(NewBiz, wire.Bind(new(IBiz), new(*biz)))
```

**用户工作流**：

| 阶段 | 操作 | 是否 lin 介入 |
| --- | --- | --- |
| `lin new` | 模板包含 `ProviderSet` 与 `wire.go`（手写的 `+build wireinject` 入口） | ✅ |
| `lin add Post` | 自动改 `biz.go` 接口与工厂；**不修改 wire.go**（`ProviderSet` 已经覆盖新方法） | ✅ AST 注入 |
| 用户首次构建 | 必须手动跑 `make wire` 或 `wire ./...` 生成 `wire_gen.go` | ❌ lin 不调用 |
| 后续 `lin add` | 同上；`ProviderSet` 自动覆盖；用户重跑 `make wire` 即可 | ❌ |

### 12.3 wire 与 ProviderSet 的注入策略

由于 `wire.NewSet` 接受函数引用，**新增方法不需要修改 ProviderSet**——只要新方法被绑定到 `IBiz` 接口，`wire_gen.go` 重新生成时即可识别。

```go
// 加 PostV1() 之前
var ProviderSet = wire.NewSet(NewBiz, wire.Bind(new(IBiz), new(*biz)))

// 加 PostV1() 之后（无变化）
var ProviderSet = wire.NewSet(NewBiz, wire.Bind(new(IBiz), new(*biz)))
```

> 因此 `lin add` **无需** AST 注入 `ProviderSet`；只注入 `IBiz` 接口与 `*biz` 工厂方法（[05 §3.1](./05-registration-strategy.md#31-mutator_interfacego---接口扩展)）。

### 12.4 next-step 提示用户跑 wire

`lin add` 输出的 `📦 Next steps`（[02 §4.7](./02-command-set.md#47-输出示例)）包含：

```
📦 Next steps:
   make protoc       # 生成 .pb.go
   make wire         # 重新生成 wire_gen.go ← 关键
   go mod tidy
   go build ./...
```

---

## 13. 跨平台与文件类型

### 13.1 行尾符（Line Endings）策略

| 阶段 | 策略 |
| --- | --- |
| 模板源（仓库内 `.tpl` 文件） | **统一 LF**，`.gitattributes` 强制 |
| 渲染中（内存中字符串） | LF（与模板源一致） |
| 写入磁盘（`.go` 输出） | **统一 LF**（无论 OS） |
| 写入磁盘（`Makefile`/`*.sh`） | LF |
| 写入磁盘（`.bat`/`.cmd`） | CRLF（Windows 批处理需要） |
| 写入磁盘（其他文本文件） | LF |

**理由**：

- Go 工具链（`gofmt`/`go test`）在 Windows 也使用 LF；统一 LF 避免文件 hash 不稳定
- `.gitattributes` 配合 `* text=auto eol=lf` 保证仓库内一致

`lin new` 自动生成 `.gitattributes`：

```gitattributes
* text=auto eol=lf
*.bat text eol=crlf
*.cmd text eol=crlf
```

### 13.2 BOM 处理

| 输入 | 处理 |
| --- | --- |
| 模板源含 BOM (`\ufeff`) | 加载时自动 strip |
| 用户外部模板含 BOM | 同上自动 strip |
| 渲染产物 | **永不**写入 BOM |

实现位置：`pkg/tpl/loader.go` 的 `parseFromFile` / `parseFromEmbed`。

### 13.3 文件类型与处理方式

模板目录可包含多种文件类型；**仅 `.tpl` 后缀**走 template 渲染：

| 后缀 | 处理 | 例子 |
| --- | --- | --- |
| `.tpl` | text/template 渲染，输出去掉 `.tpl` | `Makefile.tpl` → `Makefile` |
| 其他文本文件 | **按位拷贝** | `LICENSE` → `LICENSE` |
| 二进制文件 | **按位拷贝** | `assets/logo.png` → `assets/logo.png` |
| 隐藏文件（`.gitignore.tpl` 等） | 按 `.tpl` 规则；输出去掉 `.tpl` 后保留 dot | `.gitignore.tpl` → `.gitignore` |

> **设计取舍**：仅 `.tpl` 一种渲染后缀，避免心智负担。如需输出 `.go` 文件，直接命名为 `*.go.tpl`（如 `main.go.tpl` → `main.go`）。

**判定逻辑**（`scaffold/render.go`）：

```go
const tplSuffix = ".tpl"

if strings.HasSuffix(name, tplSuffix) {
    renderTemplate(src, strings.TrimSuffix(dst, tplSuffix), vars)
} else {
    copyFileBytes(src, dst)
}
```

> 使用 `strings.TrimSuffix` 而不是 `dst[:len(dst)-4]`，避免硬编码长度。

### 13.4 文件权限

| 文件类型 | 权限（生成时） |
| --- | --- |
| 普通源码（`.go` / `.proto` / `.yaml`） | `0644` |
| 脚本（`.sh` / `.bash`） | `0755` |
| 二进制（嵌入资源） | `0644` |
| 目录 | `0755` |

模板内可在 sidecar `*.perm` 文件指定（MVP 阶段不实现，按上述默认）。

---

_Last reviewed: 2026-04-29_
