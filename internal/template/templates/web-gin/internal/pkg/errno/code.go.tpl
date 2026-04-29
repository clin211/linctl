package errno

import (
	"{{ .Project.Metadata.Module }}/pkg/errorsx"
)

// 重新导出错误码和预定义错误，方便业务层使用
var (
	// 错误码
	CodeOK                      = errorsx.CodeOK
	CodeUserNotFound            = errorsx.CodeUserNotFound
	CodeUserAlreadyExists       = errorsx.CodeUserAlreadyExists
	CodeUserInvalidCredentials  = errorsx.CodeUserInvalidCredentials
	CodeUserInsufficientBalance = errorsx.CodeUserInsufficientBalance
	CodeUserInvalidUsername     = errorsx.CodeUserInvalidUsername
	CodeUserInvalidPassword     = errorsx.CodeUserInvalidPassword
	CodeUserPermissionDenied    = errorsx.CodeUserPermissionDenied

	CodePostNotFound         = errorsx.CodePostNotFound
	CodePostAlreadyPublished = errorsx.CodePostAlreadyPublished
	CodePostPermissionDenied = errorsx.CodePostPermissionDenied

	CodeCommentNotFound         = errorsx.CodeCommentNotFound
	CodeCommentPermissionDenied = errorsx.CodeCommentPermissionDenied

	CodeAuthUnauthenticated = errorsx.CodeAuthUnauthenticated
	CodeAuthTokenInvalid    = errorsx.CodeAuthTokenInvalid
	CodeAuthTokenExpired    = errorsx.CodeAuthTokenExpired
	CodeAuthSignToken       = errorsx.CodeAuthSignToken

	CodeDatabaseConnectFailed = errorsx.CodeDatabaseConnectFailed
	CodeDatabaseReadFailed    = errorsx.CodeDatabaseReadFailed
	CodeDatabaseWriteFailed   = errorsx.CodeDatabaseWriteFailed

	CodeCacheConnectFailed = errorsx.CodeCacheConnectFailed
	CodeCacheReadFailed    = errorsx.CodeCacheReadFailed
	CodeCacheWriteFailed   = errorsx.CodeCacheWriteFailed

	CodeInternalServer     = errorsx.CodeInternalServer
	CodeServiceUnavailable = errorsx.CodeServiceUnavailable
	CodeRequestTimeout     = errorsx.CodeRequestTimeout
	CodeTooManyRequests    = errorsx.CodeTooManyRequests

	// 预定义错误
	OK = errorsx.OK

	ErrInternal        = errorsx.ErrInternal
	ErrNotFound        = errorsx.ErrNotFound
	ErrBind            = errorsx.ErrBind
	ErrInvalidArgument = errorsx.ErrInvalidArgument
	ErrUnauthenticated = errorsx.ErrUnauthenticated
	ErrPermissionDenied = errorsx.ErrPermissionDenied
	ErrOperationFailed = errorsx.ErrOperationFailed

	// 数据库错误
	ErrDBRead    = errorsx.NewBizError(errorsx.CodeDatabaseReadFailed, "Database.ReadFailed", "Database read failure.")
	ErrDBWrite   = errorsx.NewBizError(errorsx.CodeDatabaseWriteFailed, "Database.WriteFailed", "Database write failure.")
	ErrDBConnect = errorsx.NewBizError(errorsx.CodeDatabaseConnectFailed, "Database.ConnectFailed", "Database connection failed.")

	// 缓存错误
	ErrCacheRead    = errorsx.NewBizError(errorsx.CodeCacheReadFailed, "Cache.ReadFailed", "Cache read failure.")
	ErrCacheWrite   = errorsx.NewBizError(errorsx.CodeCacheWriteFailed, "Cache.WriteFailed", "Cache write failure.")
	ErrCacheConnect = errorsx.NewBizError(errorsx.CodeCacheConnectFailed, "Cache.ConnectFailed", "Cache connection failed.")

	// Token 相关错误
	ErrSignToken    = errorsx.NewBizError(errorsx.CodeAuthSignToken, "Auth.SignToken", "Error occurred while signing the JSON web token.")
	ErrTokenInvalid = errorsx.NewBizError(errorsx.CodeAuthTokenInvalid, "Auth.TokenInvalid", "Token was invalid.")
	ErrTokenExpired = errorsx.NewBizError(errorsx.CodeAuthTokenExpired, "Auth.TokenExpired", "Token was expired.")

	// 通用业务错误
	ErrPageNotFound       = errorsx.NewBizError(errorsx.CodeUserNotFound, "NotFound.PageNotFound", "Page not found.")
	ErrServiceUnavailable = errorsx.NewBizError(errorsx.CodeServiceUnavailable, "Service.Unavailable", "Service temporarily unavailable.")
	ErrTooManyRequests    = errorsx.NewBizError(errorsx.CodeTooManyRequests, "Service.TooManyRequests", "Too many requests, please try again later.")

	// 角色管理错误
	ErrAddRole    = errorsx.NewBizError(errorsx.CodeInternalServer, "Role.AddFailed", "Error occurred while adding the role.")
	ErrRemoveRole = errorsx.NewBizError(errorsx.CodeInternalServer, "Role.RemoveFailed", "Error occurred while removing the role.")
)
