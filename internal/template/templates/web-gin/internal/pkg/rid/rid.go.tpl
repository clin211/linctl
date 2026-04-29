package rid

import (
	"{{ .Project.Metadata.Module }}/pkg/id"
)

// defaultABC 是生成短码时使用的默认字符表（小写字母 + 数字）。
const defaultABC = "abcdefghijklmnopqrstuvwxyz1234567890"

// ResourceID 是资源 ID 的语义类型，用作生成 ID 时的前缀。
type ResourceID string

const (
	// UserID 表示用户资源的 ID 前缀。
	UserID ResourceID = "user"
)

// String 把资源 ID 转换为字符串。
func (rid ResourceID) String() string {
	return string(rid)
}

// New 生成一个带前缀的全局唯一 ID（前缀 + "-" + 短码）。
//
// counter 通常使用 sonyflake 之类分布式唯一序列号；短码长度固定 6 位、
// 字符集来自 defaultABC，通过 Salt() 加盐避免可枚举。
func (rid ResourceID) New(counter uint64) string {
	uniqueStr := id.NewCode(
		counter,
		id.WithCodeChars([]rune(defaultABC)),
		id.WithCodeL(6),
		id.WithCodeSalt(Salt()),
	)
	return rid.String() + "-" + uniqueStr
}
