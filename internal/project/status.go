package project

// ProjectState 是序列化到独立 `PROJECT` 文件的运行时状态镜像（SSOT §1.7 / docs/04-config-schema.md §4.3.6）。
//
// `linctl.yaml` 只含用户输入的 spec；`PROJECT` 由 linctl 工具维护，记录每次 plan/apply 之后的
// 元信息，例如 generatedAt / cliVersion / lastApplyHash 以及 schema 演进的历史 migrations。
//
// 两个文件均提交进 git，但用户**仅手动编辑** `linctl.yaml`，永不改 `PROJECT`。
//
// 仅同步 metadata.name 是为了一致性校验（与 `linctl.yaml` 的项目名比对，若不一致即拒绝
// 加载，提示用户运行 `linctl reconcile`）。
//
//nolint:revive // ProjectState 是面向 PROJECT 文件的明确名词，不应缩写为 State。
type ProjectState struct {
	APIVersion APIVersion `yaml:"apiVersion" json:"apiVersion"`
	Kind       string     `yaml:"kind"       json:"kind"`
	Metadata   Metadata   `yaml:"metadata"   json:"metadata"`
	Status     Status     `yaml:"status"     json:"status"`
}

// Status 是 PROJECT 文件中工具维护的运行时状态。所有字段均可选，
// 因为首次写入时 schemaMigrations 与 lastApplyHash 可能为空。
type Status struct {
	GeneratedAt      string            `yaml:"generatedAt,omitempty"      json:"generatedAt,omitempty"`
	CLIVersion       string            `yaml:"cliVersion,omitempty"       json:"cliVersion,omitempty"`
	SchemaMigrations []SchemaMigration `yaml:"schemaMigrations,omitempty" json:"schemaMigrations,omitempty"`
	LastApplyHash    string            `yaml:"lastApplyHash,omitempty"    json:"lastApplyHash,omitempty"`
}

// SchemaMigration 记录一次 apiVersion 演进。From / To 是 `linctl.dev/v1alpha1` 这种完整字符串；
// At 是 RFC3339 格式的时间戳。
type SchemaMigration struct {
	From APIVersion `yaml:"from" json:"from" validate:"required"`
	To   APIVersion `yaml:"to"   json:"to"   validate:"required"`
	At   string     `yaml:"at"   json:"at"   validate:"required"`
}

// IsEmpty 报告 Status 是否完全为零值（用于判断 PROJECT 文件是否首次创建）。
func (s Status) IsEmpty() bool {
	return s.GeneratedAt == "" &&
		s.CLIVersion == "" &&
		len(s.SchemaMigrations) == 0 &&
		s.LastApplyHash == ""
}

// AppendMigration 追加一条迁移记录，便于多次 upgrade 链式累加。原数组不变。
func (s Status) AppendMigration(m SchemaMigration) Status {
	out := s
	migrations := make([]SchemaMigration, 0, len(s.SchemaMigrations)+1)
	migrations = append(migrations, s.SchemaMigrations...)
	migrations = append(migrations, m)
	out.SchemaMigrations = migrations
	return out
}
