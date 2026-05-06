package known

import "github.com/clin211/linhub/errx"

// HTTP 头常量（与 linhub errx / core 保持一致）。
const (
	// XRequestID 是唯一请求 ID 的 HTTP 头名称。
	XRequestID = errx.HeaderRequestID

	// XUserID 是已认证用户 ID 的 HTTP 头名称。
	XUserID = "x-user-id"

	// XUsername 是已认证用户名的 HTTP 头名称。
	XUsername = "x-username"
)

// 应用级常量。
const (
	// AdminUsername 是默认管理员用户名。
	AdminUsername = "root"

	// MaxErrGroupConcurrency 限制 errgroup 中的 goroutine 并发数。
	MaxErrGroupConcurrency = 1000
)
