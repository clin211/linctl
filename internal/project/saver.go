package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/clin211/lin/internal/linctlerr"
)

// projectFileHeader 是写入 PROJECT 文件顶部的禁止人工修改头注释。
//
// 与 docs/04-config-schema.md §4.8 的版本一致。每次写入都会重新拼接，
// 不影响 yaml 内容（注释会在 yaml 解码时被忽略）。
const projectFileHeader = `# DO NOT EDIT MANUALLY.
# This file (PROJECT) is maintained by linctl. To modify project configuration,
# update linctl.yaml and run 'linctl plan' / 'linctl apply'.
#
# Linctl docs: https://github.com/clin211/lin
`

// nowFunc 返回当前时间字符串（RFC3339）。它是一个**变量**而非函数，
// 便于测试时打桩为固定时间戳——避免黄金文件因时间差异而 flaky。
//
// 默认实现是 time.Now().UTC().Format(time.RFC3339)。
var nowFunc = func() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// SaveState 把运行时状态写入 PROJECT 文件（**不是 linctl.yaml**）。
//
// 调用流程：
//  1. 取 p 的 apiVersion / kind / metadata.name 作为身份镜像
//  2. 把 status 与 cliVersion / generatedAt 写入 status 段
//  3. 在 yaml 编码结果前拼接禁止人工修改头注释
//  4. 原子写入（先写 .tmp 再 rename）以避免半文件
//
// 边界处理：
//   - p == nil → 返回 ErrInternal（这是编程错误，不应发生在已 Validate 的项目上）
//   - path 父目录不存在 → 自动 MkdirAll（与 osbuilder 行为对齐）
//   - cliVersion 为 "" → 不覆盖 status 中已有的 CLIVersion
func SaveState(path string, p *Project, status Status, cliVersion string) error {
	if p == nil {
		return linctlerr.New(linctlerr.ErrInternal,
			"SaveState: project is nil",
			"This is a programming error; ensure the project is loaded and validated before saving.")
	}
	if path == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"SaveState: empty path",
			"Pass an absolute or workspace-relative path to the PROJECT file.")
	}

	if status.GeneratedAt == "" {
		status.GeneratedAt = nowFunc()
	}
	if cliVersion != "" {
		status.CLIVersion = cliVersion
	}

	snapshot := ProjectState{
		APIVersion: p.APIVersion,
		Kind:       p.Kind,
		Metadata:   Metadata{Name: p.Metadata.Name},
		Status:     status,
	}

	body, err := marshalState(&snapshot)
	if err != nil {
		return err
	}

	content := make([]byte, 0, len(projectFileHeader)+1+len(body))
	content = append(content, projectFileHeader...)
	content = append(content, '\n')
	content = append(content, body...)

	if err := writeFileAtomic(path, content, 0o644); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"SaveState: write %s", path)
	}
	return nil
}

// LoadState 读取并解析 PROJECT 文件。
//
// 与 [Load] 不同：
//   - 不强制 KnownFields（允许更早版本写入但本版本未识别的 status 字段）
//   - 不调用 ApplyDefaults 或强校验
//   - 仅把内容解析为 ProjectState，不做语义检查
//
// 调用方拿到 *ProjectState 后通常做：
//   - 与 linctl.yaml 的 metadata.name 比较（确保未被人手挪动 PROJECT 文件）
//   - 把 status.SchemaMigrations 续写新一次 migrate 记录
func LoadState(path string) (*ProjectState, error) {
	if path == "" {
		return nil, linctlerr.New(linctlerr.ErrConfigInvalid,
			"LoadState: empty path",
			"Pass an absolute or workspace-relative path to the PROJECT file.")
	}
	data, err := os.ReadFile(path) //nolint:gosec // path supplied by caller
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"LoadState: read %s", path)
	}
	return decodeState(data)
}

// decodeState 把 PROJECT 文件字节流解码为 ProjectState。
//
// 注意此处**不开 KnownFields**：PROJECT 文件由工具维护，向前兼容比严格性更重要。
func decodeState(data []byte) (*ProjectState, error) {
	state := &ProjectState{}
	if err := yaml.Unmarshal(data, state); err != nil {
		return nil, linctlerr.Wrap(linctlerr.ErrConfigInvalid, err, "PROJECT yaml decode")
	}
	return state, nil
}

// marshalState 把 ProjectState 序列化为 yaml 字节流。indent 与 osbuilder 习惯一致（2 空格）。
func marshalState(state *ProjectState) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(state); err != nil {
		return nil, linctlerr.Wrap(linctlerr.ErrInternal, err, "marshalState: encode")
	}
	if err := enc.Close(); err != nil {
		return nil, linctlerr.Wrap(linctlerr.ErrInternal, err, "marshalState: close")
	}
	return buf.Bytes(), nil
}

// writeFileAtomic 原子写入：先写临时文件，再 rename 覆盖目标路径。
// 中途崩溃不会留下半内容文件。
//
// 父目录会被自动创建（与 osbuilder 行为对齐）。
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	tmp, err := os.CreateTemp(dir, ".linctl-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}
