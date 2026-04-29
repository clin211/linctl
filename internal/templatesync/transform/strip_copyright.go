package transform

import (
	"context"
	"strings"
)

// StripCopyrightHeader 去掉文件顶部的版权 / license 注释块。
//
// 算法：从文件第一个非空白行开始，连续的 `//` 注释行（或 `#` for shell/yaml）
// 中如果包含以下任一关键词（大小写不敏感），则视为版权头并整块剥离：
//
//	"Copyright" / "License" / "All rights reserved" / "SPDX-License-Identifier"
//
// 之后保留剥离后的剩余内容（含中间空行）。
//
// 不删除内容里间杂的版权块（仅扫文件头）；不影响其他注释（如 doc.go 的包注释）。
//
// sync.yaml 配置（无参数）：
//
//	- kind: stripCopyrightHeader
type StripCopyrightHeader struct{}

// Kind implements Transform.
func (s *StripCopyrightHeader) Kind() string { return "stripCopyrightHeader" }

var copyrightKeywords = []string{
	"copyright",
	"license",
	"all rights reserved",
	"spdx-license-identifier",
}

// Apply implements Transform.
func (s *StripCopyrightHeader) Apply(_ context.Context, rc *RuntimeContext, content []byte) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return content, nil
	}

	// 选择注释前缀：默认 //（Go/proto/c-style）；yaml/shell/dockerfile 用 #
	commentPrefix := "//"
	switch {
	case strings.HasSuffix(rc.SrcPath, ".yaml"),
		strings.HasSuffix(rc.SrcPath, ".yml"),
		strings.HasSuffix(rc.SrcPath, ".sh"),
		strings.HasSuffix(rc.SrcPath, ".bash"),
		strings.HasSuffix(rc.SrcPath, ".dockerfile"),
		strings.HasSuffix(rc.SrcPath, ".toml"):
		commentPrefix = "#"
	}

	// 1) 跳过开头的连续空行 / shebang
	start := 0
	for start < len(lines) {
		t := strings.TrimSpace(lines[start])
		if t == "" {
			start++
			continue
		}
		if commentPrefix == "#" && strings.HasPrefix(t, "#!") {
			start++
			continue
		}
		break
	}

	// 2) 检测 [start, end) 是否是连续注释块
	end := start
	hasCopyright := false
	for end < len(lines) {
		t := strings.TrimSpace(lines[end])
		if t == "" {
			// 注释块内允许空行（如 license 排版）；但只在 hasCopyright 已发现后接受
			if hasCopyright {
				end++
				continue
			}
			break
		}
		if !strings.HasPrefix(t, commentPrefix) {
			break
		}
		lower := strings.ToLower(t)
		for _, kw := range copyrightKeywords {
			if strings.Contains(lower, kw) {
				hasCopyright = true
				break
			}
		}
		end++
	}

	if !hasCopyright {
		return content, nil
	}

	// 3) 跳过紧随其后的连续空行（让产物干净）
	for end < len(lines) && strings.TrimSpace(lines[end]) == "" {
		end++
	}

	// 4) 拼回剩余内容
	keep := append([]string{}, lines[:start]...)
	keep = append(keep, lines[end:]...)
	return []byte(strings.Join(keep, "\n")), nil
}

func stripCopyrightFactory(_ map[string]any) (Transform, error) {
	return &StripCopyrightHeader{}, nil
}

func init() {
	DefaultRegistry.MustRegister("stripCopyrightHeader", stripCopyrightFactory)
}
