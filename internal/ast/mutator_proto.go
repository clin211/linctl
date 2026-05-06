package ast

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// ProtoPayload carries parameters for AddProtoImport.
//
// Insertion is purely structural: the target import is placed inside the existing
// import block (sorted-aware), or directly after the package/option header lines
// when no imports exist yet. No anchor comments are used.
type ProtoPayload struct {
	// File is the path to the .proto file to modify (overwritten by the caller).
	File string
	// Import is the import string to add (e.g. "post.proto").
	Import string
}

// AddProtoImport inserts an import statement into a .proto file.
// Idempotent: skips if the import already exists.
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

// detectImportLines returns 0-based line indices of import "..." lines.
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

// extractImportValues extracts the quoted import path from each import line.
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

// isSortedImports returns true if the import values are in ascending order.
func isSortedImports(vals []string) bool {
	return sort.StringsAreSorted(vals)
}

// insertAfterHeader places importLine right after the proto header
// (option / package lines), skipping any leading comments.
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

// insertSorted inserts importLine at the alphabetically correct position.
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

// insertAfterLast inserts importLine after the last import line.
func insertAfterLast(lines []string, importLineNos []int, importLine string) []string {
	last := importLineNos[len(importLineNos)-1]
	return insertAt(lines, last+1, importLine)
}

// insertAt inserts a line at position i (0-based).
func insertAt(lines []string, i int, line string) []string {
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:i]...)
	result = append(result, line)
	result = append(result, lines[i:]...)
	return result
}
