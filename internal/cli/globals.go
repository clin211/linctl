package cli

import (
	"github.com/spf13/pflag"
)

// GlobalOptions 是所有子命令共享的全局 flag 集合（详见 docs/03-cli-design.md §3.2）。
//
// SSOT 规范：
//   - --output / -o 是全局机器可解析输出 flag（取值 text/json/yaml）
//   - --hook-policy 是 Hook 安全策略（restricted/confirm/unrestricted）
//   - --hook-strategy 是 Hook 执行流程（confirm/auto/skip），与 policy 正交
//   - --lock-timeout 项目锁等待超时
//   - --strict PairBuilder 同文件覆盖时直接 fail
type GlobalOptions struct {
	// 通用控制
	ConfigPath string
	RootDir    string
	LogLevel   string
	LogFormat  string
	Output     string
	NoColor    bool
	NoEmoji    bool
	DryRun     bool
	Yes        bool
	Verbose    bool

	// 诊断与可观测
	Debug      string
	DebugOut   string
	Profile    string
	ProfileOut string

	// 安全与 Hook
	HookPolicy   string
	HookStrategy string
	LockTimeout  string
	Strict       bool
	NoBackup     bool

	// 国际化
	Lang string

	// 环境覆盖
	Env string
}

// RegisterFlags 把 GlobalOptions 注册到给定 FlagSet（通常是 cobra 的 PersistentFlags）。
//
// 默认值与 docs/03-cli-design.md §3.2 表格严格一致。
func (o *GlobalOptions) RegisterFlags(fs *pflag.FlagSet) {
	// §3.2.1 通用控制
	fs.StringVar(&o.ConfigPath, "config", "linctl.yaml", "项目配置文件路径")
	fs.StringVar(&o.RootDir, "root-dir", ".", "项目根目录")
	fs.StringVar(&o.LogLevel, "log-level", "info", "日志级别：trace/debug/info/warn/error")
	fs.StringVar(&o.LogFormat, "log-format", "text", "日志格式：text/json")
	fs.StringVarP(&o.Output, "output", "o", "text",
		"机器可解析输出格式：text(人类可读) / json / yaml")
	fs.BoolVar(&o.NoColor, "no-color", false, "禁用彩色输出")
	fs.BoolVar(&o.NoEmoji, "no-emoji", false, "禁用 emoji")
	fs.BoolVar(&o.DryRun, "dry-run", false, "不写文件，仅打印将要发生的变更")
	fs.BoolVarP(&o.Yes, "yes", "y", false,
		"自动同意所有文件冲突确认（不跳过 hook 确认）")
	fs.BoolVarP(&o.Verbose, "verbose", "v", false, "详细输出（≈ --log-level=debug）")
	fs.StringVar(&o.Env, "env", "",
		"启用环境覆盖：加载 linctl.<env>.yaml")

	// §3.2.2 诊断与可观测
	fs.StringVar(&o.Debug, "debug", "",
		"模块级 trace 开关（逗号分隔，如 template,ast,fs；* 全部）")
	fs.StringVar(&o.DebugOut, "debug-out", "text", "trace 输出格式：text/json/tree")
	fs.StringVar(&o.Profile, "profile", "",
		"启用 pprof：cpu/mem/goroutine/block/mutex（逗号分隔）")
	fs.StringVar(&o.ProfileOut, "profile-out", "_output/profile/{type}.pb.gz",
		"profile 输出路径（{type} 为占位符）")

	// §3.2.3 安全与 Hook
	fs.StringVar(&o.HookPolicy, "hook-policy", "",
		"Hook 安全策略：restricted/confirm/unrestricted。空值=本地 confirm，CI=restricted")
	fs.StringVar(&o.HookStrategy, "hook-strategy", "confirm",
		"Hook 执行流程：confirm/auto/skip（与 --hook-policy 正交）")
	fs.StringVar(&o.LockTimeout, "lock-timeout", "30s",
		"项目锁等待超时（如 30s, 1m）")
	fs.BoolVar(&o.Strict, "strict", false,
		"严格模式：PairBuilder 同文件覆盖时直接 fail")
	fs.BoolVar(&o.NoBackup, "no-backup", false,
		"apply 前不创建 .linctl/backups/<ts>/（不推荐）")

	// §3.2.4 国际化
	fs.StringVar(&o.Lang, "lang", "", "CLI 输出语言：en/zh-CN（自动检测 LANG）")
}
