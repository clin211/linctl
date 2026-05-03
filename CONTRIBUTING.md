# 贡献 linctl（lin 仓库）

感谢你对 lin 感兴趣！本文档涵盖核心开发约定。

## 开发环境

```bash
git clone https://github.com/clin211/lin.git
cd lin
make tools     # 安装 pin 版本的工具链（gofumpt / mockgen / gotestsum）
make all       # lint + test + build
```

需要 Go >= 1.22。

## 行为准则

- 严格遵守 [docs/features/](./docs/features/) 中的设计文档（这是 SSOT）。
- 新增 CLI flag 必须先在 [docs/features/02-command-set.md](./docs/features/02-command-set.md) 中定义。
- 新增模板必须与 [docs/features/04-template-system.md](./docs/features/04-template-system.md) 保持一致。
- 不能修改 `internal/templates/` 中已有模板，除非有充分理由并更新相应文档。

## PR 检查清单

- [ ] 添加了测试（新代码覆盖率目标 ≥ 70%）
- [ ] `make vet` 通过
- [ ] `make test` 通过（含 race detector）
- [ ] `make e2e` 通过（new / add / lint / doctor 全部 E2E 测试）
- [ ] 行为变更已同步更新了 docs/features/ 中对应文档
- [ ] Conventional Commits 格式（`feat:`, `fix:`, `docs:`, `refactor:`, ...）

## 代码架构

```
lin/
├── cmd/linctl/        # L0 入口层（≤ 50 行）
├── internal/
│   ├── cli/           # L1 命令分发层（cobra 命令注册）
│   ├── scaffold/      # L2 核心生成层（context / plan / render / project / resource）
│   ├── ast/           # L3 能力层 - AST 注入
│   ├── check/         # L3 能力层 - lint / doctor 检查
│   ├── templates/     # L4 资产层（embed.FS）
│   ├── version/       # 版本元数据
│   └── pkg/           # 工具库（errs / fsx / tpl / logx）
└── tests/e2e/         # E2E 测试脚本
```

### L1 → L2 数据流

```
cobra command
  → scaffold.LoadContext / NewContextFromFlags
  → scaffold.BuildPlan
  → scaffold.Render / AddResource
  → ast.Injector.Inject
```

## 文档目录（docs/features/）

| 文件 | 内容 |
|------|------|
| 00-refactor-rationale.md | 重构背景与决策 |
| 01-architecture-blueprint.md | 架构分层与关键抽象 |
| 02-command-set.md | 6 个命令的完整设计（flag / 行为 / 退出码） |
| 03-resource-scaffold.md | lin add 生成的文件清单 |
| 04-template-system.md | 模板加载 / 渲染 / 变量体系 |
| 05-registration-strategy.md | AST 注入语义与锚点规范 |
| 06-migration-plan.md | v1 → v2 分阶段迁移计划 |
| 07-interactive-ux.md | 交互式向导 UX 流程 |
