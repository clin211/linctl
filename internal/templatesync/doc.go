// Package templatesync 实现 linctl 维护者侧的"上游模板同步"机制。
//
// 它解决的核心问题：把 monorepo 内的"模板源"项目（如 miniblog-v4）的演进
// 半自动同步到 lin 仓库内的"模板镜像"目录（lin/internal/template/templates/web-gin/）。
//
// 与终端用户视角的项目升级机制（详见 docs/META-template-lifecycle-2026-04-28.md）
// 解决"正交问题"：本包面向 lin 仓库维护者，不出现在面向用户的 CLI help 中
// （命令前缀 linctl internal templatesync ...）。
//
// 设计要点（详见 docs/META-template-upstream-sync-2026-04-28.md）：
//
//   - sync.yaml：声明式同步规则（哪个上游路径 → 哪个目标 .tpl 路径 + 应用哪些 transform）
//   - upstream-sync.lock.json：同步状态（每文件 srcHash / dstHash / owner / 上次同步时间）
//   - Transform 接口：可组合的变换规则（rewriteImports / replaceLiteral / addExtension 等）
//   - Owner 三态：upstream（完全跟随上游）/ shared（双方共维护，3-way merge）/ linSpecific（仅 lin 自有）
//   - 复用 internal/gitmerge/（与 META-template-lifecycle L2 共享）做 3-way merge
//
// U1 阶段（当前）：sync.yaml 解析 + 5 个核心 transform + status / plan / apply --strategy=force。
// U2 阶段：3-way merge + ask 策略 + add / revert / check 子命令 + CI 集成。
// U3 阶段：translateComments AI 辅助 / 双向 push / 自动新文件归类。
package templatesync
