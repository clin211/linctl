package templatesync

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clin211/linctl/internal/linctlerr"
)

// AppendFileEntry 把一个新的 file 映射追加到 sync.yaml 的 files: 块末尾。
//
// 设计原则：
//   - 保留原 sync.yaml 的注释 / 缩进 / 顺序（不全文 yaml marshal 重写）
//   - 只追加，不修改已存在条目
//   - 追加位置：files: 数组的最后一个元素之后；如果 files: 缺失则创建
//
// 参数：
//   - manifestPath：sync.yaml 绝对路径
//   - entry：要追加的条目（src + dst + owner 必填）
//
// 限制：
//   - 暂不支持 splits / extraTransforms 字段（U3 引入；U2 仅 add 简单条目）
//   - sync.yaml 必须使用 2 空格缩进（与项目其他 yaml 文件一致）
func AppendFileEntry(manifestPath string, entry FileMapping) error {
	if entry.Src == "" || entry.Dst == "" {
		return linctlerr.New(linctlerr.ErrConfigInvalid,
			"AppendFileEntry: src and dst are required")
	}
	if entry.Owner == "" {
		entry.Owner = OwnerUpstream
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"AppendFileEntry: read %s", manifestPath)
	}

	new, err := injectFileEntry(data, entry)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"AppendFileEntry: mkdir %s", filepath.Dir(manifestPath))
	}
	tmp := manifestPath + ".tmp"
	if err := os.WriteFile(tmp, new, 0o644); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"AppendFileEntry: write tmp %s", tmp)
	}
	if err := os.Rename(tmp, manifestPath); err != nil {
		_ = os.Remove(tmp)
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"AppendFileEntry: rename %s -> %s", tmp, manifestPath)
	}
	return nil
}

// injectFileEntry 是 AppendFileEntry 的纯字节实现（便于单测）。
func injectFileEntry(content []byte, entry FileMapping) ([]byte, error) {
	// 渲染要插入的 YAML 片段（4-line block，2-space indent）。
	block := fmt.Sprintf("  - src: %s\n    dst: %s\n    owner: %s\n",
		yamlScalar(entry.Src),
		yamlScalar(entry.Dst),
		entry.Owner,
	)

	lines := bytes.Split(content, []byte("\n"))

	filesIdx := findTopLevelKey(lines, "files:")
	if filesIdx < 0 {
		// files: 不存在 → 在文件末尾追加完整 files: 块
		out := make([]byte, 0, len(content)+len(block)+16)
		out = append(out, content...)
		if !bytes.HasSuffix(content, []byte("\n")) {
			out = append(out, '\n')
		}
		out = append(out, []byte("\nfiles:\n")...)
		out = append(out, []byte(block)...)
		return out, nil
	}

	// 找到 files: 块的结束（下一个顶级键 / 文件末尾）
	endIdx := findNextTopLevelKey(lines, filesIdx+1)
	if endIdx < 0 {
		endIdx = len(lines)
	}

	// 检查 files: 是否是 "[]" 形式（空数组）
	headLine := strings.TrimSpace(string(lines[filesIdx]))
	if headLine == "files: []" {
		// 把 files: [] 改为 files:\n + block
		lines[filesIdx] = []byte("files:")
		// 在 filesIdx+1 位置插入 block（去掉 block 末尾的 \n 让 split 正常）
		blockLines := bytes.Split([]byte(strings.TrimRight(block, "\n")), []byte("\n"))
		lines = insertLines(lines, filesIdx+1, blockLines)
		return bytes.Join(lines, []byte("\n")), nil
	}

	// 正常路径：在 files: 块的最后一个有效条目后插入 block。
	// 反向找最后一行属于本块的内容（缩进非顶级、非空、非"归属下个 key 的注释块"）。
	insertAt := endIdx
	for insertAt > filesIdx+1 {
		ln := lines[insertAt-1]
		t := strings.TrimSpace(string(ln))
		// 跳过空行
		if t == "" {
			insertAt--
			continue
		}
		// 跳过紧贴下一 top-level key 的注释块（# ... 顶头无缩进） →
		// 视为属于下个 section 的标题注释，应保持在新条目之后。
		if len(ln) > 0 && ln[0] == '#' {
			insertAt--
			continue
		}
		// 找到了真正的 files: 末尾条目
		break
	}

	blockLines := bytes.Split([]byte(strings.TrimRight(block, "\n")), []byte("\n"))
	lines = insertLines(lines, insertAt, blockLines)
	return bytes.Join(lines, []byte("\n")), nil
}

// findTopLevelKey 在 lines 中找到第一个等于给定顶级 key 的行；-1 表示找不到。
func findTopLevelKey(lines [][]byte, key string) int {
	for i, ln := range lines {
		t := strings.TrimSpace(string(ln))
		if strings.HasPrefix(string(ln), key) ||
			strings.HasPrefix(t, key) && !strings.HasPrefix(string(ln), " ") && !strings.HasPrefix(string(ln), "\t") {
			// 顶级 key（即顶头无缩进）
			if !strings.HasPrefix(string(ln), " ") && !strings.HasPrefix(string(ln), "\t") {
				return i
			}
		}
	}
	return -1
}

// findNextTopLevelKey 从 startIdx 开始找下一个顶级 key 行；-1 表示没有（已到文件末尾）。
func findNextTopLevelKey(lines [][]byte, startIdx int) int {
	for i := startIdx; i < len(lines); i++ {
		ln := lines[i]
		if len(ln) == 0 {
			continue
		}
		// 跳过纯注释 / 空白行
		t := strings.TrimSpace(string(ln))
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		// 顶级 key：首字符非空白
		if ln[0] != ' ' && ln[0] != '\t' && ln[0] != '-' {
			return i
		}
	}
	return -1
}

// insertLines 在 base[at:] 之前插入 inserted 切片（不含 trailing newline）。
func insertLines(base [][]byte, at int, inserted [][]byte) [][]byte {
	out := make([][]byte, 0, len(base)+len(inserted))
	out = append(out, base[:at]...)
	out = append(out, inserted...)
	out = append(out, base[at:]...)
	return out
}

// yamlScalar 给字符串加引号（如果含 yaml 特殊字符）。
//
// MVP：保守地永远不加引号（路径都是 ASCII 安全字符）。如有 :/{}[]# 等再加。
func yamlScalar(s string) string {
	if strings.ContainsAny(s, ":#{}[]&*!|>'\"%@`") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
