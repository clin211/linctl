// Package version 集中管理 linctl 的版本元数据。
//
// 这些变量由 -ldflags 注入：
//
//	go build -ldflags "-X github.com/clin211/linctl/internal/version.Version=v1.0.0"
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// 这些变量由构建系统在 -ldflags 注入；不要在代码中修改。
var (
	// Version 是语义化版本号，例如 "v1.0.0"。dev 构建为 "dev"。
	Version = "dev"

	// Commit 是 git short commit hash。
	Commit = "unknown"

	// BuildDate 是构建时间（RFC3339 格式）。
	BuildDate = "unknown"
)

// Info 是结构化版本信息，用于 `linctl version --output json/yaml`。
type Info struct {
	Version    string `json:"version" yaml:"version"`
	Commit     string `json:"commit" yaml:"commit"`
	BuildDate  string `json:"buildDate" yaml:"buildDate"`
	GoVersion  string `json:"goVersion" yaml:"goVersion"`
	OS         string `json:"os" yaml:"os"`
	Arch       string `json:"arch" yaml:"arch"`
	Modified   bool   `json:"modified,omitempty" yaml:"modified,omitempty"`
	BuildFlags string `json:"buildFlags,omitempty" yaml:"buildFlags,omitempty"`
}

// Get 返回当前 linctl 二进制的版本信息。
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	// 从 debug.BuildInfo 补充 commit / modified（dev 构建场景）
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if info.Commit == "unknown" || info.Commit == "" {
					info.Commit = shortHash(s.Value)
				}
			case "vcs.modified":
				info.Modified = s.Value == "true"
			case "vcs.time":
				if info.BuildDate == "unknown" || info.BuildDate == "" {
					info.BuildDate = s.Value
				}
			case "-tags":
				info.BuildFlags = s.Value
			}
		}
	}
	return info
}

// String 返回人类可读的单行版本字符串。
func (i Info) String() string {
	mod := ""
	if i.Modified {
		mod = " (modified)"
	}
	return fmt.Sprintf("linctl %s (%s%s) built %s with %s on %s/%s",
		i.Version, i.Commit, mod, i.BuildDate, i.GoVersion, i.OS, i.Arch)
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
