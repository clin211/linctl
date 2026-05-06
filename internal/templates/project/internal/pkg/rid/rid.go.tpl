package rid

import (
	"github.com/clin211/linhub/id"
)

// defaultABC 是 ResourceID.New 默认使用的字符集。
const defaultABC = "abcdefghijklmnopqrstuvwxyz1234567890"

// ResourceID 是资源类型的标识符，与生成的实例 ID 共同构成完整的业务 ID。
//
// 例如：`user-abc123` 表示一个用户资源的 ID，前缀 `user` 即为 ResourceID。
type ResourceID string

// String 将 ResourceID 转换为字符串。
func (rid ResourceID) String() string {
	return string(rid)
}

// New 基于 counter 生成一个带前缀的唯一 ID，前缀为 ResourceID。
//
// 默认实现依赖 linhub.id（基于 sonyflake 与自定义编码生成短 ID）。
// 如需自定义字符集 / 长度 / Salt，请直接调用 id.NewCode 并传入对应选项。
func (rid ResourceID) New(counter uint64) string {
	uniqueStr := id.NewCode(
		counter,
		id.WithCodeChars([]rune(defaultABC)),
		id.WithCodeL(6),
		id.WithCodeSalt(Salt()),
	)
	return rid.String() + "-" + uniqueStr
}
