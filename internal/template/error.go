package template

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// RenderError 表示一次模板渲染的失败。它包装底层 error，并携带渲染上下文，便于调试。
//
// 实现 error 与 Unwrap，可与 errors.Is / errors.As 协作。
//
// 设计意图（详见 docs/11 §11.3.1 Story 2.6）：
//   - 用户友好：Error() 显示模板路径 + 行号 + 列号 + 数据上下文摘要
//   - 调试友好：Snippet / Data 字段便于 errors.As 提取后做深度诊断
type RenderError struct {
	Template string // 模板相对路径
	Err      error  // 底层错误（来自 text/template）
	Data     any    // 渲染时传入的数据（可能为 nil）
	Snippet  string // 出错位置附近的模板代码片段（可选）
	Line     int    // 出错行号（>0 时有效；从 text/template error 中解析）
	Col      int    // 出错列号（>0 时有效）
}

// Error 实现 error 接口。
func (e *RenderError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "template render failed: %s", e.Template)
	if e.Line > 0 {
		fmt.Fprintf(&b, ":%d", e.Line)
		if e.Col > 0 {
			fmt.Fprintf(&b, ":%d", e.Col)
		}
	}
	fmt.Fprintf(&b, ": %v", e.Err)
	if e.Snippet != "" {
		fmt.Fprintf(&b, "\n--- snippet ---\n%s", e.Snippet)
	}
	if summary := dataSummary(e.Data); summary != "" {
		fmt.Fprintf(&b, "\n--- data ---\n%s", summary)
	}
	return b.String()
}

// Unwrap 暴露底层错误，便于 errors.Is/errors.As。
func (e *RenderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// templatePosRe 匹配 text/template 错误信息中的 "template: <name>:<line>:<col>:" 前缀。
// text/template 标准错误格式见 https://pkg.go.dev/text/template#hdr-Errors
var templatePosRe = regexp.MustCompile(`template: [^:]+:(\d+)(?::(\d+))?`)

// extractRenderPos 从 text/template error 中提取 line:col。
// 不匹配时返回 (0, 0)。
func extractRenderPos(err error) (int, int) {
	if err == nil {
		return 0, 0
	}
	m := templatePosRe.FindStringSubmatch(err.Error())
	if m == nil {
		return 0, 0
	}
	line, _ := strconv.Atoi(m[1])
	col := 0
	if len(m) > 2 && m[2] != "" {
		col, _ = strconv.Atoi(m[2])
	}
	return line, col
}

// dataSummary 返回 data 的紧凑摘要（最多 200 字符），用于错误信息。
// 大对象不会被完整展开，避免炸日志。
func dataSummary(data any) string {
	if data == nil {
		return ""
	}
	s := fmt.Sprintf("%+v", data)
	const max = 200
	if len(s) > max {
		return s[:max] + "...(truncated)"
	}
	return s
}
