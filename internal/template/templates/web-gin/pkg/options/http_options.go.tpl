package options

import (
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*HTTPOptions)(nil)

// HTTPOptions 包含 HTTP 服务器启动相关的配置项。
type HTTPOptions struct {
	// Network 是服务监听的网络类型。
	Network string `json:"network" mapstructure:"network"`

	// Addr 是服务监听地址。
	Addr string `json:"addr" mapstructure:"addr"`

	// Timeout 是服务的超时时长，HTTP 客户端侧也会用到。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewHTTPOptions 用默认值构造 *HTTPOptions。
func NewHTTPOptions() *HTTPOptions {
	return &HTTPOptions{
		Network: "tcp",
		Addr:    "0.0.0.0:38443",
		Timeout: 30 * time.Second,
	}
}

// Validate 校验命令行 / 配置文件中的参数是否合法，启动时调用。
func (o *HTTPOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errors := []error{}

	if err := ValidateAddress(o.Addr); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// AddFlags 把 HTTP 相关的字段注册成命令行 flag，挂在指定 FlagSet 上，
// 并使用 fullPrefix 作为 flag 名前缀。
//
// 示例：
//
//	o.AddFlags(fs, "apiserver.http")  // --apiserver.http.network、--apiserver.http.addr 等
//	o.AddFlags(fs, "gateway.http")    // --gateway.http.network、--gateway.http.addr 等
func (o *HTTPOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Network, fullPrefix+".network", o.Network,
		"HTTP 服务的网络类型（tcp / tcp4 / tcp6）")
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr,
		"HTTP 服务监听地址（例如 :8080、0.0.0.0:8443）")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout,
		"传入 HTTP 连接的超时时长")
}

// Complete 在所有参数加载完之后填充必要的默认值；当前为空。
func (s *HTTPOptions) Complete() error {
	return nil
}
