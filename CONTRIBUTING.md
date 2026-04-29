# 贡献 linctl

感谢你对 linctl 感兴趣！本文档涵盖核心约定。

## 开发环境

```bash
git clone https://github.com/clin211/lin.git
cd linctl
make tools     # 安装 pin 版本的工具链
make all       # lint + test + build
```

## 行为准则

- 严格遵守 [docs/13-coding-standards.md](./docs/13-coding-standards.md)。
- 所有设计决策必须与 [docs/META-fix-decisions-2026-04-25.md](./docs/META-fix-decisions-2026-04-25.md)（SSOT）一致。
- 新增 CLI flag 必须先在 [docs/03-cli-design.md](./docs/03-cli-design.md) 中定义。
- 新增 `spec.*` 字段必须先在 [docs/04-config-schema.md](./docs/04-config-schema.md) 中定义。

## PR 检查清单

- [ ] 添加了测试（变更代码覆盖率目标 ≥70%）
- [ ] `make lint` 通过（gofumpt v0.7.0 + golangci-lint v1.59.1）
- [ ] `make test` 通过（启用 race detector）
- [ ] 行为变更同步更新了文档
- [ ] 遵守 SSOT 决策（见 META-fix-decisions）
- [ ] Conventional Commits 格式（`feat:`, `fix:`, `docs:`, `refactor:`, ...）

## 架构概览

linctl 使用 5 层架构（L0–L4），详见 [docs/01-architecture.md](./docs/01-architecture.md)。
层级依赖规则在 CI 中由 `go-cleanarch` 强制检查。

新增 Feature 时建议遵循以下生命周期：

1. 如果是重大决策，先在 `docs/adr/` 写一个 ADR
2. 如果跨多份文档，更新 SSOT（`docs/META-fix-decisions-*.md`）
3. 实现代码（含测试）
4. 提交 PR 并附上链接到 ADR / 文档变更

## 包路径与命名

- 错误类型：`internal/linctlerr`（**不是** `internal/errors`，详见 SSOT §1.1 / §5.1）
- 项目加载器：`internal/project/loader.go`（**不在** `internal/orchestrator`，详见 SSOT §1.2）
- Feature 接口：`internal/feature/feature.go`（含 `Requires() / ResourceContributions() / Apply()`，详见 SSOT §1.15 / §1.16）

## License

提交贡献即视为同意你的代码以 MIT License 发布。
