// Package errs 定义 lin v2 的错误码与错误类型。
//
// 设计来源：lin/docs/features/02-command-set.md §9.2「错误类型 → 退出码映射」。
//
// 关键约束：
//   - 用户**永远**只看到 §9.1 的退出码范围（0/1/2/10–39/130）
//   - 40+ / 50+ 的内部码仅出现在 debug 日志，由 mapToUserCode 映射到命令对应区间
package errs

import "errors"

// Code 表示一个错误码（同时是用户可见的退出码或内部码）。
type Code int

// 通用码（0-9）。
const (
	CodeOK         Code = 0
	CodeUnknown    Code = 1
	CodeInvalidArg Code = 2
)

// new 命令码（10-19）。
const (
	CodeTargetExists  Code = 10 // 目标目录已存在且未传 --force
	CodeBadModule     Code = 11 // module path 不合法
	CodeRenderFailed  Code = 12 // 模板渲染失败（透传 40-49）
	CodeWriteFailed   Code = 13 // 文件写入失败（权限/磁盘）
	CodeUserCancelled Code = 14 // 用户取消
)

// add 命令码（20-29）。
const (
	CodeNotProjectRoot  Code = 20 // 找不到 go.mod
	CodeBadGoMod        Code = 21 // go.mod 解析失败
	CodeMultiAppNoFlag  Code = 22 // 多 app 但未传 --app
	CodeBadResourceName Code = 23 // 资源名不合法（非 PascalCase 等）
	CodeSymbolMissing   Code = 24 // AST 注入目标符号缺失（如接口/函数未定义）
	CodeInjectFailed    Code = 25 // AST 注入失败但回滚成功
	CodeRollbackFailed  Code = 26 // AST 注入失败且回滚失败（人工介入）
	CodeAddCancelled    Code = 27 // 用户取消
	CodeFlagConflict    Code = 28 // flag 互斥冲突（如 --with 与 --without 同传）
)

// lint 命令码（30-34）。
const (
	CodeLintIssues    Code = 30 // 至少一项 error
	CodeFixPartial    Code = 31 // --fix 部分失败
	CodeLintNoProject Code = 32 // 项目根识别失败
)

// doctor 命令码（35-39）。
const (
	CodeDoctorErrors     Code = 35 // 至少一项 error
	CodeDoctorWarnStrict Code = 36 // --strict 下有 warning
)

// 模板层内部码（40-49，由 new/add/lint 透传）。
const (
	CodeTplNotFound      Code = 40
	CodeTplParseError    Code = 41
	CodeTplExecError     Code = 42
	CodeTplPathTraversal Code = 43 // 安全：渲染路径跳出 RootDir
)

// AST 层内部码（50-59，仅由 add 触发）。
const (
	CodeASTParseError    Code = 50
	CodeASTSymbolMissing Code = 51 // 内部码：AST 找不到目标符号
	CodeASTApplyError    Code = 52
	CodeASTBackupFailed  Code = 53
)

// 标准约定。
const (
	CodeSIGINT Code = 130
)

// Error 是 lin v2 的标准错误类型。
//
// 字段含义：
//   - Code  : 错误码（可能是用户可见码或内部码，由 mapToUserCode 转换）
//   - Message: 用户友好的简短描述
//   - Hint   : 修复建议（可选）
//   - Cause  : 链式根因（支持 errors.Is/As/Unwrap）
type Error struct {
	Code    Code
	Message string
	Hint    string
	Cause   error
}

// Error 实现 error 接口。
func (e *Error) Error() string { return e.Message }

// Unwrap 支持 errors.Is / errors.As 链式追溯。
func (e *Error) Unwrap() error { return e.Cause }

// New 创建一个无根因的 Error。
func New(code Code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

// Wrap 创建一个携带根因的 Error。
func Wrap(code Code, msg string, cause error) *Error {
	return &Error{Code: code, Message: msg, Cause: cause}
}

// WithHint 给 Error 追加修复建议。
func (e *Error) WithHint(hint string) *Error {
	e.Hint = hint
	return e
}

// CodeOf 将 error 映射到当前命令的用户可见退出码。
//
// cmd 应为当前执行的命令名（"new" / "add" / "lint" / "doctor" / "version" / "completion"）。
// 非 *Error 类型的 error 会返回 CodeUnknown。
func CodeOf(cmd string, err error) Code {
	if err == nil {
		return CodeOK
	}
	var e *Error
	if errors.As(err, &e) {
		return mapToUserCode(cmd, e.Code)
	}
	return CodeUnknown
}

// mapToUserCode 将内部码（40-59）按当前 cmd 映射到用户可见范围。
func mapToUserCode(cmd string, internal Code) Code {
	// 命令自身范围内的码原样返回
	switch cmd {
	case "new":
		if internal >= 10 && internal <= 19 {
			return internal
		}
	case "add":
		if internal >= 20 && internal <= 29 {
			return internal
		}
	case "lint":
		if internal >= 30 && internal <= 34 {
			return internal
		}
	case "doctor":
		if internal >= 35 && internal <= 39 {
			return internal
		}
	}

	// 模板层 (40-49) → 命令对应码
	if internal >= CodeTplNotFound && internal <= CodeTplPathTraversal {
		switch cmd {
		case "new":
			return CodeRenderFailed // 12
		case "add":
			return CodeInjectFailed // 25
		case "lint":
			return CodeFixPartial // 31
		default:
			return CodeUnknown
		}
	}

	// AST 层 (50-59) → 仅 add 关心
	switch internal {
	case CodeASTSymbolMissing:
		return CodeSymbolMissing // 24
	case CodeASTBackupFailed:
		return CodeRollbackFailed // 26
	case CodeASTParseError, CodeASTApplyError:
		return CodeInjectFailed // 25
	}

	// 通用 / 标准约定原样返回
	return internal
}
