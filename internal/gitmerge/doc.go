// Package gitmerge 包装系统 `git merge-file` 命令做 3-way merge。
//
// 为什么不自实现 diff3：
//   - 系统 git 已被全球数千万开发者验证，边界场景比自实现更稳
//   - 用户解决冲突时看到的是熟悉的 `<<<<<<<` / `=======` / `>>>>>>>` 标记，
//     IDE / `git mergetool` 都自带支持，零学习成本
//   - 实现复杂度低（fork+exec git）
//
// 但本包仍提供一个内置 diff3 fallback，覆盖：
//   - 用户机器没装 git（虽然罕见）
//   - 单元测试 / hermetic CI 不希望依赖外部命令
//
// 使用示例：
//
//	out, conflict, err := gitmerge.MergeFile(ctx, gitmerge.Files{
//	    Base:   baseContent,
//	    Ours:   oursContent,
//	    Theirs: theirsContent,
//	    Labels: gitmerge.Labels{Base: "v0.3.0", Ours: "lin-side", Theirs: "miniblog-v4@def67890"},
//	})
//
// 双视角共用：
//   - META-template-lifecycle §4.2：用户改 vs 模板改
//   - META-template-upstream-sync §3-§5：upstream 改 vs lin-side 改
package gitmerge
