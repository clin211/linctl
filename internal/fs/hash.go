package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// HashContent 计算字节切片的 SHA256 hex 摘要。
//
// 性能：一次 New + Write + Sum，hex.EncodeToString 内部预分配。
func HashContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ShortHash 返回 hash 的前 12 个字符（适合附在文件注释中减少视觉噪音）。
func ShortHash(b []byte) string {
	full := HashContent(b)
	if len(full) > 12 {
		return full[:12]
	}
	return full
}

// HashCommentMarker 是 linctl 在生成文件中插入的 hash 注释标记。
//
// 格式形如：// linctl: hash=<hex>，根据文件类型自动选择前缀。
const HashCommentMarker = "linctl: hash="

// hashCommentTemplates 按扩展名映射到注释格式。
//
// %s 占位符是完整 hash（hex）。
var hashCommentTemplates = map[string]string{
	".go":         "// " + HashCommentMarker + "%s",
	".proto":      "// " + HashCommentMarker + "%s",
	".java":       "// " + HashCommentMarker + "%s",
	".ts":         "// " + HashCommentMarker + "%s",
	".js":         "// " + HashCommentMarker + "%s",
	".c":          "// " + HashCommentMarker + "%s",
	".cpp":        "// " + HashCommentMarker + "%s",
	".h":          "// " + HashCommentMarker + "%s",
	".rs":         "// " + HashCommentMarker + "%s",
	".kt":         "// " + HashCommentMarker + "%s",
	".swift":      "// " + HashCommentMarker + "%s",
	".scala":      "// " + HashCommentMarker + "%s",
	".py":         "# " + HashCommentMarker + "%s",
	".rb":         "# " + HashCommentMarker + "%s",
	".sh":         "# " + HashCommentMarker + "%s",
	".bash":       "# " + HashCommentMarker + "%s",
	".yaml":       "# " + HashCommentMarker + "%s",
	".yml":        "# " + HashCommentMarker + "%s",
	".toml":       "# " + HashCommentMarker + "%s",
	".dockerfile": "# " + HashCommentMarker + "%s",
	".html":       "<!-- " + HashCommentMarker + "%s -->",
	".xml":        "<!-- " + HashCommentMarker + "%s -->",
	".md":         "<!-- " + HashCommentMarker + "%s -->",
	".css":        "/* " + HashCommentMarker + "%s */",
	".scss":       "/* " + HashCommentMarker + "%s */",
}

// hashCommentRegex 通用提取 hash 的正则。
var hashCommentRegex = regexp.MustCompile(`linctl: hash=([0-9a-fA-F]+)`)

// CommentTemplateFor 返回给定扩展名的 hash 注释格式串；不支持时返回 ""（调用方应略过）。
func CommentTemplateFor(ext string) string {
	return hashCommentTemplates[strings.ToLower(ext)]
}

// AppendHashComment 在文件内容末尾追加一行 hash 注释（按扩展名选注释格式）。
//
// 如果扩展名未在支持列表中（如 .png 二进制），返回原内容（不附加）。
func AppendHashComment(content []byte, hash string, ext string) []byte {
	tpl := CommentTemplateFor(ext)
	if tpl == "" {
		return content
	}
	line := strings.Replace(tpl, "%s", hash, 1)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		return append(append(content, '\n'), append([]byte(line), '\n')...)
	}
	return append(content, append([]byte(line), '\n')...)
}

// ExtractHashComment 从内容中提取 linctl hash 注释；返回 (hash, true) 或 ("", false)。
func ExtractHashComment(content []byte) (string, bool) {
	m := hashCommentRegex.FindSubmatch(content)
	if len(m) < 2 {
		return "", false
	}
	return string(m[1]), true
}

// StripHashComment 移除 hash 注释行（用于 hash 比对前的归一化）。
//
// 仅移除最后一行包含 marker 的内容（保留其他注释）。
func StripHashComment(content []byte) []byte {
	idx := hashCommentRegex.FindIndex(content)
	if idx == nil {
		return content
	}
	// 找到该行起始
	lineStart := idx[0]
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}
	// 找到该行末尾（含换行）
	lineEnd := idx[1]
	for lineEnd < len(content) && content[lineEnd] != '\n' {
		lineEnd++
	}
	if lineEnd < len(content) {
		lineEnd++ // include trailing newline
	}
	out := make([]byte, 0, len(content)-(lineEnd-lineStart))
	out = append(out, content[:lineStart]...)
	out = append(out, content[lineEnd:]...)
	return out
}
