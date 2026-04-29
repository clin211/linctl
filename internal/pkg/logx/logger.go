// Package logx 是 lin v2 的 slog 封装。
//
// 设计来源：lin/docs/features/02-command-set.md §2.2「结构化日志字段（slog）」。
//
// 标准字段：cmd / phase / file / mutator / resource / app / duration_ms / exit_code / err / err_code。
package logx

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Format 决定日志输出格式。
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Options 控制 logger 行为。
type Options struct {
	Level  string // debug / info / warn / error
	Format Format // text / json
	Writer io.Writer
}

// New 创建一个标准 slog.Logger。Writer 默认 stderr。
func New(opts Options) *slog.Logger {
	w := opts.Writer
	if w == nil {
		w = os.Stderr
	}

	var lvl slog.Level
	switch strings.ToLower(opts.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	hOpts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if opts.Format == FormatJSON {
		h = slog.NewJSONHandler(w, hOpts)
	} else {
		h = slog.NewTextHandler(w, hOpts)
	}
	return slog.New(h)
}

// Phase 标准化 slog 的 phase 字段值（详见 02 §2.2）。
const (
	PhaseParse    = "parse"
	PhasePlan     = "plan"
	PhaseRender   = "render"
	PhaseInject   = "inject"
	PhaseVerify   = "verify"
	PhaseRollback = "rollback"
	PhaseDone     = "done"
)

// AttrCmd 返回标准 slog.Attr。
func AttrCmd(name string) slog.Attr {
	return slog.String("cmd", name)
}

// AttrPhase 返回标准 slog.Attr。
func AttrPhase(phase string) slog.Attr {
	return slog.String("phase", phase)
}
