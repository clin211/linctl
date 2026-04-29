package errorsx

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 业务错误码，格式：LLMMMNNN
// LL: 错误级别（1-5）
// MMM: 业务模块（001-999）
// NNN: 具体错误（001-999）
type BizCode int32

// 错误级别定义
const (
	LevelSystem     = 1 // 系统级错误，需要开发介入
	LevelUser       = 2 // 用户操作错误，可提示用户
	LevelBusiness   = 3 // 业务逻辑错误，正常的业务流程
	LevelUpstream   = 4 // 上游服务错误
	LevelDownstream = 5 // 下游服务错误
)

// 业务模块定义
const (
	ModuleCommon   = 1 // 通用模块
	ModuleUser     = 1 // 用户模块
	ModulePost     = 1 // 博客文章模块
	ModuleComment  = 1 // 评论模块
	ModuleAuth     = 1 // 认证模块
	ModuleDatabase = 1 // 数据库模块
	ModuleCache    = 1 // 缓存模块
)

// 常用错误码定义
const (
	CodeOK                      BizCode = 0     // 成功
	CodeUserNotFound            BizCode = 20101 // 用户不存在 (Level=2, Module=01, Error=01)
	CodeUserAlreadyExists       BizCode = 20102 // 用户已存在
	CodeUserInvalidCredentials  BizCode = 20103 // 用户名或密码错误
	CodeUserInsufficientBalance BizCode = 20104 // 用户余额不足
	CodeUserInvalidUsername     BizCode = 20105 // 用户名无效
	CodeUserInvalidPassword     BizCode = 20106 // 密码无效
	CodeUserPermissionDenied    BizCode = 20107 // 用户权限不足

	CodePostNotFound         BizCode = 30101 // 文章不存在 (Level=3, Module=01, Error=01)
	CodePostAlreadyPublished BizCode = 30102 // 文章已发布
	CodePostPermissionDenied BizCode = 30103 // 文章权限不足

	CodeCommentNotFound         BizCode = 40101 // 评论不存在 (Level=4, Module=01, Error=01)
	CodeCommentPermissionDenied BizCode = 40102 // 评论权限不足

	CodeAuthUnauthenticated BizCode = 50101 // 未认证 (Level=5, Module=01, Error=01)
	CodeAuthTokenInvalid    BizCode = 50102 // Token无效
	CodeAuthTokenExpired    BizCode = 50103 // Token过期
	CodeAuthSignToken       BizCode = 50104 // Token签名失败

	CodeDatabaseConnectFailed BizCode = 60101 // 数据库连接失败 (Level=6, Module=01, Error=01)
	CodeDatabaseReadFailed    BizCode = 60102 // 数据库读取失败
	CodeDatabaseWriteFailed   BizCode = 60103 // 数据库写入失败

	CodeCacheConnectFailed BizCode = 70101 // 缓存连接失败 (Level=7, Module=01, Error=01)
	CodeCacheReadFailed    BizCode = 70102 // 缓存读取失败
	CodeCacheWriteFailed   BizCode = 70103 // 缓存写入失败

	CodeInternalServer     BizCode = 10101 // 内部服务器错误 (Level=1, Module=01, Error=01)
	CodeServiceUnavailable BizCode = 10102 // 服务不可用
	CodeRequestTimeout     BizCode = 10103 // 请求超时
	CodeTooManyRequests    BizCode = 10104 // 请求过于频繁
)

// BizError 业务错误结构
type BizError struct {
	// 业务错误码
	Code BizCode `json:"code"`

	// 错误级别
	Level int `json:"level"`

	// 错误原因（英文，用于日志和监控）
	Reason string `json:"reason,omitempty"`

	// 用户友好消息（中文，用于前端显示）
	Message string `json:"message,omitempty"`

	// 详细信息
	Details string `json:"details,omitempty"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// APIResponse 标准API响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Reason  string      `json:"reason,omitempty"`
}

// ResponseDetail 响应详情（用于替代 ErrorDetail）
type ResponseDetail struct {
	Details  string                 `json:"details,omitempty"`
	StackID  string                 `json:"stack_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Headers 常用的HTTP Headers
const (
	HeaderRequestID    = "X-Request-ID"
	HeaderTimestamp    = "X-Timestamp"
	HeaderResponseTime = "X-Response-Time"
	HeaderServerID     = "X-Server-ID"
	HeaderTraceID      = "X-Trace-ID"
)

// GetErrorLevel 从错误码中提取错误级别
func GetErrorLevel(code BizCode) int {
	if code == 0 {
		return 0
	}
	return int(code/10000) % 10
}

// GetModule 从错误码中提取业务模块
func GetModule(code BizCode) int {
	if code == 0 {
		return 0
	}
	return int(code/100) % 100
}

// GetHTTPCode 根据错误码映射对应的HTTP状态码
func GetHTTPCode(code BizCode) int {
	if code == CodeOK {
		return http.StatusOK
	}

	level := GetErrorLevel(code)
	switch level {
	case LevelSystem:
		return http.StatusInternalServerError
	case LevelUpstream, LevelDownstream:
		return http.StatusBadGateway
	case LevelUser, LevelBusiness:
		// 业务错误统一返回200，通过code字段区分
		return http.StatusOK
	default:
		return http.StatusInternalServerError
	}
}

// NewBizError 创建新的业务错误
func NewBizError(code BizCode, reason, message string) *BizError {
	return &BizError{
		Code:    code,
		Level:   GetErrorLevel(code),
		Reason:  reason,
		Message: message,
	}
}

// WithDetails 设置错误详情
func (err *BizError) WithDetails(details string) *BizError {
	err.Details = details
	return err
}

// WithMetadata 设置元数据
func (err *BizError) WithMetadata(metadata map[string]interface{}) *BizError {
	err.Metadata = metadata
	return err
}

// WithMessage 设置错误消息（创建新实例避免内存共享问题）
func (err *BizError) WithMessage(message string) *BizError {
	// 创建新的错误实例，避免内存共享问题
	newErr := &BizError{
		Code:     err.Code,
		Level:    err.Level,
		Reason:   err.Reason,
		Message:  message,
		Details:  err.Details,
		Metadata: err.Metadata,
	}

	// 如果有元数据，创建副本避免共享
	if err.Metadata != nil {
		newErr.Metadata = make(map[string]interface{})
		for k, v := range err.Metadata {
			newErr.Metadata[k] = v
		}
	}

	return newErr
}

// Error 实现 error 接口
func (err *BizError) Error() string {
	return fmt.Sprintf("biz error: code=%d reason=%s message=%s", err.Code, err.Reason, err.Message)
}

// Is 判断错误是否匹配
func (err *BizError) Is(target error) bool {
	if bizErr, ok := target.(*BizError); ok {
		return bizErr.Code == err.Code
	}
	return false
}

// ToGRPCCode converts HTTP status code to gRPC status code
func ToGRPCCode(code int) codes.Code {
	switch code {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	default:
		return codes.Unknown
	}
}

// FromGRPCCode converts gRPC status code to HTTP status code
func FromGRPCCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Success 创建成功响应
func Success(data interface{}, message ...string) *APIResponse {
	msg := "success"
	if len(message) > 0 {
		msg = message[0]
	}
	return &APIResponse{
		Code:    0,
		Message: msg,
		Data:    data,
	}
}

// Failure 创建失败响应
func Failure(code int, message string) *APIResponse {
	return &APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// WithReason 添加错误原因
func (resp *APIResponse) WithReason(reason string) *APIResponse {
	resp.Reason = reason
	return resp
}

// FromBizError 将业务错误转换为API响应
func FromBizError(err *BizError) *APIResponse {
	resp := Failure(int(err.Code), err.Message)
	resp.Reason = err.Reason
	return resp
}

// GRPCStatus 返回 gRPC 状态表示
func (err *BizError) GRPCStatus() *status.Status {
	details := errdetails.ErrorInfo{Reason: err.Reason}
	if err.Metadata != nil {
		details.Metadata = make(map[string]string)
		for k, v := range err.Metadata {
			details.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	s, _ := status.New(ToGRPCCode(GetHTTPCode(err.Code)), err.Message).WithDetails(&details)
	return s
}

// Code 从错误中提取错误码
func Code(err error) int {
	if err == nil {
		return 0
	}
	if bizErr := new(BizError); errors.As(err, &bizErr) {
		return int(bizErr.Code)
	}
	return int(CodeInternalServer)
}

// Reason 从错误中提取错误原因
func Reason(err error) string {
	if err == nil {
		return ""
	}
	if bizErr := new(BizError); errors.As(err, &bizErr) {
		return bizErr.Reason
	}
	return "InternalError"
}

// FromError 尝试将通用错误转换为业务错误
func FromError(err error) *BizError {
	if err == nil {
		return nil
	}

	// 如果已经是BizError，直接返回
	if bizErr := new(BizError); errors.As(err, &bizErr) {
		return bizErr
	}

	// 处理兼容版本的 ErrorXCompat
	if compatErr := new(ErrorXCompat); errors.As(err, &compatErr) {
		// 将旧版本错误映射到新的错误码
		bizCode := BizCode(11001) // 默认系统错误
		switch compatErr.Code {
		case http.StatusBadRequest:
			bizCode = CodeUserInvalidCredentials
		case http.StatusNotFound:
			bizCode = CodeUserNotFound
		case http.StatusUnauthorized:
			bizCode = CodeAuthUnauthenticated
		case http.StatusForbidden:
			bizCode = CodeUserPermissionDenied
		case http.StatusInternalServerError:
			bizCode = CodeInternalServer
		}

		bizErr := NewBizError(bizCode, compatErr.Reason, compatErr.Message)
		if compatErr.Metadata != nil {
			bizErr.Metadata = make(map[string]interface{})
			for k, v := range compatErr.Metadata {
				bizErr.Metadata[k] = v
			}
		}
		return bizErr
	}

	// 处理 gRPC 错误
	gs, ok := status.FromError(err)
	if ok {
		code := FromGRPCCode(gs.Code())
		bizCode := BizCode(11001) // 默认系统错误
		if code == http.StatusBadRequest {
			bizCode = CodeUserInvalidCredentials
		}

		bizErr := NewBizError(bizCode, "gRPC", gs.Message())

		// 提取详细信息
		for _, detail := range gs.Details() {
			if typed, ok := detail.(*errdetails.ErrorInfo); ok {
				bizErr.Reason = typed.Reason
				metadata := make(map[string]interface{})
				for k, v := range typed.Metadata {
					metadata[k] = v
				}
				bizErr.Metadata = metadata
				break
			}
		}

		return bizErr
	}

	// 其他未知错误
	return NewBizError(CodeInternalServer, "Unknown", err.Error())
}
