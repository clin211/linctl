package errno

import (
	"errors"
	"fmt"
	"net/http"
)

// BizError 表示带有业务错误码、原因与消息的业务级错误。
type BizError struct {
	// HTTP 状态码
	HTTPCode int
	// 机器可读的业务错误码（如 100001）
	Code int
	// 机器可读的原因（如 "User.NotFound"）
	Reason string
	// 人类可读的消息
	Message string
	// 可选的详细信息
	Details string
}

// Error 实现 error 接口。
func (e *BizError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s (%s)", e.Code, e.Reason, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s: %s", e.Code, e.Reason, e.Message)
}

// WithDetails 返回一个附带额外 details 的 BizError 副本。
func (e *BizError) WithDetails(details string) *BizError {
	cp := *e
	cp.Details = details
	return &cp
}

// New 创建一个新的 BizError。
func New(httpCode, code int, reason, message string) *BizError {
	return &BizError{
		HTTPCode: httpCode,
		Code:     code,
		Reason:   reason,
		Message:  message,
	}
}

// FromError 将标准 error 转换为 BizError（无法识别时回退为 ErrInternal）。
func FromError(err error) *BizError {
	if err == nil {
		return nil
	}
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return ErrInternal.WithDetails(err.Error())
}

// 标准错误集合。
var (
	ErrInternal          = New(http.StatusInternalServerError, 500001, "Internal.ServerError", "Internal server error.")
	ErrNotFound          = New(http.StatusNotFound, 404001, "NotFound.Resource", "Resource not found.")
	ErrBind              = New(http.StatusBadRequest, 400001, "Request.BindError", "Failed to bind request.")
	ErrInvalidArgument   = New(http.StatusBadRequest, 400002, "Request.InvalidArgument", "Invalid argument.")
	ErrUnauthenticated   = New(http.StatusUnauthorized, 401001, "Auth.Unauthenticated", "Unauthenticated.")
	ErrPermissionDenied  = New(http.StatusForbidden, 403001, "Auth.PermissionDenied", "Permission denied.")
)
