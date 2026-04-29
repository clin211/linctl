/*
Package contextx 是 context 的扩展工具包，用于在 context 中安全地传递与请求相关
的用户信息（用户 ID、用户名、access token 等）。

包名后缀 "x" 表示扩展（extension），便于在引用时与标准库 `context` 区分。
此包内的辅助函数让中间件 / 业务函数可以方便地把用户信息塞入 context、再在下游
函数中取出，避免使用全局变量或在大量函数中显式传递参数。

典型用法：

	// 创建一个新的 context
	ctx := context.Background()

	// 把用户 ID 与用户名写入 context
	ctx = contextx.WithUserID(ctx, "user-xxxx")
	ctx = contextx.WithUsername(ctx, "sampleUser")

	// 在下游从 context 中取出用户信息
	userID := contextx.UserID(ctx)
	username := contextx.Username(ctx)
*/
package contextx // import "{{ .Project.Metadata.Module }}/internal/pkg/contextx"
