package errorsx

import (
	"errors"
	"fmt"
)

// 预定义的标准错误（兼容旧版本）
var (
	// OK 代表请求成功
	OK = NewBizError(CodeOK, "OK", "success")

	// ErrInternal 表示所有未知的服务器端错误
	ErrInternal = NewBizError(CodeInternalServer, "InternalError", "Internal server error.")

	// ErrNotFound 表示资源未找到
	ErrNotFound = NewBizError(CodeUserNotFound, "NotFound", "Resource not found.")

	// ErrBind 表示请求体绑定错误
	ErrBind = NewBizError(CodeUserInvalidCredentials, "BindError", "Error occurred while binding the request body to the struct.")

	// ErrInvalidArgument 表示参数验证失败
	ErrInvalidArgument = NewBizError(CodeUserInvalidCredentials, "InvalidArgument", "Argument verification failed.")

	// ErrUnauthenticated 表示认证失败
	ErrUnauthenticated = NewBizError(CodeAuthUnauthenticated, "Unauthenticated", "Unauthenticated.")

	// ErrPermissionDenied 表示请求没有权限
	ErrPermissionDenied = NewBizError(CodeUserPermissionDenied, "PermissionDenied", "Permission denied. Access to the requested resource is forbidden.")

	// ErrOperationFailed 表示操作失败
	ErrOperationFailed = NewBizError(CodeInternalServer, "OperationFailed", "The requested operation has failed. Please try again later.")
)

// ErrorX 兼容旧版本的错误类型（已弃用，建议使用 BizError）
// Deprecated: 使用 BizError 替代
type ErrorXCompat struct {
	// Code 表示错误的 HTTP 状态码，用于与客户端进行交互时标识错误的类型.
	Code int `json:"code,omitempty"`

	// Reason 表示错误发生的原因，通常为业务错误码，用于精准定位问题.
	Reason string `json:"reason,omitempty"`

	// Message 表示简短的错误信息，通常可直接暴露给用户查看.
	Message string `json:"message,omitempty"`

	// Metadata 用于存储与该错误相关的额外元信息，可以包含上下文或调试信息.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewCompat 创建一个新的错误（兼容旧版本）
// Deprecated: 使用 NewBizError 替代
func NewCompat(code int, reason string, format string, args ...any) *ErrorXCompat {
	return &ErrorXCompat{
		Code:    code,
		Reason:  reason,
		Message: fmt.Sprintf(format, args...),
	}
}

// Error 实现 error 接口中的 `Error` 方法.
func (err *ErrorXCompat) Error() string {
	return fmt.Sprintf("error: code = %d reason = %s message = %s metadata = %v", err.Code, err.Reason, err.Message, err.Metadata)
}

// WithMessage 设置错误的 Message 字段.
func (err *ErrorXCompat) WithMessage(format string, args ...any) *ErrorXCompat {
	err.Message = fmt.Sprintf(format, args...)
	return err
}

// WithMetadata 设置元数据.
func (err *ErrorXCompat) WithMetadata(md map[string]string) *ErrorXCompat {
	err.Metadata = md
	return err
}

// KV 使用 key-value 对设置元数据.
func (err *ErrorXCompat) KV(kvs ...string) *ErrorXCompat {
	if err.Metadata == nil {
		err.Metadata = make(map[string]string) // 初始化元数据映射
	}

	for i := 0; i < len(kvs); i += 2 {
		// kvs 必须是成对的
		if i+1 < len(kvs) {
			err.Metadata[kvs[i]] = kvs[i+1]
		}
	}
	return err
}

// WithRequestID 设置请求 ID.
func (err *ErrorXCompat) WithRequestID(requestID string) *ErrorXCompat {
	return err.KV("X-Request-ID", requestID) // 设置请求 ID
}

// Is 判断当前错误是否与目标错误匹配.
func (err *ErrorXCompat) Is(target error) bool {
	if errx := new(ErrorXCompat); errors.As(target, &errx) {
		return errx.Code == err.Code && errx.Reason == err.Reason
	}
	return false
}
