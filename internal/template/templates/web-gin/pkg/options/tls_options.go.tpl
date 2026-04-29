package options

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

var _ IOptions = (*TLSOptions)(nil)

// TLSOptions 是 TLS 加密相关的配置项。
type TLSOptions struct {
	// UseTLS 表示是否启用 TLS。
	UseTLS             bool   `json:"use-tls" mapstructure:"use-tls"`
	InsecureSkipVerify bool   `json:"insecure-skip-verify" mapstructure:"insecure-skip-verify"`
	CaCert             string `json:"ca-cert" mapstructure:"ca-cert"`
	Cert               string `json:"cert" mapstructure:"cert"`
	Key                string `json:"key" mapstructure:"key"`
}

// NewTLSOptions 构造一个零值 *TLSOptions。
func NewTLSOptions() *TLSOptions {
	return &TLSOptions{}
}

// Validate 校验 TLSOptions 的参数。
func (o *TLSOptions) Validate() []error {
	errs := []error{}

	if !o.UseTLS {
		return errs
	}

	if (o.Cert != "" && o.Key == "") || (o.Cert == "" && o.Key != "") {
		errs = append(errs, fmt.Errorf("only one of cert and key configuration option is setted, you should set both to enable tls"))
	}

	return errs
}

// AddFlags 把 TLSOptions 上的字段注册为命令行 flag。
func (o *TLSOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.UseTLS, fullPrefix+".use-tls", o.UseTLS, "是否使用 TLS 连接服务器")
	fs.BoolVar(&o.InsecureSkipVerify, fullPrefix+".insecure-skip-verify", o.InsecureSkipVerify,
		"客户端是否跳过对服务端证书链与主机名的校验")
	fs.StringVar(&o.CaCert, fullPrefix+".ca-cert", o.CaCert, "连接服务器使用的 CA 证书路径")
	fs.StringVar(&o.Cert, fullPrefix+".cert", o.Cert, "连接服务器使用的客户端证书路径")
	fs.StringVar(&o.Key, fullPrefix+".key", o.Key, "连接服务器使用的客户端私钥路径")
}

// MustTLSConfig 调用 TLSConfig 并屏蔽错误（出错时返回零值 *tls.Config）。
func (o *TLSOptions) MustTLSConfig() *tls.Config {
	tlsConf, err := o.TLSConfig()
	if err != nil {
		return &tls.Config{}
	}

	return tlsConf
}

// TLSConfig 根据 TLSOptions 构造一个 *tls.Config。
//
// 当 UseTLS=false 时返回 (nil, nil)，调用方应自行降级到明文链路。
func (o *TLSOptions) TLSConfig() (*tls.Config, error) {
	if !o.UseTLS {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: o.InsecureSkipVerify,
	}

	if o.Cert != "" && o.Key != "" {
		var cert tls.Certificate
		cert, err := tls.LoadX509KeyPair(o.Cert, o.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to loading tls certificates: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if o.CaCert != "" {
		data, err := os.ReadFile(o.CaCert)
		if err != nil {
			return nil, err
		}

		capool := x509.NewCertPool()
		for {
			var block *pem.Block
			block, _ = pem.Decode(data)
			if block == nil {
				break
			}
			cacert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			capool.AddCert(cacert)
		}

		tlsConfig.RootCAs = capool
	}

	return tlsConfig, nil
}

// Scheme 根据 TLS 配置返回 URL 协议（http / https）。
func (o *TLSOptions) Scheme() string {
	if o.UseTLS {
		return "https"
	}
	return "http"
}
