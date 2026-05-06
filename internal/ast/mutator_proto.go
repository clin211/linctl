package ast

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// ProtoPayload 是 AddProtoImport 的入参。
//
// 插入完全是结构性的：目标 import 会被放入已有 import 块（按字典序感知插入），
// 或当尚无任何 import 时直接置于 package/option 等头部声明之后。
// 不使用任何锚点注释。
type ProtoPayload struct {
	// File 是要修改的 .proto 文件路径（由调用方覆写）。
	File string
	// Import 是要新增的 import 字符串（如 "post.proto"）。
	Import string
}

// AddProtoImport 向 .proto 文件中插入一条 import 语句。
// 注入幂等：若 import 已存在则跳过。
func AddProtoImport(file string, p ProtoPayload) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return errs.Wrap(errs.CodeASTApplyError,
			fmt.Sprintf("ast: proto read %s", file), err)
	}

	lines := strings.Split(string(data), "\n")

	importLine := fmt.Sprintf(`import "%s";`, p.Import)
	for _, l := range lines {
		if strings.TrimSpace(l) == importLine {
			fmt.Printf("⊝ skipped (already exists): %s in %s\n", importLine, file)
			return nil
		}
	}

	importLineNos := detectImportLines(lines)

	var result []string
	if len(importLineNos) == 0 {
		result = insertAfterHeader(lines, importLine)
	} else {
		existing := extractImportValues(lines, importLineNos)
		if isSortedImports(existing) {
			result = insertSorted(lines, importLineNos, importLine, p.Import)
		} else {
			result = insertAfterLast(lines, importLineNos, importLine)
		}
	}

	out := strings.Join(result, "\n")
	if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
		return errs.Wrap(errs.CodeASTApplyError,
			fmt.Sprintf("ast: proto write %s", file), err)
	}
	return nil
}

// detectImportLines 返回所有 import "..." 行的 0-based 行号。
func detectImportLines(lines []string) []int {
	var idx []int
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, `import "`) && strings.HasSuffix(trimmed, `";`) {
			idx = append(idx, i)
		}
	}
	return idx
}

// extractImportValues 从每行 import 语句中抽取被引号包裹的 import 路径。
func extractImportValues(lines []string, lineNos []int) []string {
	var vals []string
	for _, i := range lineNos {
		l := strings.TrimSpace(lines[i])
		l = strings.TrimPrefix(l, `import "`)
		l = strings.TrimSuffix(l, `";`)
		vals = append(vals, l)
	}
	return vals
}

// isSortedImports 判断 import 列表是否已按升序排列。
func isSortedImports(vals []string) bool {
	return sort.StringsAreSorted(vals)
}

// insertAfterHeader 将 importLine 插入到 proto 头部（option / package 行）之后，
// 并跳过紧贴头部的注释。
func insertAfterHeader(lines []string, importLine string) []string {
	insertAt := 0
	for i, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "syntax"),
			strings.HasPrefix(t, "package "),
			strings.HasPrefix(t, "option "):
			insertAt = i + 1
		}
	}
	if insertAt == 0 {
		return append([]string{importLine, ""}, lines...)
	}
	result := make([]string, 0, len(lines)+2)
	result = append(result, lines[:insertAt]...)
	result = append(result, "")
	result = append(result, importLine)
	result = append(result, lines[insertAt:]...)
	return result
}

// insertSorted 将 importLine 插入到字典序应在的位置。
func insertSorted(lines []string, importLineNos []int, importLine, importVal string) []string {
	insertIdx := importLineNos[len(importLineNos)-1] + 1
	for _, lineNo := range importLineNos {
		l := strings.TrimSpace(lines[lineNo])
		l = strings.TrimPrefix(l, `import "`)
		l = strings.TrimSuffix(l, `";`)
		if importVal < l {
			insertIdx = lineNo
			break
		}
	}
	return insertAt(lines, insertIdx, importLine)
}

// insertAfterLast 将 importLine 插入到最后一行 import 之后。
func insertAfterLast(lines []string, importLineNos []int, importLine string) []string {
	last := importLineNos[len(importLineNos)-1]
	return insertAt(lines, last+1, importLine)
}

// insertAt 在指定的 0-based 位置 i 插入一行。
func insertAt(lines []string, i int, line string) []string {
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:i]...)
	result = append(result, line)
	result = append(result, lines[i:]...)
	return result
}
