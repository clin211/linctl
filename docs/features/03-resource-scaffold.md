# 03. 资源骨架规范

> **前置阅读**：[01-architecture-blueprint.md](./01-architecture-blueprint.md) §8
>
> 本文档定义 `linctl add <Resource>` 命令生成的**全栈资源骨架**——文件清单、命名约定、模板预览、AST 注入点。

---

## 1. 资源分层总览

以 `linctl add Post` 在 `myblog/` 项目下生成为例：

```
myblog/
├── cmd/myblog/
│   └── main.go
├── internal/
│   ├── myblog/
│   │   ├── handler/
│   │   │   ├── handler.go                      # ← linctl new 已创建
│   │   │   ├── healthz.go                      # ← linctl new 已创建
│   │   │   └── post.go                         # ✨ linctl add 创建
│   │   ├── biz/
│   │   │   ├── biz.go                          # ✏ AST 注入：PostV1()
│   │   │   └── v1/
│   │   │       └── post/                       # ✨ linctl add 创建该子目录
│   │   │           ├── post.go                 # 接口 + 工厂
│   │   │           ├── create.go               # Create 方法
│   │   │           ├── update.go               # Update 方法
│   │   │           ├── delete.go               # Delete 方法
│   │   │           ├── get.go                  # Get 方法
│   │   │           └── list.go                 # List 方法
│   │   ├── store/
│   │   │   ├── store.go                        # ✏ AST 注入：Posts()
│   │   │   └── post.go                         # ✨ linctl add 创建
│   │   ├── model/
│   │   │   └── post.gen.go                     # ✨ linctl add 创建（占位）
│   │   └── pkg/
│   │       ├── conversion/
│   │       │   └── post.go                     # ✨ linctl add 创建
│   │       └── validation/
│   │           └── post.go                     # ✨ linctl add 创建
│   └── pkg/
│       ├── errno/
│       │   ├── errno.go                        # ← linctl new 已创建
│       │   ├── register.go                     # ✏ AST 注入：RegisterErrors
│       │   └── post.go                         # ✨ linctl add 创建
│       └── ...
└── pkg/
    └── api/
        └── myblog/
            └── v1/
                ├── myblog.proto                # ✏ AST 注入：import "post.proto"
                └── post.proto                  # ✨ linctl add --with proto 创建
```

**总览**：`linctl add Post`（默认 `--with conversion,validation,proto,errno`）会：

- ✨ **创建 13 个文件**：1 handler + 6 biz 文件（1 接口 + 5 动词）+ 1 store + 1 model + 1 conversion + 1 validation + 1 errno + 1 proto = **1+6+1+1+1+1+1+1 = 13**
- ✏ **AST 注入 4 个中央文件**：`biz.go` / `store.go` / `<app>.proto` / `errno/register.go`

---

## 2. 命名约定

| 概念 | 大小写 | 例子 |
| --- | --- | --- |
| 资源名（命令参数） | **PascalCase** | `Post`、`UserProfile`、`OrderItem` |
| 包名（biz/v1/、handler 内部） | **lowercase**（去除驼峰） | `post`、`userprofile`、`orderitem` |
| 文件名 | **lowercase**（与包名一致） | `post.go`、`userprofile.go` |
| 类型名（Biz/Store 接口） | **PascalCase + Suffix** | `PostBiz`、`PostStore` |
| 方法名（biz.go 接口） | **PascalCase + V1** | `PostV1() postv1.PostBiz` |
| 方法名（store.go 接口） | **PascalCase（复数）** | `Posts() PostStore` |
| 方法名（handler） | **动词 + PascalCase** | `CreatePost`、`UpdatePost`、`DeletePost` |
| URL 路径 | **lowercase + 连字符 + 复数** | `/v1/posts`、`/v1/user-profiles` |

### 2.1 模板内提供的转换函数

```go
// internal/pkg/tpl/funcs.go
funcMap := template.FuncMap{
    "Pascal":   strcase.ToPascal,    // Post / UserProfile
    "Camel":    strcase.ToCamel,     // post / userProfile
    "Snake":    strcase.ToSnake,     // post / user_profile
    "Kebab":    strcase.ToKebab,     // post / user-profile
    "Lower":    strings.ToLower,     // post / userprofile
    "Plural":   inflection.Plural,   // posts / user_profiles
    "PluralKebab": ...                // posts / user-profiles
}
```

模板内调用：

```go
// {{ .Resource }} = "Post"
package {{ .Resource | Lower }}                     // package post

func New{{ .Resource }}Biz() *{{ .Resource | Camel }}Biz {  // func NewPostBiz() *postBiz
    ...
}
```

---

## 3. 文件清单（默认生成 12 个 + 注入 4 个）

### 3.1 Handler 层（1 个文件）

#### `internal/<app>/handler/post.go`

```go
package handler

import (
    "github.com/<module>/pkg/core"
    "github.com/gin-gonic/gin"
)

// init 注册路由到 handler 全局表（约定式：无需 AST 注入）
func init() {
    Register(func(v1 *gin.RouterGroup, handler *Handler) {
        rg := v1.Group("/posts")
        rg.Use(handler.mws...)
        rg.POST("",        handler.CreatePost)
        rg.PUT(":postID",  handler.UpdatePost)
        rg.DELETE(":postID", handler.DeletePost)
        rg.GET(":postID",  handler.GetPost)
        rg.GET("",         handler.ListPost)
    })
}

// CreatePost 创建 Post.
func (h *Handler) CreatePost(c *gin.Context) {
    core.HandleJSONRequest(c, h.biz.PostV1().Create, h.val.ValidateCreatePostRequest)
}

// UpdatePost 更新 Post.
func (h *Handler) UpdatePost(c *gin.Context) {
    core.HandleJSONRequest(c, h.biz.PostV1().Update, h.val.ValidateUpdatePostRequest)
}

// DeletePost 删除 Post.
func (h *Handler) DeletePost(c *gin.Context) {
    core.HandleUriRequest(c, h.biz.PostV1().Delete, h.val.ValidateDeletePostRequest)
}

// GetPost 获取 Post.
func (h *Handler) GetPost(c *gin.Context) {
    core.HandleUriRequest(c, h.biz.PostV1().Get, h.val.ValidateGetPostRequest)
}

// ListPost 列出 Post.
func (h *Handler) ListPost(c *gin.Context) {
    core.HandleQueryRequest(c, h.biz.PostV1().List, h.val.ValidateListPostRequest)
}
```

> **关键点**：handler 通过 `init() + Register()` **约定式**注册到全局，无需 AST 注入到任何中央文件。

---

### 3.2 Biz 层（6 个文件：接口 1 个 + 动词 5 个）

#### `internal/<app>/biz/v1/post/post.go`

```go
package post

import (
    "github.com/<module>/internal/<app>/store"
    "github.com/<module>/pkg/authz"
)

// PostBiz 定义 Post 业务方法集合.
type PostBiz interface {
    Create(ctx context.Context, req *v1.CreatePostRequest) (*v1.CreatePostResponse, error)
    Update(ctx context.Context, req *v1.UpdatePostRequest) (*v1.UpdatePostResponse, error)
    Delete(ctx context.Context, req *v1.DeletePostRequest) (*v1.DeletePostResponse, error)
    Get(ctx context.Context, req *v1.GetPostRequest) (*v1.GetPostResponse, error)
    List(ctx context.Context, req *v1.ListPostRequest) (*v1.ListPostResponse, error)
}

// postBiz 是 PostBiz 的具体实现.
type postBiz struct {
    store store.IStore
    authz *authz.Authz
}

// 编译期断言.
var _ PostBiz = (*postBiz)(nil)

// New 创建 PostBiz 实例.
func New(s store.IStore, a *authz.Authz) *postBiz {
    return &postBiz{store: s, authz: a}
}
```

#### `internal/<app>/biz/v1/post/create.go`

```go
package post

import (
    "context"

    v1 "github.com/<module>/pkg/api/<app>/v1"
)

// Create 创建 Post.
func (b *postBiz) Create(ctx context.Context, req *v1.CreatePostRequest) (*v1.CreatePostResponse, error) {
    // TODO: 实现 Post 创建逻辑
    // 1. 入参转模型
    // 2. 调用 store
    // 3. 模型转响应
    return &v1.CreatePostResponse{}, nil
}
```

`update.go` / `delete.go` / `get.go` / `list.go` 类似，每个文件一个方法 + TODO 占位。

> **设计意图**：动词文件分离让单个文件保持小（≤ 50 行），便于代码评审与协作。

---

### 3.3 Store 层（1 个文件）

#### `internal/<app>/store/post.go`

```go
package store

import (
    "context"

    "github.com/<module>/internal/<app>/model"
    "github.com/<module>/pkg/store/where"
    genericstore "github.com/<module>/pkg/store"
)

// PostStore 定义 Post 存储方法集合.
type PostStore interface {
    Create(ctx context.Context, post *model.PostM) error
    Update(ctx context.Context, post *model.PostM) error
    Delete(ctx context.Context, opts *where.Options) (int64, error)
    Get(ctx context.Context, opts *where.Options) (*model.PostM, error)
    List(ctx context.Context, opts *where.Options) (int64, []*model.PostM, error)
}

// postStore PostStore 实现.
type postStore struct {
    *genericstore.Store[model.PostM]
}

var _ PostStore = (*postStore)(nil)

// newPostStore 创建 PostStore 实例.
func newPostStore(store *datastore) *postStore {
    return &postStore{
        Store: genericstore.NewStore[model.PostM](store, nil),
    }
}
```

> **关键点**：基于 `genericstore.Store[T]` 泛型存储，自动获得 CRUD 默认实现，资源 `*Store` 仅做特化（如关联查询）。

---

### 3.4 Model 层（1 个占位 + 由 `cmd/gen-gorm-model/` 自动覆盖）

#### `internal/<app>/model/post.gen.go`（占位 → 待 gorm gen 替换）

```go
// Code generated placeholder by lin. Replace with `gorm gen` output via:
//   make gen-model
//go:generate go run {{ .Module }}/cmd/gen-gorm-model
package model

import "time"

// PostM 是 Post 的数据库映射模型（占位）.
//
// ⚠️ 此文件为 lin 生成的占位代码，仅用于让生成项目立即可编译。
// 推荐工作流：
//   1. 在数据库中创建对应表（DDL 自管理）
//   2. 跑 `make gen-model` 调用 cmd/gen-gorm-model 自动反推真实 model
//   3. lin 生成的占位会被覆盖；用户 commit gorm gen 产出
//
// 详见 §3.4.1 推荐工作流。
type PostM struct {
    ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    PostID    string    `gorm:"column:post_id;uniqueIndex;type:varchar(35);not null" json:"postID"`
    Title     string    `gorm:"column:title;type:varchar(255);not null" json:"title"`
    Content   string    `gorm:"column:content;type:text;not null" json:"content"`
    CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
    UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName 指定 PostM 对应的数据库表名.
func (PostM) TableName() string {
    return "post"
}
```

#### 3.4.1 推荐工作流：搭配 `cmd/gen-gorm-model/`

`linctl new` **默认生成** `cmd/gen-gorm-model/gen_gorm_model.go`（基于 GORM Gen 的标准模型生成器）：

```
<project-root>/
├── cmd/
│   ├── <app>/                        # 业务二进制
│   │   └── main.go
│   └── gen-gorm-model/                # ✨ linctl new 默认生成
│       └── gen_gorm_model.go          # 调用 gorm.io/gen 反推 model
└── Makefile                           # 含 gen-model target
```

`gen_gorm_model.go` 关键片段（模板）。**复用 `configs/<app>.yaml` 的 db section，不引入额外 db.yaml**：

```go.tpl
package main

import (
    "log"
    "os"
    "path/filepath"

    "{{ .Module }}/pkg/db"
    "github.com/spf13/pflag"
    "gopkg.in/yaml.v3"
    "gorm.io/gen"
)

// AppConfig 仅声明 gen-gorm-model 需要的子集（db + model 输出路径）.
// 与 internal/<app>/server 启动时使用的完整配置共享同一 yaml。
type AppConfig struct {
    DB struct {
        Postgres db.PostgreSQLOptions `yaml:"postgres"`
        // MySQL / SQLite 子结构按需扩展
    } `yaml:"db"`
}

// 命令行参数（默认值可被 flag 覆盖）.
var (
    configPath = "../../configs/{{ .AppName }}.yaml"
    modelPath  = "../../internal/{{ .AppName }}/model"
)

func main() {
    pflag.StringVar(&configPath, "config",     configPath, "Path to {{ .AppName }}.yaml")
    pflag.StringVar(&modelPath,  "model-path", modelPath,  "Output path for generated models")
    pflag.Parse()

    cfg := mustLoadConfig(configPath)
    dbInst, err := db.NewPostgreSQL(&cfg.DB.Postgres)
    if err != nil { log.Fatalf("connect db: %v", err) }

    abs, _ := filepath.Abs(modelPath)
    g := gen.NewGenerator(gen.Config{
        Mode:              gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
        ModelPkgPath:      abs,
        WithUnitTest:      false,
        FieldNullable:     true,
        FieldSignable:     false,
        FieldWithIndexTag: false,
        FieldWithTypeTag:  false,
    })
    g.UseDB(dbInst)
    applyOptions(g)
    g.GenerateAllTable()  // 默认：自动反推所有表
    g.Execute()
}

func mustLoadConfig(path string) *AppConfig {
    raw, err := os.ReadFile(path)
    if err != nil { log.Fatalf("read config %q: %v", path, err) }
    var c AppConfig
    if err := yaml.Unmarshal(raw, &c); err != nil {
        log.Fatalf("parse config: %v", err)
    }
    return &c
}
```

**`configs/<app>.yaml` 的 db section**（与运行时配置共享）：

```yaml
# configs/{{ .AppName }}.yaml
server:
  http:
    addr: ":8080"

db:
  postgres:
    addr: localhost:5432
    username: postgres
    password: postgres
    database: {{ .AppName }}
```

> **设计取舍**：单一配置文件减少认知负担；gen-gorm-model 仅 `Unmarshal` 所需子集，不影响主程序的完整 schema。

**生成的 Makefile** 含 target：

```makefile
.PHONY: gen-model
gen-model: ## 反推数据库 model（覆盖 internal/<app>/model/*.gen.go）
	cd cmd/gen-gorm-model && go run .

# build target 显式排除 cmd/gen-gorm-model（避免被打入业务二进制）
COMMANDS ?= $(filter-out $(PROJ_ROOT_DIR)/cmd/gen-gorm-model, \
                          $(filter-out %.md, $(wildcard $(PROJ_ROOT_DIR)/cmd/*)))
```

#### 3.4.2 行为约定

| 维度 | 决策 |
| --- | --- |
| `cmd/gen-gorm-model/` 由谁生成 | ✅ `linctl new` 默认生成 |
| 占位 `*.gen.go` 由谁生成 | ✅ `linctl add <Resource>` 生成（让 `go build` 立即通过） |
| `gorm.io/gen` 依赖 | ✅ 自动加入生成项目的 `go.mod`（非 lin 自身依赖） |
| `samber/lo` 依赖 | ✅ 同上（用于 JSON 标签 camelCase 转换） |
| `pkg/db` | ✅ 由 `linctl new` 生成（包含 `NewPostgreSQL` 等连接器） |
| 重复 `linctl add` 时 placeholder 处理 | 默认 skip（[02 §4.6](./02-command-set.md#46-幂等性与文件冲突策略)）；保留用户 gorm gen 产出 |
| `make gen-model` 后产物归属 | 用户提交（替代 lin 的占位） |
| `cmd/gen-gorm-model` 排除 build | ✅ Makefile 显式 filter-out |

#### 3.4.3 与 lin 自身依赖的边界

`gorm.io/gen` / `samber/lo` / `gorm` 仅出现在**生成项目的 go.mod**，**不**进 lin 自身依赖。

lin 工具自身仍保持 ≤ 8 个直接依赖（详见 [01 §9](./01-architecture-blueprint.md#9-关键非功能需求)）。

---

### 3.5 Conversion 层（1 个文件，可选）

#### `internal/<app>/pkg/conversion/post.go`

```go
package conversion

import (
    "github.com/<module>/internal/<app>/model"
    v1 "github.com/<module>/pkg/api/<app>/v1"
)

// PostMToProto 将 Model 转 Proto.
func PostMToProto(m *model.PostM) *v1.PostInfo {
    return &v1.PostInfo{
        PostID:    m.PostID,
        Title:     m.Title,
        Content:   m.Content,
        CreatedAt: m.CreatedAt.Unix(),
        UpdatedAt: m.UpdatedAt.Unix(),
    }
}

// PostProtoToM 将 Proto 转 Model.
func PostProtoToM(p *v1.PostInfo) *model.PostM {
    return &model.PostM{
        PostID:  p.PostID,
        Title:   p.Title,
        Content: p.Content,
    }
}
```

---

### 3.6 Validation 层（1 个文件，可选）

#### `internal/<app>/pkg/validation/post.go`

```go
package validation

import (
    "context"

    v1 "github.com/<module>/pkg/api/<app>/v1"
)

// ValidateCreatePostRequest 校验 CreatePost 请求.
func (v *Validator) ValidateCreatePostRequest(ctx context.Context, req *v1.CreatePostRequest) error {
    // TODO: 校验 req.Title / req.Content 等字段
    return nil
}

// ValidateUpdatePostRequest 校验 UpdatePost 请求.
func (v *Validator) ValidateUpdatePostRequest(ctx context.Context, req *v1.UpdatePostRequest) error {
    return nil
}

// ValidateDeletePostRequest 校验 DeletePost 请求.
func (v *Validator) ValidateDeletePostRequest(ctx context.Context, req *v1.DeletePostRequest) error {
    return nil
}

// ValidateGetPostRequest 校验 GetPost 请求.
func (v *Validator) ValidateGetPostRequest(ctx context.Context, req *v1.GetPostRequest) error {
    return nil
}

// ValidateListPostRequest 校验 ListPost 请求.
func (v *Validator) ValidateListPostRequest(ctx context.Context, req *v1.ListPostRequest) error {
    return nil
}
```

---

### 3.7 Errno 层（1 个文件）

#### `internal/pkg/errno/post.go`

```go
package errno

import (
    "net/http"
)

// Post 业务错误.
var (
    ErrPostNotFound = New(http.StatusNotFound, "PostNotFound", "post not found")
    ErrPostExists   = New(http.StatusConflict, "PostExists", "post already exists")
    ErrPostInvalid  = New(http.StatusBadRequest, "PostInvalid", "post is invalid")
)

// PostErrors 返回 Post 相关错误，用于 register.go 注册.
func PostErrors() []*Errno {
    return []*Errno{ErrPostNotFound, ErrPostExists, ErrPostInvalid}
}
```

---

### 3.8 Proto 层（1 个文件，可选）

#### `pkg/api/<app>/v1/post.proto`

```proto
syntax = "proto3";

package <app>.v1;

option go_package = "<module>/pkg/api/<app>/v1;v1";

import "google/api/annotations.proto";
import "validate/validate.proto";

// Post 实体.
message PostInfo {
  string post_id    = 1;
  string title      = 2;
  string content    = 3;
  int64  created_at = 4;
  int64  updated_at = 5;
}

// CreatePostRequest 创建 Post 请求.
message CreatePostRequest {
  string title   = 1 [(validate.rules).string.min_len = 1];
  string content = 2 [(validate.rules).string.min_len = 1];
}
message CreatePostResponse {
  string post_id = 1;
}

// UpdatePostRequest 更新 Post 请求.
message UpdatePostRequest {
  string post_id = 1;
  string title   = 2;
  string content = 3;
}
message UpdatePostResponse {}

// DeletePostRequest 删除 Post 请求.
message DeletePostRequest {
  string post_id = 1;
}
message DeletePostResponse {}

// GetPostRequest 查询单个 Post 请求.
message GetPostRequest {
  string post_id = 1;
}
message GetPostResponse {
  PostInfo post = 1;
}

// ListPostRequest 列表查询请求.
message ListPostRequest {
  int64 offset = 1;
  int64 limit  = 2;
}
message ListPostResponse {
  int64              total = 1;
  repeated PostInfo  posts = 2;
}
```

---

## 4. AST 注入点（4 个中央文件）

详细的 AST 注入实现详见 [05-registration-strategy.md](./05-registration-strategy.md)。这里仅列举注入清单。

### 4.1 注入到 `internal/<app>/biz/biz.go`

**操作**：在 `IBiz` 接口添加 `PostV1() postv1.PostBiz` 方法；在 `biz` 结构体添加 `PostV1()` 实现方法；在 import 段追加 alias。

**注入前**：

```go
package biz

import (
    userv1 "github.com/<module>/internal/<app>/biz/v1/user"
    "github.com/<module>/internal/<app>/store"
)

type IBiz interface {
    UserV1() userv1.UserBiz
}

type biz struct {
    store store.IStore
}

func (b *biz) UserV1() userv1.UserBiz {
    return userv1.New(b.store)
}
```

**注入后**：

```go
package biz

import (
    postv1 "github.com/<module>/internal/<app>/biz/v1/post"   // ← 新增
    userv1 "github.com/<module>/internal/<app>/biz/v1/user"
    "github.com/<module>/internal/<app>/store"
)

type IBiz interface {
    UserV1() userv1.UserBiz
    PostV1() postv1.PostBiz                                    // ← 新增
}

type biz struct {
    store store.IStore
}

func (b *biz) UserV1() userv1.UserBiz {
    return userv1.New(b.store)
}

func (b *biz) PostV1() postv1.PostBiz {                       // ← 新增
    return postv1.New(b.store)
}
```

### 4.2 注入到 `internal/<app>/store/store.go`

**操作**：在 `IStore` 接口添加 `Posts() PostStore`；在 `datastore` 结构体添加 `Posts()` 实现。

**注入后**（增量部分）：

```go
type IStore interface {
    DB(ctx context.Context, wheres ...where.Where) *gorm.DB
    TX(ctx context.Context, fn func(ctx context.Context) error) error
    User() UserStore
    Posts() PostStore                                     // ← 新增
}

func (store *datastore) Posts() PostStore {              // ← 新增
    return newPostStore(store)
}
```

### 4.3 注入到 `pkg/api/<app>/v1/<app>.proto`

**操作**：在 import 段追加 `import "post.proto";`。

**注入前**：

```proto
syntax = "proto3";
package <app>.v1;

import "user.proto";

service <App>Service {
  // RPC 定义
}
```

**注入后**：

```proto
syntax = "proto3";
package <app>.v1;

import "user.proto";
import "post.proto";                                      // ← 新增

service <App>Service {
  // RPC 定义
}
```

### 4.4 注入到 `internal/pkg/errno/register.go`

**操作**：在 `RegisterAll` 函数内追加 `RegisterErrors(PostErrors()...)`。

**注入前**：

```go
package errno

func RegisterAll() {
    RegisterErrors(UserErrors()...)
}
```

**注入后**：

```go
package errno

func RegisterAll() {
    RegisterErrors(UserErrors()...)
    RegisterErrors(PostErrors()...)                       // ← 新增
}
```

---

## 5. `--with` / `--without` Flag 控制矩阵

`--with` 与 `--without` 都接受**层级名白名单**（`conversion` / `validation` / `proto` / `errno`），二者**互斥**。语义如下：

| flag 用法 | 语义 |
| --- | --- |
| 不传 | 等价于 `--with conversion,validation,proto`（默认包含 errno） |
| `--with X,Y` | 显式列出可选层级；未列出的可选层级**不生成** |
| `--without X,Y` | 在默认基础上**剔除**指定层级 |

**必选层级（无法裁剪）**：handler + biz(6 文件) + store + model。

**可选层级（受 flag 控制）**：

| 层级名 | 默认 | 文件 |
| --- | --- | --- |
| `conversion` | ✅ 包含 | `internal/<app>/pkg/conversion/<resource>.go` |
| `validation` | ✅ 包含 | `internal/<app>/pkg/validation/<resource>.go` |
| `errno` | ✅ 包含 | `internal/pkg/errno/<resource>.go` + AST 注入 register.go |
| `proto` | ✅ 包含 | `pkg/api/<app>/v1/<resource>.proto` + AST 注入 `<app>.proto` |

**典型组合**（基线 = 必选 9 文件 + 可选 4 文件 = **13**）：

| 命令 | 等价于 | 文件数 | 适用 |
| --- | --- | --- | --- |
| `linctl add Post` | `--with conversion,validation,proto,errno` | **13**（默认全栈） | 完整 REST 资源 |
| `linctl add Post --without conversion,validation` | `--with proto,errno` | **11** | 只要 REST 接口，手工写转换/校验 |
| `linctl add Post --without proto` | `--with conversion,validation,errno` | **12** | 不需要 grpc-gateway |
| `linctl add Post --without conversion,validation,proto` | `--with errno` | **10** | 极简，仅含错误码 |
| `linctl add Post --without conversion,validation,proto,errno` | `--with`（空） | **9** | 极简纯逻辑（不推荐） |

> **必选 9 文件**：handler(1) + biz(6) + store(1) + model(1)；可选 4 文件：conversion / validation / errno / proto 各 1。

**AST 注入与可选层级的联动**：

| 可选层级 | 不包含时跳过的注入 |
| --- | --- |
| `proto` 缺失 | `<app>.proto` 不会被 import 注入 |
| `errno` 缺失 | `register.go` 不会被追加 `RegisterErrors` 调用 |
| `conversion` / `validation` | 无 AST 注入，独立文件 |

> **说明**：biz 层始终生成 6 个文件（1 接口 + 5 动词），不可裁剪。如需自定义动词，使用 `--ops` flag（见 §6）。

> **互斥规则**：`--with` 与 `--without` 不能同时传，传则报错（exit 28，详见 [02 §9.1](./02-command-set.md#91-退出码分段)）。

---

## 6. CRUD 动词裁剪

如不需要 5 个标准动词（CRUD + List），可以在 `linctl add` 命令使用 `--ops` flag：

```bash
# 仅生成 Get + List（只读资源）
linctl add Audit --ops get,list

# 仅生成 Create + Get（不可变资源）
linctl add Event --ops create,get
```

支持的 `--ops` 值：`create` / `update` / `delete` / `get` / `list`（默认全部）。

---

## 7. 多个资源批量生成

```bash
linctl add Post Comment Tag
```

`linctl` 会**逐个**处理每个资源（不并发，避免 AST 注入冲突），每个资源完整执行：创建文件 → AST 注入 → 校验。任意一个失败时，**整批回滚**（前面已成功的也回滚）。

---

## 8. 资源命名最佳实践

| ✅ 推荐 | ❌ 不推荐 | 原因 |
| --- | --- | --- |
| `Post`、`Comment`、`Tag` | `posts`、`tags` | 单数 PascalCase |
| `UserProfile` | `user_profile`、`user-profile` | PascalCase |
| `OrderItem` | `OrderItems` | 单数 |
| `APIKey` | `ApiKey` | 缩略词全大写（与 Go 风格一致） |

> 命令解析时会做规范化校验，不规范名称会被拒绝并给出修正建议。

---

## 9. 设计取舍与简化说明

| 维度 | 完整生产实现 | linctl 生成 | 备注 |
| --- | --- | --- | --- |
| biz/v1 子目录命名 | `user/` | `post/` | 一致：lowercase |
| biz 接口命名 | `UserBiz` | `PostBiz` | 一致：PascalCase + Biz |
| biz 动词文件分离 | ✅ 是（create/update/...） | ✅ 是 | 一致 |
| store 单文件 | ✅ 是（user.go） | ✅ 是 | 一致 |
| model 命名 | `UserM` (后缀 M) | `PostM` | 一致 |
| handler init 注册 | ✅ 是 | ✅ 是 | 一致 |
| conversion 包名 | `conversion` | `conversion` | 一致 |
| validation 包名 | `validation` | `validation` | 一致 |
| proto 路径 | `pkg/api/apiserver/v1/` | `pkg/api/<app>/v1/` | 一致 |
| wire ProviderSet | ✅ 是 | ✅ 是（在 biz.go / store.go） | 一致 |

---

## 10. 模板文件清单

> **SoT（Single Source of Truth）**：完整模板目录树（含 `project/` 与 `resource/`）维护在 [04-template-system.md §3.1](./04-template-system.md#31-标准结构embed-与外部目录都遵守)。本文档仅引用，不重复维护。

资源相关模板位于 `internal/templates/resource/`，关键文件：

| 模板 | 输出（示例 `linctl add Post`） |
| --- | --- |
| `handler.go.tpl` | `internal/myblog/handler/post.go` |
| `biz/biz_iface.go.tpl` + `biz/biz_struct.go.tpl` | `internal/myblog/biz/v1/post/post.go`（合并输出） |
| `biz/verb_{create,update,delete,get,list}.go.tpl` | `internal/myblog/biz/v1/post/{create,update,delete,get,list}.go` |
| `store.go.tpl` | `internal/myblog/store/post.go` |
| `model.gen.go.tpl` | `internal/myblog/model/post.gen.go` |
| `conversion.go.tpl` | `internal/myblog/pkg/conversion/post.go` |
| `validation.go.tpl` | `internal/myblog/pkg/validation/post.go` |
| `errno.go.tpl` | `internal/pkg/errno/post.go` |
| `proto.tpl` | `pkg/api/myblog/v1/post.proto` |

> 单个资源接口 + 工厂合并到 `biz_iface.go.tpl + biz_struct.go.tpl`（最终输出仍是单文件 `post.go`）；动词文件独立模板。

---

_Last reviewed: 2026-04-29_
