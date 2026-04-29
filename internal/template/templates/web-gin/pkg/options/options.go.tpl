package options

import "github.com/spf13/pflag"

// IOptions 定义通用 Options 类型应实现的方法集。
type IOptions interface {
	// Validate 校验所有必填字段，必要时也可以在这里补全默认值。
	Validate() []error

	// AddFlags 把 Options 上的字段注册为命令行 flag，挂在指定 FlagSet 上，
	// 全部带 fullPrefix 前缀。
	//
	// fullPrefix 是完整前缀字符串，例如 "app.otel"；实现方在该前缀基础上
	// 追加自己的字段名，得到最终 flag 名，例如：
	//   --app.otel.endpoint
	//   --app.otel.insecure
	AddFlags(fs *pflag.FlagSet, fullPrefix string)
}
