package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"{{ .Project.Metadata.Module }}/cmd/{{ .Component.Name }}/app/options"
	"{{ .Project.Metadata.Module }}/pkg/core"
	"{{ .Project.Metadata.Module }}/pkg/version"
)

const (
	// defaultHomeDir 是 {{ .Component.Name }} 服务默认的配置目录（相对用户 home）。
	defaultHomeDir = ".{{ .Project.Metadata.Name }}"

	// defaultConfigName 是 {{ .Component.Name }} 默认配置文件名。
	defaultConfigName = "{{ .Component.Name }}.yaml"

	// envPrefix 是覆盖配置文件用的环境变量前缀。
	//
	// 命名规则：`<PROJECT>_<COMPONENT>`，与 miniblog-v4 的 `MINIBLOG-V4_APISERVER`
	// 保持一致；项目名保留 kebab-case 是为了让 ToLowerKebab 反推回 yaml key 时
	// 不丢分词信息（viper.SetEnvKeyReplacer 中 `-` / `_` 会被同等替换）。
	envPrefix = "{{ upperKebab .Project.Metadata.Name }}_{{ upper .Component.Name }}"
)

// configFile 保存 --config 指定的配置文件路径。
var configFile string

// NewWebServerCommand 构造启动应用所需的 *cobra.Command。
func NewWebServerCommand() *cobra.Command {
	// 用默认值初始化命令行选项。
	opts := options.NewServerOptions()

	cmd := &cobra.Command{
		Use:          "{{ .Component.Name }}",
		Short:        "{{ .Component.Name }} HTTP API 服务",
		Long:         "{{ .Component.Name }} HTTP API 服务，由 linctl 脚手架生成。",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := genericapiserver.SetupSignalContext()

			// 处理 `--version` 与 `--version=raw`。
			version.PrintAndExitIfRequested()

			// 把 viper 中的配置反序列化到 opts 上。
			if err := viper.Unmarshal(opts); err != nil {
				return fmt.Errorf("failed to unmarshal configuration: %w", err)
			}

			// 校验命令行参数与配置文件。
			if err := opts.Validate(); err != nil {
				return fmt.Errorf("invalid options: %w", err)
			}
			// 启动 OpenTelemetry 全局 provider。
			if err := opts.OTelOptions.Apply(); err != nil {
				return err
			}
			defer func() {
				_ = opts.OTelOptions.Shutdown(ctx)
			}()

			return run(ctx, opts)
		},
		Args: cobra.NoArgs,
	}

	// 每次命令执行前都会触发：从配置文件 + 环境变量加载到 viper。
	cobra.OnInitialize(core.OnInitialize(&configFile, envPrefix, searchDirs(), defaultConfigName))

	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", filePath(),
		"{{ .Component.Name }} 配置文件路径")

	// 把 ServerOptions 上的字段绑定为命令行 flag。
	opts.AddFlags(cmd.PersistentFlags())
	// 注册 `--version` flag。
	version.AddFlags(cmd.PersistentFlags())

	return cmd
}

// run 是初始化并运行服务器的主体逻辑。
func run(ctx context.Context, opts *options.ServerOptions) error {
	cfg, err := opts.Config()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	server, err := cfg.NewServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return server.Run(ctx)
}

// searchDirs 返回默认配置文件的搜索路径列表（用户 home 下的 defaultHomeDir + 当前目录）。
func searchDirs() []string {
	homeDir, err := os.UserHomeDir()
	cobra.CheckErr(err)
	return []string{filepath.Join(homeDir, defaultHomeDir), "."}
}

// filePath 返回默认配置文件的绝对路径，作为 --config 的默认值。
func filePath() string {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	return filepath.Join(home, defaultHomeDir, defaultConfigName)
}
