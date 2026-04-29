// Package template 是 linctl 的模板渲染引擎。
//
// 核心设计（详见 docs/05-template-system.md 与 SSOT §5.5）：
//   - 业务模板首次 render 时 Parse，结果缓存
//   - missingkey=error 严格模式（模板访问不存在字段直接报错）
//   - 渲染失败时 RenderError 携带模板路径 + 数据上下文
package template

import "embed"

// TemplatesFS 通过 go:embed 嵌入全部 templates/ 目录。
//
// 设计原则：
//   - all: 前缀确保隐藏文件（如 .gitkeep）也被嵌入
//   - go:embed 不允许 ../ 跨包，所以 templates/ 放在本包下
//
//go:embed all:templates
var TemplatesFS embed.FS
