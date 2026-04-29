package linctlerr

// 错误码常量定义。
//
// 命名规则（SSOT §1.1 + §5.X）：
//   - 常量名 Err<DescriptiveName>（与 Go 标准库 errors 风格一致）
//   - 字符串值 snake_case（便于 JSON / YAML / 日志 grep）
//   - 与 docs/03-cli-design.md §3.6 退出码表对应
const (
	// ErrConfigInvalid 表示配置文件无效（schema invalid / missing required）
	// → 退出码 2
	ErrConfigInvalid Code = "config_invalid"

	// ErrComponentExists 表示尝试创建已存在的 Component → 退出码 4（冲突）
	ErrComponentExists Code = "component_exists"

	// ErrComponentNotFound 表示引用了不存在的 Component → 退出码 3
	ErrComponentNotFound Code = "component_not_found"

	// ErrTemplateRender 表示模板渲染失败 → 退出码 1
	ErrTemplateRender Code = "template_render"

	// ErrFileConflict 表示文件冲突（apply 时无 strategy） → 退出码 4
	ErrFileConflict Code = "file_conflict"

	// ErrASTInjection 表示 AST 注入失败 → 退出码 1
	ErrASTInjection Code = "ast_injection"

	// ErrEnvironment 表示环境错误（go/git 未安装等） → 退出码 5
	ErrEnvironment Code = "environment"

	// ErrNetwork 表示网络错误（plugin install 等） → 退出码 6
	ErrNetwork Code = "network"

	// ErrSecurityPolicy 表示安全策略违规（hook policy violation, CI restricted 强制失败等）
	// → 退出码 7（详见 docs/META-fix-decisions-2026-04-25.md §5.6）
	ErrSecurityPolicy Code = "security_policy"

	// ErrFeatureDependency 表示 Feature 依赖错误（环、未注册依赖等）
	ErrFeatureDependency Code = "feature_dependency"

	// ErrUnsafePath 表示路径越界（SafeJoin 校验失败）
	ErrUnsafePath Code = "unsafe_path"

	// ErrLockTimeout 表示获取项目锁超时（详见 docs/06-codegen-pipeline.md §6.13）
	ErrLockTimeout Code = "lock_timeout"

	// ErrPlanDigestMismatch 表示 apply --plan 时 digest 与当前 plan 不一致
	// （详见 SSOT §1.12）
	ErrPlanDigestMismatch Code = "plan_digest_mismatch"

	// ErrNotImplementedYet 表示 schema 中合法但当前 Phase 尚未实现的能力
	// （详见 docs/11-implementation-plan.md §11.2.0）
	ErrNotImplementedYet Code = "not_implemented_yet"

	// ErrInternal 表示 linctl 自身的内部错误（应当报告 bug）
	ErrInternal Code = "internal"
)
