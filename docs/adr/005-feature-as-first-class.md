# ADR-005: 把 Feature 提升为一等公民（接口 + Registry）

## 元数据

| 字段 | 值 |
| --- | --- |
| **状态** | ✅ Accepted (2026-04-25) |
| **日期** | 2026-04-25 |
| **作者** | @clin211 |
| **审阅人** | TBD |
| **相关 ADR** | [ADR-004](./004-plan-apply-pattern.md) |
| **影响范围** | `internal/feature/`、`internal/component/`、`08-feature-system.md` |
| **相关 issue/PR** | - |

---

## 1. 背景

osbuilder 用 **bool 字段 + if-switch** 表达"特性"：

```go
type WebServer struct {
    WithUser     bool
    WithHealthz  bool
    WithOTel     bool
    WithWS       bool
    WithPreloader bool
    // ...
}

func (ws *WebServer) Pairs() map[string]string {
    pairs := map[string]string{...}
    if ws.WithUser     { /* 加 8~10 个文件 */ }
    if ws.WithHealthz  { /* 加 2~3 个文件 */ }
    if ws.WithOTel     { /* 加 1~2 个文件 */ }
    if ws.WithWS       { /* 加 4~5 个文件 */ }
    if ws.WithPreloader{ /* 加 2~3 个文件 */ }
    switch ws.WebFramework { /* 5 个分支 */ }
    switch ws.ServiceRegistry { /* 4 个分支 */ }
    return pairs
}
```

问题：

1. **不可扩展**：要加新特性（如 Sentry / AuditLog），必须改这个核心函数 + 加新 bool 字段。
2. **第三方贡献门槛高**：必须 fork 整个 osbuilder。
3. **隐式依赖**：`WithUser` 必须在 `WithOTel` 之前生效（否则 metrics 配置缺字段），但代码里看不出来。
4. **测试困难**：必须把 32 = 2^5 种特性组合都跑一遍才有信心。
5. **配置语法死板**：`withUser: true` 不能优雅表达"启用"语义；将来想加 `withUser: { jwtSecret: ... }` 就要把 bool 改 struct 破坏兼容。

## 2. 决策

我们决定把 Feature 提升为**一等抽象**：定义 `Feature` 接口 + `Registry` 注册中心，让所有特性都通过统一接口贡献模板对、AST mutator、FuncMap、默认值。

```go
type Feature interface {
    Name() string
    AppliesTo() []string  // ["WebServer", "Worker"]
    Apply(p *Project, c Component, b *PairBuilder)  // 贡献文件
    Mutators(p *Project, c Component) []ASTMutator  // 贡献 AST 改动
    FuncMap() template.FuncMap                      // 贡献模板函数
    Defaults(p *Project, c Component) map[string]any
    Validate(p *Project, c Component) error
    Order() int  // 解决依赖顺序
}
```

关键变更：

1. `linctl.yaml` 中 `features` 改为字符串列表：`features: [healthz, opentelemetry, user]`。
2. 内置 5 个 Feature：`HealthzFeature`、`OpenTelemetryFeature`、`UserFeature`、`WebSocketFeature`、`PreloaderFeature`。
3. 每个 Feature 在 `init()` 中通过 `feature.MustRegister(&XxxFeature{})` 注册到全局 `feature.Default`。
4. Phase 5 引入插件机制：`linctl-plugin-<name>` 子进程通过 stdio JSON-RPC 提供 Feature。

## 3. 备选方案

| # | 方案 | 优势 | 劣势 | 否决原因 |
| --- | --- | --- | --- | --- |
| 1 | **Feature 接口 + Registry**（chosen） | 真正可扩展；测试 / 隔离友好；插件可外接 | 接口稳定后变更代价大 | - |
| 2 | 保留 bool 字段 + 模式（osbuilder 现状） | 简单 | 不可扩展；不能插件化 | 长期不可维护 |
| 3 | 用 Go plugin (`plugin/v8`) 实现内置 Feature | 完全 Go 化 | plugin 依赖完全相同的 Go 版本；不支持 cross-compile；不支持 Windows | 限制太多 |
| 4 | YAML/Lua 等 DSL 描述 Feature | 配置即文件 | DSL 表达力受限；调试困难 | ROI 太低 |
| 5 | 完全用 Webhook（HTTP API）做 Feature | 跨语言 | 引入网络依赖；扩展性反而更差 | 与本地 CLI 工具理念冲突 |

## 4. 后果

### 4.1 正面

- **真正可扩展**：第三方贡献者不需要 fork linctl，开发独立的 `linctl-plugin-xxx` 包即可分发。
- **特性间依赖显式**：通过 `Order() int` 明确表达"user 必须先于 OTel"。
- **Single Responsibility**：每个 Feature 一个文件，一个测试套件。
- **配置可扩展**：未来从 `features: [user]` 升级为 `features: [{name: user, jwtSecret: ...}]` 不破坏兼容（向后兼容的 schema 演进）。
- **可插拔测试**：测试时可以创建空的 Registry，只注册需要的 Feature，避免 32 种组合爆炸。

### 4.2 负面 / Trade-offs

- **接口稳定性是契约**：`Feature` 接口一旦发布就不能轻易变（所有插件都依赖它）。需要 SemVer 严格管理。
- **学习曲线**：开发者需要理解 Feature/Component/Pair/Mutator 四个抽象之间的关系。
- **运行时开销**：每个 Feature 实例化 + 反射调用，约增加 ~5-10ms 启动时间（可接受）。
- **插件协议复杂度**：Phase 5 的 stdio JSON-RPC 需要设计序列化 Mutator 协议。

### 4.3 中性

- 内置 Feature 数量从 osbuilder 的 5 个扩展为可插件化无上限。
- 配置 schema 从 `bool` 字段改为 `[]string`，是 breaking change（但 linctl 是新工具，无历史包袱）。

## 5. 风险缓解

- **接口稳定性**：`Feature` 接口在 v1 通过严格 review 后冻结；任何破坏性变更（如增加方法）通过包装类型 + adaptor 实现。
- **接口扩展机制**：未来如要新增能力（如 `ConfigMutations`），用 type assertion 检测：
  ```go
  if cm, ok := f.(ConfigMutator); ok {
      muts := cm.ConfigMutations(p, c)
  }
  ```
  这样老 Feature 不实现新接口也不报错。
- **插件版本兼容性**：插件元数据中带 `linctlMinVersion`，运行时校验。

## 6. 实施分阶段

| Phase | 范围 |
| --- | --- |
| Phase 1 | Feature 接口 + 5 个内置 Feature（healthz/otel/user/ws/preloader） |
| Phase 2 | Order 机制；Validate 加完善；ConfigMutations 接口（可选） |
| Phase 3 | `linctl add feature <name>` 子命令 |
| Phase 5 | 插件机制（kubectl 风格）+ `linctl plugin install/list/remove` |

详见 [11-implementation-plan.md](../11-implementation-plan.md) 的 Phase 1 / Phase 5。

## 7. 参考资料

- [kubectl plugin 机制](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [Hashicorp go-plugin](https://github.com/hashicorp/go-plugin)（subprocess 模式参考）
- [设计文档：08-feature-system.md](../08-feature-system.md)
- [Cobra 命令的 init 注册模式](https://github.com/spf13/cobra)

---

_Last reviewed: 2026-04-25_
