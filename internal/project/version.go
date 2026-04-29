package project

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/clin211/lin/internal/linctlerr"
)

// Migrator 把某一旧版 apiVersion 的 yaml 字节流升级为下一版本。
//
// 实现要点（与 docs/04-config-schema.md §4.9 一致）：
//   - From() / To() 是完整的 apiVersion 字符串（"linctl.dev/v1alpha1" 等）
//   - Migrate(data) 接收**整个 yaml 字节流**，返回升级后的字节流
//   - 实现内部应使用**宽松解析**（不开 KnownFields），以保留未知字段
//   - 升级后必须更新顶层 apiVersion 字段为 To()
//
// 多个 Migrator 通过 [Upgrade] 链式叠加，按 [migrators] 的顺序依次执行。
type Migrator interface {
	From() APIVersion
	To() APIVersion
	Migrate(data []byte) ([]byte, error)
}

// migrators 是已注册迁移器的有序列表（按依赖顺序）。
//
// 添加新版本时：在末尾追加新的 Migrator 实现即可。Loader 不会用到这个列表；
// 仅 [Upgrade] 函数读取它。
var migrators = []Migrator{
	&v1alpha1ToV1{},
}

// Upgrade 反复应用所有注册的 Migrator，直到 apiVersion 不再发生变化。
//
// 返回值：
//   - 升级后的 yaml 字节流（原字节流不被修改）
//   - 应用过的迁移记录（按时间顺序），调用方可写入 ProjectState.Status.SchemaMigrations
//   - error：任意一步失败时停止链路
//
// 边界：
//   - data 为空 → 返回 (nil, nil, ErrConfigInvalid)
//   - data 中没有 apiVersion 字段 → 返回原 data 与空 migrations（视作未知版本，不动）
func Upgrade(data []byte) ([]byte, []SchemaMigration, error) {
	if len(data) == 0 {
		return nil, nil, linctlerr.New(linctlerr.ErrConfigInvalid,
			"upgrade: empty input bytes",
			"Provide a non-empty linctl.yaml content")
	}

	current, err := peekAPIVersion(data)
	if err != nil {
		return nil, nil, err
	}
	if current == "" {
		// 没有 apiVersion 字段：保持原样，由后续 Loader 报「missing apiVersion」。
		return data, nil, nil
	}

	out := append([]byte(nil), data...)
	now := nowFunc()
	var applied []SchemaMigration

	// 反复扫描 migrator 列表，直到一轮无变化（防止环路）。
	const maxPasses = 32
	for pass := 0; pass < maxPasses; pass++ {
		next := pickMigrator(current)
		if next == nil {
			return out, applied, nil
		}
		newBytes, err := next.Migrate(out)
		if err != nil {
			return nil, nil, linctlerr.Wrapf(linctlerr.ErrConfigInvalid, err,
				"upgrade: migrate %s -> %s", next.From(), next.To())
		}
		applied = append(applied, SchemaMigration{
			From: next.From(),
			To:   next.To(),
			At:   now,
		})
		out = newBytes
		current = next.To()
	}
	return nil, nil, linctlerr.New(linctlerr.ErrInternal,
		"upgrade: exceeded maximum migration passes (cycle?)",
		"Inspect internal/project/version.go migrators list for cycles")
}

// pickMigrator 在 migrators 列表中查找以 from 为入口的下一个迁移器。
// 找不到返回 nil（终态）。
func pickMigrator(from APIVersion) Migrator {
	for _, m := range migrators {
		if m.From() == from {
			return m
		}
	}
	return nil
}

// peekAPIVersion 在不严格解析的前提下，仅提取顶层 apiVersion 字段的字符串值。
//
// 这样我们能在 [Upgrade] 中读出"当前是哪个版本"，而无需先 KnownFields 严格解码
// （否则 v1alpha1 的旧字段会让解码失败）。
func peekAPIVersion(data []byte) (APIVersion, error) {
	type apiOnly struct {
		APIVersion APIVersion `yaml:"apiVersion"`
	}
	var head apiOnly
	if err := yaml.Unmarshal(data, &head); err != nil {
		return "", linctlerr.Wrap(linctlerr.ErrConfigInvalid, err, "yaml peek apiVersion")
	}
	return head.APIVersion, nil
}

// ===== v1alpha1 → v1 迁移实现 =====

// v1alpha1ToV1 实现 SSOT §1.18 锁定的字段重命名：
//   - 顶层 apiVersion: linctl.dev/v1alpha1 → linctl.dev/v1
//   - spec.defaults.apiVersion → spec.defaults.protoVersion
//
// 实现策略：用 yaml.Node 树做最小手术，保留所有未知字段（前向兼容）。
type v1alpha1ToV1 struct{}

// From 返回此 Migrator 的入口 apiVersion。
func (m *v1alpha1ToV1) From() APIVersion { return APIVersionV1Alpha1 }

// To 返回此 Migrator 的目标 apiVersion。
func (m *v1alpha1ToV1) To() APIVersion { return APIVersionV1 }

// Migrate 执行 v1alpha1 → v1 的字段重命名。
func (m *v1alpha1ToV1) Migrate(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("migrate v1alpha1->v1: empty input")
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	root := documentRoot(&doc)
	if root == nil {
		return nil, errors.New("migrate v1alpha1->v1: document has no mapping root")
	}

	if err := setMappingValue(root, "apiVersion", APIVersionV1); err != nil {
		return nil, fmt.Errorf("set apiVersion: %w", err)
	}

	// 重命名 spec.defaults.apiVersion → spec.defaults.protoVersion（若存在）。
	if defaults := mappingChild(mappingChild(root, "spec"), "defaults"); defaults != nil {
		if err := renameMappingKey(defaults, "apiVersion", "protoVersion"); err != nil {
			return nil, fmt.Errorf("rename spec.defaults.apiVersion: %w", err)
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("yaml encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("yaml encoder close: %w", err)
	}
	return buf.Bytes(), nil
}

// ===== yaml.Node 辅助函数 =====

// documentRoot 返回文档节点下的 mapping root；如果不是 mapping 返回 nil。
func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return nil
		}
		doc = doc.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return nil
	}
	return doc
}

// mappingChild 从 mapping 节点中按 key 查子节点。未找到返回 nil。
//
// 注意：mapping 子节点是平铺的 [k1, v1, k2, v2, ...] 结构。
func mappingChild(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// setMappingValue 将 mapping[key] 的 scalar value 设置为 value（不改变其他字段）。
// 如果 key 不存在，则在末尾追加。
func setMappingValue(node *yaml.Node, key, value string) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return errors.New("not a mapping node")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			v := node.Content[i+1]
			v.Kind = yaml.ScalarNode
			v.Tag = "!!str"
			v.Value = value
			v.Style = 0
			return nil
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	node.Content = append(node.Content, keyNode, valNode)
	return nil
}

// renameMappingKey 把 mapping 中的 oldKey 重命名为 newKey（仅改 key，不改 value）。
//
// 如果 oldKey 不存在，no-op；如果 newKey 已存在，返回 error 以避免覆盖。
func renameMappingKey(node *yaml.Node, oldKey, newKey string) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return errors.New("not a mapping node")
	}
	if mappingChild(node, newKey) != nil {
		return fmt.Errorf("target key %q already exists", newKey)
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == oldKey {
			k.Value = newKey
			return nil
		}
	}
	return nil
}
