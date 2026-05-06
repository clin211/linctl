// Package check 实现 lin v2 的 lint 与 doctor 检查逻辑。
//
// 设计来源：lin/docs/features/02-command-set.md §5（lint）§6（doctor）。
package check

import (
	"encoding/json"
	"fmt"
	"io"
)

// Report 持有 Doctor 或 Lint 的输出。
type Report struct {
	Items   []Item  `json:"items"`
	Summary Summary `json:"summary"`
}

// Item 表示单条检查结果。
type Item struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Status   string `json:"status"` // "ok" | "warning" | "error" | "info"
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}

// Summary 汇总各项检查结果数量。
type Summary struct {
	OK       int `json:"ok"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
	Info     int `json:"info"`
}

// addItem 将一条检查结果追加到 report，并自增对应的统计计数。
func (r *Report) addItem(item Item) {
	r.Items = append(r.Items, item)
	switch item.Status {
	case "ok":
		r.Summary.OK++
	case "warning":
		r.Summary.Warnings++
	case "error":
		r.Summary.Errors++
	case "info":
		r.Summary.Info++
	}
}

// PrintReport 将 report 以指定格式（"text" 或 "json"）写入 w。
func PrintReport(report *Report, format string, w io.Writer) error {
	if format == "json" {
		return json.NewEncoder(w).Encode(report)
	}

	for _, item := range report.Items {
		switch item.Status {
		case "ok":
			fmt.Fprintf(w, "✔ %-20s %s\n", item.Name, item.Message)
		case "warning":
			msg := item.Message
			if item.Hint != "" {
				msg += "; " + item.Hint
			}
			fmt.Fprintf(w, "⚠ %-20s %s\n", item.Name, msg)
		case "error":
			msg := item.Message
			if item.Hint != "" {
				msg += "; " + item.Hint
			}
			fmt.Fprintf(w, "✘ %-20s %s\n", item.Name, msg)
		case "info":
			msg := item.Message
			if item.Hint != "" {
				msg += "; " + item.Hint
			}
			fmt.Fprintf(w, "ℹ %-20s %s\n", item.Name, msg)
		}
	}

	fmt.Fprintf(w, "\n%d ok / %d warning / %d error\n",
		report.Summary.OK, report.Summary.Warnings, report.Summary.Errors)

	return nil
}
