package binding

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// 包内变量。
var (
	// originalValidator 保存 Gin 原始的 validator，留给最终统一校验时使用。
	originalValidator binding.StructValidator

	// initOnce 保证只在第一次 import 包时捕获原始 validator。
	initOnce sync.Once
)

// init 在包加载时捕获原始 validator，并把全局 validator 置为 nil，
// 这样 Gin 在多次绑定的过程中就不会触发分散的校验。
func init() {
	// 用 Once 防止多 goroutine 同时进入。
	initOnce.Do(func() {
		// 保存原始 validator
		originalValidator = binding.Validator

		// 把全局 validator 置为 nil，跳过绑定阶段的校验
		binding.Validator = nil
	})
}

// Bind 依次执行多个绑定函数（不在中间做校验），最后再统一校验一次。
//
// 它解决的问题：当某个字段来自 URI、另一个字段来自 JSON Body 时，单独从 URI
// 绑定就会触发校验失败（JSON 字段尚未注入），导致整个请求被拒。
//
// 用法示例：
//
//	var req UserUpdateRequest
//	if err := binding.Bind(c, &req, binding.URI, binding.JSON); err != nil {
//	    c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
//	    return
//	}
func Bind(c *gin.Context, obj interface{}, bindFuncs ...func(*gin.Context, interface{}) error) error {
	// 依次执行绑定（校验已被禁用）。
	for _, bindFunc := range bindFuncs {
		if err := bindFunc(c, obj); err != nil {
			return err // 仅返回解析错误
		}
	}

	// 全部绑定完成后再做一次完整校验。
	if originalValidator != nil {
		return originalValidator.ValidateStruct(obj)
	}

	return nil
}

// 下面是常用的绑定函数，配合 Bind 使用。

// URI 把 URI 参数绑定到 obj，使用 Gin 的 ShouldBindUri 但不做校验。
func URI(c *gin.Context, obj interface{}) error {
	return c.ShouldBindUri(obj)
}

// JSON 把请求体（JSON）绑定到 obj，使用 Gin 的 ShouldBindJSON 但不做校验。
func JSON(c *gin.Context, obj interface{}) error {
	return c.ShouldBindJSON(obj)
}
