package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/validate"
)

// Loader 负责把 `linctl.yaml` 字节流解析为 *Project：
//
//  1. **严格 yaml 解码**（KnownFields(true)）：未知字段直接报错
//  2. **应用默认值**（[ApplyDefaults]）
//  3. **强校验**（go-playground/validator/v10 + 自定义规则）
//
// 设计原则：
//   - 任何错误都包装为 *linctlerr.LinctlError，自带 Hint
//   - yaml 解码错误的 hint 会包含失败行号 / 字段名（来自 yaml.v3 的 TypeError）
//   - Validator 校验错误会被聚合并附上修复建议（参见 internal/validate.ToLinctlError）
//
// 多次调用 Loader 是 goroutine-safe 的，因为 *validate.Validator 是并发安全单例。
type Loader struct {
	v *validate.Validator
}

// NewLoader 构造一个使用全局单例 Validator 的 Loader。
//
// 在测试或需要替换校验规则的场景下，请使用 [NewLoaderWithValidator] 注入自定义实例。
func NewLoader() *Loader {
	return &Loader{v: validate.Default()}
}

// NewLoaderWithValidator 构造一个使用给定 Validator 的 Loader。v 为 nil 时回退到 [validate.Default]。
func NewLoaderWithValidator(v *validate.Validator) *Loader {
	if v == nil {
		v = validate.Default()
	}
	return &Loader{v: v}
}

// Load 是 [LoadFromFile] 的别名，便于与 docs/04-config-schema.md §4.7 命名一致。
func (l *Loader) Load(path string) (*Project, error) {
	return l.LoadFromFile(path)
}

// LoadFromFile 读取磁盘上的 `linctl.yaml`，并执行完整的解析-默认-校验流程。
//
// 边界处理：
//   - path == "" → ErrConfigInvalid + hint
//   - 文件不存在 / 无权限 → ErrEnvironment（不是 ErrConfigInvalid，因为属于环境问题）
//   - 内容为空 → ErrConfigInvalid + hint
func (l *Loader) LoadFromFile(path string) (*Project, error) {
	if path == "" {
		return nil, linctlerr.New(linctlerr.ErrConfigInvalid,
			"LoadFromFile: empty path",
			"Pass an absolute or workspace-relative path, e.g. './linctl.yaml'.")
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is the user-supplied config path
	if err != nil {
		// os.ReadFile 在文件不存在时返回 *PathError；统一映射为 environment。
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"LoadFromFile: read %s", path)
	}
	p, err := l.LoadFromBytes(data)
	if err != nil {
		// 富化 Hint：附加 path 信息，便于多文件/多环境调试。
		var lerr *linctlerr.LinctlError
		if errors.As(err, &lerr) {
			return nil, lerr.WithHint(fmt.Sprintf("source: %s", path))
		}
		return nil, err
	}
	return p, nil
}

// LoadFromBytes 把字节流解析为 *Project。
//
// 严格 KnownFields(true) 模式：yaml 中出现 struct 未声明的字段会立即报错，
// 避免因拼写错误（如 `componnets` 而非 `components`）静默丢失数据。
//
// 错误码：
//   - 解码失败：linctlerr.ErrConfigInvalid（带 yaml.TypeError 的行号信息）
//   - 校验失败：linctlerr.ErrConfigInvalid（带字段路径与修复建议）
func (l *Loader) LoadFromBytes(data []byte) (*Project, error) {
	if len(data) == 0 {
		return nil, linctlerr.New(linctlerr.ErrConfigInvalid,
			"LoadFromBytes: empty input",
			"linctl.yaml must contain at least apiVersion, kind, metadata and spec.")
	}

	p := &Project{}
	if err := decodeStrict(data, p); err != nil {
		return nil, err
	}

	ApplyDefaults(p)

	if err := l.v.Struct(p); err != nil {
		return nil, err
	}

	if err := crossFieldValidate(p); err != nil {
		return nil, err
	}

	return p, nil
}

// decodeStrict 用 yaml.v3 + KnownFields(true) 严格解码到 dst。
//
// 把 *yaml.TypeError 中的多行错误信息抽出为 hint（每行一个失败字段）。
func decodeStrict(data []byte, dst any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(dst); err != nil {
		return wrapYAMLError(err)
	}

	// 多文档检测：如果还能再读出一个 doc，说明用户在同一文件中放了多个文档。
	var extra yaml.Node
	if err := dec.Decode(&extra); err == nil {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"LoadFromBytes: multiple YAML documents in single file",
			"linctl.yaml must contain exactly one document; remove the extra '---' separator.")
	} else if !errors.Is(err, io.EOF) {
		return wrapYAMLError(err)
	}
	return nil
}

// wrapYAMLError 把 yaml.TypeError 或普通 error 包装为 LinctlError，并填充 hint。
func wrapYAMLError(err error) *linctlerr.LinctlError {
	if err == nil {
		return nil
	}
	out := linctlerr.Wrap(linctlerr.ErrConfigInvalid, err, "yaml decode")
	if out == nil {
		return nil
	}

	var terr *yaml.TypeError
	if errors.As(err, &terr) && len(terr.Errors) > 0 {
		out = out.WithHint(strings.Join(terr.Errors, "\n"))
		return out
	}
	out = out.WithHint("Run 'linctl lint' for a more detailed report.")
	return out
}

// crossFieldValidate 执行 struct tag 表达不了的跨字段约束。
//
// 目前覆盖（与 docs/04-config-schema.md §4.3.4 一致）：
//   - WebServer：framework=grpc → grpcPort 必填且 > 0；framework=gin → grpcPort 必须为 0
//   - Worker：variants=cron → cron.jobs 必须 ≥ 1
//   - Worker：variants=kafka → kafka.brokers 与 kafka.topics 必须 ≥ 1
//   - Worker：variants=customized → customized 必须 ≥ 1
//   - 全局：components.name 不重复
func crossFieldValidate(p *Project) error {
	if p == nil {
		return nil
	}

	seenNames := make(map[string]int, len(p.Spec.Components))

	for i := range p.Spec.Components {
		c := &p.Spec.Components[i]

		if prev, dup := seenNames[c.Name]; dup {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"duplicate component name %q (first at index %d, again at index %d)",
				c.Name, prev, i,
			).WithHint("Each component must have a unique name across the project.")
		}
		seenNames[c.Name] = i

		switch c.Kind {
		case "WebServer":
			if err := validateWebServer(c, i); err != nil {
				return err
			}
		case "Worker":
			if err := validateWorker(c, i); err != nil {
				return err
			}
		case "CLI":
			if err := validateCLI(c, i); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateWebServer(c *Component, idx int) error {
	switch c.Framework {
	case "grpc":
		if c.GRPCPort == 0 {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"spec.components[%d] (%q): framework=grpc requires grpcPort", idx, c.Name).
				WithHint("Add `grpcPort: <1024-65535>` to the component, e.g. grpcPort: 6666.")
		}
	case "", "gin":
		if c.GRPCPort != 0 {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"spec.components[%d] (%q): framework=gin must not set grpcPort", idx, c.Name).
				WithHint("Remove `grpcPort` or set framework=grpc.")
		}
	}
	return nil
}

func validateWorker(c *Component, idx int) error {
	if len(c.Variants) == 0 {
		return linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"spec.components[%d] (%q): worker requires at least one variant", idx, c.Name).
			WithHint("Set `variants: [cron]` or `variants: [kafka]` or `variants: [customized]`.")
	}

	hasCron, hasKafka, hasCustomized := false, false, false
	for _, v := range c.Variants {
		switch v {
		case "cron":
			hasCron = true
		case "kafka":
			hasKafka = true
		case "customized":
			hasCustomized = true
		}
	}

	if hasCron {
		if c.Cron == nil || len(c.Cron.Jobs) == 0 {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"spec.components[%d] (%q): variant=cron requires cron.jobs (>=1)", idx, c.Name).
				WithHint("Add `cron.jobs: [{name: someJob}]` to the component.")
		}
	}
	if hasKafka {
		if c.Kafka == nil || len(c.Kafka.Brokers) == 0 || len(c.Kafka.Topics) == 0 {
			return linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"spec.components[%d] (%q): variant=kafka requires kafka.brokers and kafka.topics (>=1 each)",
				idx, c.Name,
			).WithHint("Add `kafka.brokers: [host:port]` and `kafka.topics: [{name: someTopic}]`.")
		}
	}
	if hasCustomized && len(c.Customized) == 0 {
		return linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"spec.components[%d] (%q): variant=customized requires customized (>=1)", idx, c.Name).
			WithHint("Add `customized: [{name: someWatcher}]` to the component.")
	}
	return nil
}

func validateCLI(c *Component, idx int) error {
	if len(c.Commands) == 0 {
		return linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"spec.components[%d] (%q): CLI requires at least one command", idx, c.Name).
			WithHint("Add `commands: [{name: get}, {name: create}]` to the component.")
	}
	return nil
}
