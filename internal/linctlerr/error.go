// Package linctlerr 提供 linctl 全局统一的错误类型。
//
// SSOT 定稿（详见 docs/META-fix-decisions-2026-04-25.md §1.1 / §5.1）：
//   - 包路径：internal/linctlerr（不是 internal/errors，不使用别名）
//   - 类型名：LinctlError
//   - 错误码常量类型：Code（不是 ErrCode）
//   - 错误码字段名：Code
//   - 错误码常量命名：Err<DescriptiveName>（如 ErrConfigInvalid）
//   - 错误码字符串值：snake_case（便于 JSON / YAML / 日志 grep）
package linctlerr

import (
	"errors"
	"fmt"
	"strings"
)

// Code 是错误码字符串类型。所有 LinctlError 都必须携带一个 Code，便于：
//   - main 函数按 Code 决定退出码
//   - i18n 按 Code 查找本地化错误信息
//   - 日志按 Code 做结构化分析
type Code string

// LinctlError 是 linctl 中所有用户可见错误的统一类型。
//
// 字段语义：
//   - Code: 错误分类（必填），用于程序化处理
//   - Message: 人类可读的错误描述（必填）
//   - Hint: 可执行的修复建议（推荐填写，提升 UX）
//   - DocLink: 相关文档/ADR 链接（可选；Pretty 输出时显示）
//   - Cause: 底层 error（用于 errors.Unwrap，可选）
//
// 用法示例：
//
//	return linctlerr.New(linctlerr.ErrConfigInvalid,
//	    "invalid framework \"kratos\" for component \"myblog\"",
//	    "Allowed values: [gin, grpc]")
//
//	return linctlerr.Wrap(linctlerr.ErrEnvironment, err, "read linctl.yaml").
//	    WithDocLink("https://github.com/clin211/lin/blob/main/lin/docs/04-config-schema.md")
type LinctlError struct {
	Code    Code
	Message string
	Hint    string
	DocLink string
	Cause   error
}

// Error 实现 error 接口。格式：[<code>] <message>: <cause>
//
// Error() 是机器/单测友好的紧凑格式；用户可见的多行格式见 Pretty()。
func (e *LinctlError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Pretty 返回用户友好的多行错误展示，包含 Reason / Hint / DocLink 三段。
//
// 设计意图（详见 docs/11 §11.3.1 Story 2.6）：
//   - main 函数 / cobra 默认错误打印用紧凑 Error()
//   - 顶层 CLI 错误展示请用 Pretty() 给用户更友好提示
//   - noColor=true 时不渲染 ANSI 颜色（CI / 文件输出）
//
// 输出示例：
//
//	Error: [config_invalid] invalid framework "kratos"
//	Hint:  Allowed values: [gin, grpc]
//	Doc:   https://github.com/clin211/lin/blob/main/lin/docs/04-config-schema.md
func (e *LinctlError) Pretty(noColor bool) string {
	if e == nil {
		return ""
	}
	const (
		red    = "\x1b[31m"
		yellow = "\x1b[33m"
		cyan   = "\x1b[36m"
		reset  = "\x1b[0m"
	)
	colorize := func(c, s string) string {
		if noColor {
			return s
		}
		return c + s + reset
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", colorize(red, "Error:"), e.Error())
	if e.Hint != "" {
		for _, line := range strings.Split(e.Hint, "\n") {
			fmt.Fprintf(&b, "%s  %s\n", colorize(yellow, "Hint: "), line)
		}
	}
	if e.DocLink != "" {
		fmt.Fprintf(&b, "%s   %s\n", colorize(cyan, "Doc:"), e.DocLink)
	}
	return strings.TrimRight(b.String(), "\n")
}

// Unwrap 支持 errors.Is / errors.As 链式查询。
func (e *LinctlError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is 支持按 Code 比较：errors.Is(err, &LinctlError{Code: ErrConfigInvalid})
// 注意：通常推荐用 errors.As 拿到 *LinctlError 后比较 .Code，更直观。
func (e *LinctlError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	var t *LinctlError
	if !errors.As(target, &t) {
		return false
	}
	return e.Code == t.Code
}

// New 创建一个不包装底层错误的 LinctlError。
//   - code: 错误分类
//   - message: 人类可读描述（必填）
//   - hint: 修复建议（可选；多个 hint 用 \n 分隔；空字符串表示无 hint）
func New(code Code, message string, hint ...string) *LinctlError {
	e := &LinctlError{Code: code, Message: message}
	if len(hint) > 0 {
		e.Hint = joinHints(hint)
	}
	return e
}

// Newf 是 New 的格式化版本，便于动态构造 message。
func Newf(code Code, format string, args ...any) *LinctlError {
	return &LinctlError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap 包装一个底层 error 为 LinctlError。常用于 IO/解析等场景。
//   - code: 分类
//   - cause: 底层 error（不能为 nil；nil 时返回 nil）
//   - message: 上下文描述（如 "read foo.yaml"）
func Wrap(code Code, cause error, message string) *LinctlError {
	if cause == nil {
		return nil
	}
	return &LinctlError{Code: code, Message: message, Cause: cause}
}

// Wrapf 是 Wrap 的格式化版本。
func Wrapf(code Code, cause error, format string, args ...any) *LinctlError {
	if cause == nil {
		return nil
	}
	return &LinctlError{Code: code, Message: fmt.Sprintf(format, args...), Cause: cause}
}

// WithHint 给现有 LinctlError 追加 Hint（不可变模式：返回新对象）。
func (e *LinctlError) WithHint(hint string) *LinctlError {
	if e == nil {
		return nil
	}
	cp := *e
	if cp.Hint == "" {
		cp.Hint = hint
	} else {
		cp.Hint = cp.Hint + "\n" + hint
	}
	return &cp
}

// WithDocLink 给现有 LinctlError 设置 DocLink（不可变模式：返回新对象）。
func (e *LinctlError) WithDocLink(link string) *LinctlError {
	if e == nil {
		return nil
	}
	cp := *e
	cp.DocLink = link
	return &cp
}

// CodeOf 从任意 error 中提取 Code；非 LinctlError 返回 "" + false。
func CodeOf(err error) (Code, bool) {
	var lerr *LinctlError
	if errors.As(err, &lerr) {
		return lerr.Code, true
	}
	return "", false
}

func joinHints(hints []string) string {
	out := ""
	for _, h := range hints {
		if h == "" {
			continue
		}
		if out == "" {
			out = h
		} else {
			out += "\n" + h
		}
	}
	return out
}
