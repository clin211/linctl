package templatesync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/clin211/linctl/internal/linctlerr"
	"gopkg.in/yaml.v3"
)

// 当前 sync.yaml 支持的 apiVersion / kind 组合。
const (
	manifestAPIVersion = "linctl-internal/v1"
	manifestKind       = "TemplateUpstreamSync"
)

// Owner 描述一个文件的归属语义，决定 sync 时的策略（详见
// docs/META-template-upstream-sync-2026-04-28.md §3）。
type Owner string

const (
	// OwnerUpstream：完全跟随上游，sync 时直接覆盖（lin 不改）。
	OwnerUpstream Owner = "upstream"

	// OwnerShared：双方共维护，sync 时走 3-way merge（U2 阶段引入）。
	OwnerShared Owner = "shared"

	// OwnerLinSpecific：仅 lin 自有，sync 时跳过（不来自上游）。
	OwnerLinSpecific Owner = "linSpecific"
)

// Manifest 是 sync.yaml 解析后的对象。
type Manifest struct {
	APIVersion        string             `yaml:"apiVersion"`
	Kind              string             `yaml:"kind"`
	Metadata          ManifestMetadata   `yaml:"metadata"`
	Upstream          UpstreamSpec       `yaml:"upstream"`
	DefaultTransforms []TransformConfig  `yaml:"defaultTransforms"`
	Files             []FileMapping      `yaml:"files"`
	LinSpecific       []LinSpecificEntry `yaml:"linSpecific,omitempty"`
	Ignored           []IgnoredEntry     `yaml:"ignored,omitempty"`
	NewFilePolicy     NewFilePolicySpec  `yaml:"newFilePolicy"`

	// 私有字段：解析时记录 manifest 自身的绝对路径（用于解析相对 upstream rootPath）。
	manifestPath string
}

// ManifestMetadata 是 sync.yaml metadata 块。
type ManifestMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// UpstreamSpec 是上游项目描述。
type UpstreamSpec struct {
	Name             string   `yaml:"name"`
	RootPath         string   `yaml:"rootPath"`         // 相对 sync.yaml 的路径
	ModuleOld        string   `yaml:"moduleOld"`        // 上游 go module path
	BinaryNameOld    string   `yaml:"binaryNameOld"`    // 上游 binary 名
	ComponentNameOld string   `yaml:"componentNameOld"` // 上游 component 名（如 apiserver）
	ScanPaths        []string `yaml:"scanPaths,omitempty"` // 限定扫描的子目录列表（相对 RootPath）；空时扫全部
}

// TransformConfig 是 sync.yaml 中单条 transform 配置（任意键值对，由 transform Factory 解析）。
type TransformConfig struct {
	Kind    string         `yaml:"kind"`
	RawCfg  map[string]any `yaml:",inline"` // 其余字段作为该 transform 的参数
}

// UnmarshalYAML 自定义 YAML 解析：先把整个 mapping 解到 RawCfg，再提取 kind。
func (t *TransformConfig) UnmarshalYAML(node *yaml.Node) error {
	raw := make(map[string]any)
	if err := node.Decode(&raw); err != nil {
		return err
	}
	kindAny, ok := raw["kind"]
	if !ok {
		return fmt.Errorf("transform: missing required field 'kind'")
	}
	kindStr, ok := kindAny.(string)
	if !ok {
		return fmt.Errorf("transform: 'kind' must be a string, got %T", kindAny)
	}
	t.Kind = kindStr
	delete(raw, "kind")
	t.RawCfg = raw
	return nil
}

// FileMapping 是单个文件的 src → dst 映射。
type FileMapping struct {
	Src              string            `yaml:"src,omitempty"`               // 上游相对路径；splits 模式时为空
	Dst              string            `yaml:"dst,omitempty"`               // 镜像相对路径（仅 splits 之外的单文件场景）
	Owner            Owner             `yaml:"owner"`                       // upstream / shared / linSpecific
	ExtraTransforms  []TransformConfig `yaml:"extraTransforms,omitempty"`   // 在 default 之上叠加
	Splits           []FileSplit       `yaml:"splits,omitempty"`            // 拆分输出（同一 src → 多 dst）
}

// FileSplit 是 splits 模式下的单个产物。
type FileSplit struct {
	Dst             string            `yaml:"dst"`
	SectionMatch    string            `yaml:"sectionMatch,omitempty"` // 例如 "// SECTION: core"
	ExtraTransforms []TransformConfig `yaml:"extraTransforms,omitempty"`
}

// LinSpecificEntry 描述一个仅在 lin 仓库内存在的模板文件（不来自上游）。
type LinSpecificEntry struct {
	Dst    string `yaml:"dst"`
	Reason string `yaml:"reason"`
}

// IgnoredEntry 描述一个上游有但故意不模板化的文件。
type IgnoredEntry struct {
	Src    string `yaml:"src"`
	Reason string `yaml:"reason"`
}

// NewFilePolicySpec 控制"上游加新文件、sync.yaml 没声明"时的行为。
type NewFilePolicySpec struct {
	// Default：warn / error / autoAdd
	Default string `yaml:"default"`
	// AutoAdd 时新文件的默认 owner
	AutoAddOwner Owner `yaml:"autoAddOwner,omitempty"`
}

// LoadManifest 从给定路径加载并校验 sync.yaml。
func LoadManifest(path string) (*Manifest, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"resolve manifest path %s", path)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"read manifest %s", path)
	}
	m := &Manifest{}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrConfigInvalid, err,
			"parse manifest %s", path)
	}
	m.manifestPath = abs
	if err := m.applyDefaults(); err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

// applyDefaults 给 Manifest 填默认值（不影响 Validate 的语义）。
func (m *Manifest) applyDefaults() error {
	if m.NewFilePolicy.Default == "" {
		m.NewFilePolicy.Default = "warn"
	}
	if m.NewFilePolicy.AutoAddOwner == "" {
		m.NewFilePolicy.AutoAddOwner = OwnerUpstream
	}
	return nil
}

// Validate 检查 manifest 自身的合法性。
func (m *Manifest) Validate() error {
	if m.APIVersion != manifestAPIVersion {
		return linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"manifest: apiVersion=%q; expected %q",
			m.APIVersion, manifestAPIVersion)
	}
	if m.Kind != manifestKind {
		return linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"manifest: kind=%q; expected %q", m.Kind, manifestKind)
	}
	if m.Metadata.Name == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"manifest.metadata.name is required")
	}
	if m.Upstream.Name == "" || m.Upstream.RootPath == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"manifest.upstream.{name,rootPath} are required")
	}
	if m.Upstream.ModuleOld == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"manifest.upstream.moduleOld is required (used by rewriteImports)")
	}

	// 检查 NewFilePolicy.Default 合法
	switch m.NewFilePolicy.Default {
	case "warn", "error", "autoAdd":
	default:
		return linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"manifest.newFilePolicy.default=%q; allowed: warn/error/autoAdd",
			m.NewFilePolicy.Default)
	}

	// 检查 files 项：dst 唯一
	dstSeen := make(map[string]int, len(m.Files))
	for i, f := range m.Files {
		if f.Owner == "" {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"manifest.files[%d]: owner is required", i)
		}
		switch f.Owner {
		case OwnerUpstream, OwnerShared, OwnerLinSpecific:
		default:
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"manifest.files[%d]: owner=%q; allowed: upstream/shared/linSpecific",
				i, f.Owner)
		}

		// 单文件场景
		if len(f.Splits) == 0 {
			if f.Src == "" || f.Dst == "" {
				return linctlerr.Newf(linctlerr.ErrConfigInvalid,
					"manifest.files[%d]: src and dst are required (or use splits)", i)
			}
			if prev, ok := dstSeen[f.Dst]; ok {
				return linctlerr.Newf(linctlerr.ErrConfigInvalid,
					"manifest.files[%d].dst=%q duplicates files[%d]", i, f.Dst, prev)
			}
			dstSeen[f.Dst] = i
			continue
		}

		// splits 场景
		if f.Src == "" {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"manifest.files[%d]: src is required for splits", i)
		}
		for j, s := range f.Splits {
			if s.Dst == "" {
				return linctlerr.Newf(linctlerr.ErrConfigInvalid,
					"manifest.files[%d].splits[%d]: dst is required", i, j)
			}
			if prev, ok := dstSeen[s.Dst]; ok {
				return linctlerr.Newf(linctlerr.ErrConfigInvalid,
					"manifest.files[%d].splits[%d].dst=%q duplicates files[%d]",
					i, j, s.Dst, prev)
			}
			dstSeen[s.Dst] = i
		}
	}

	// 检查 linSpecific.dst 与 files.dst 不重复
	for i, e := range m.LinSpecific {
		if e.Dst == "" {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"manifest.linSpecific[%d].dst is required", i)
		}
		if prev, ok := dstSeen[e.Dst]; ok {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"manifest.linSpecific[%d].dst=%q duplicates files[%d]",
				i, e.Dst, prev)
		}
	}

	return nil
}

// AbsUpstreamRoot 返回 Upstream.RootPath 的绝对路径（基于 manifest 自身路径解析）。
func (m *Manifest) AbsUpstreamRoot() (string, error) {
	if m.manifestPath == "" {
		return "", linctlerr.New(linctlerr.ErrInternal,
			"manifest: manifestPath not recorded; load via LoadManifest")
	}
	manifestDir := filepath.Dir(m.manifestPath)
	abs := filepath.Clean(filepath.Join(manifestDir, m.Upstream.RootPath))
	if _, err := os.Stat(abs); err != nil {
		return "", linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"resolve upstream rootPath %s (relative to %s)",
			m.Upstream.RootPath, manifestDir)
	}
	return abs, nil
}

// AllDsts 返回 manifest 涉及的全部 dst 集合（含 files / splits / linSpecific），
// 字典序排序，便于稳定输出。
func (m *Manifest) AllDsts() []string {
	out := make([]string, 0, len(m.Files)+len(m.LinSpecific)+8)
	for _, f := range m.Files {
		if len(f.Splits) == 0 {
			out = append(out, f.Dst)
			continue
		}
		for _, s := range f.Splits {
			out = append(out, s.Dst)
		}
	}
	for _, e := range m.LinSpecific {
		out = append(out, e.Dst)
	}
	sort.Strings(out)
	return out
}

// FindBySrc 按上游 src 路径查找 mapping；找不到返回 nil。
//
// splits 场景下任一 dst 命中即返回。
func (m *Manifest) FindBySrc(src string) *FileMapping {
	for i := range m.Files {
		if m.Files[i].Src == src {
			return &m.Files[i]
		}
	}
	return nil
}

// IsIgnored 判断给定上游路径是否被显式 ignored。
func (m *Manifest) IsIgnored(src string) bool {
	for _, ig := range m.Ignored {
		if ig.Src == src {
			return true
		}
	}
	return false
}
