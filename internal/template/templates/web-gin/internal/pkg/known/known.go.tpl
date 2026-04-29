package known

// HTTP / gRPC 头部常量。
//
// gRPC 底层基于 HTTP/2，规范要求 header key 全部为小写；HTTP/1.x 大多保留用户大小写，
// 但部分 web server / 代理为了简化处理也会强制小写。为了在两种协议下都不出问题，
// 这里统一使用小写。前缀 "x-" 表示自定义头部。
const (
	// XRequestID 是请求 ID 在 context / header 中的 key。
	XRequestID = "x-request-id"

	// XUserID 是请求用户 ID 在 context / header 中的 key。
	// UserID 在用户整个生命周期内是唯一的。
	XUserID = "x-user-id"

	// XUsername 是请求用户名在 context / header 中的 key。
	XUsername = "x-username"
)

// 其他公共常量。
const (
	// AdminUsername 是预置管理员账号名。
	AdminUsername = "root"

	// MaxErrGroupConcurrency 限制 errgroup 同时运行的 goroutine 上限，
	// 用于防止资源耗尽、提升程序稳定性。可按业务需要调整。
	MaxErrGroupConcurrency = 1000
)
