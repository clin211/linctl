package transform

import (
	"bytes"
	"context"
	"strings"
	"time"
)

// InsertLinctlHeader 在文件开头插入一段标记注释，说明该文件由 templatesync
// 自动同步产生、来自哪个上游路径、对应的 sync.yaml 规则在哪里。
//
// 默认模板（适用 Go / proto / 其他 // 注释类型）：
//
//	// 此文件由 linctl 模板系统从 {{ .Upstream.Name }}@{{ .UpstreamCommit }} 自动同步生成。
//	// 上游路径：{{ .UpstreamPath }}
//	// 同步规则：{{ .SyncManifestPath }}
//
// 可在 sync.yaml 自定义模板：
//
//	- kind: insertLinctlHeader
//	  template: |
//	    // Auto-synced from {{ .Upstream.Name }}; do not edit manually.
//
// 占位符：
//   - {{ .Upstream.Name }}     - manifest.upstream.name
//   - {{ .UpstreamPath }}      - 当前文件相对上游的相对路径
//   - {{ .UpstreamCommit }}    - 上次同步时的上游 commit（可选）
//   - {{ .SyncManifestPath }}  - sync.yaml 路径
//   - {{ .GeneratedAt }}       - 当前时间（RFC3339）
type InsertLinctlHeader struct {
	Template       string
	UpstreamCommit string // runtime 注入：当前同步使用的上游 commit
	SyncManifest   string // runtime 注入：sync.yaml 路径
}

// Kind implements Transform.
func (i *InsertLinctlHeader) Kind() string { return "insertLinctlHeader" }

const defaultLinctlHeader = `// 此文件由 linctl 模板系统从 {{ .Upstream.Name }}@{{ .UpstreamCommit }} 自动同步生成。
// 上游路径：{{ .UpstreamPath }}
// 同步规则：{{ .SyncManifestPath }}

`

// Apply implements Transform.
func (i *InsertLinctlHeader) Apply(_ context.Context, rc *RuntimeContext, content []byte) ([]byte, error) {
	tpl := i.Template
	if tpl == "" {
		tpl = defaultLinctlHeader
	}

	upstreamName := ""
	if rc.Manifest != nil {
		upstreamName = rc.Manifest.UpstreamName()
	}

	commit := i.UpstreamCommit
	if commit == "" {
		commit = "unknown"
	}
	syncManifest := i.SyncManifest
	if syncManifest == "" {
		syncManifest = "lin/internal/template/templates/web-gin/.sync.yaml"
	}

	header := strings.NewReplacer(
		"{{ .Upstream.Name }}", upstreamName,
		"{{.Upstream.Name}}", upstreamName,
		"{{ .UpstreamPath }}", rc.SrcPath,
		"{{.UpstreamPath}}", rc.SrcPath,
		"{{ .UpstreamCommit }}", commit,
		"{{.UpstreamCommit}}", commit,
		"{{ .SyncManifestPath }}", syncManifest,
		"{{.SyncManifestPath}}", syncManifest,
		"{{ .GeneratedAt }}", time.Now().UTC().Format(time.RFC3339),
		"{{.GeneratedAt}}", time.Now().UTC().Format(time.RFC3339),
	).Replace(tpl)

	// 处理已存在的 linctl header（幂等）：检测 content 头部是否含 marker，
	// 若含则替换之，避免重复堆叠。
	if existing := findExistingLinctlHeader(content); existing > 0 {
		return append([]byte(header), content[existing:]...), nil
	}

	// 处理 shebang：保留原 #!... 行在最前
	if bytes.HasPrefix(content, []byte("#!")) {
		idx := bytes.IndexByte(content, '\n')
		if idx > 0 {
			return append(append(content[:idx+1:idx+1], []byte(header)...), content[idx+1:]...), nil
		}
	}

	out := make([]byte, 0, len(header)+len(content))
	out = append(out, []byte(header)...)
	out = append(out, content...)
	return out, nil
}

// findExistingLinctlHeader 检测 content 开头是否已有 linctl header。
// 返回非零的 byte offset 表示"应丢弃前 N 字节"；返回 0 表示无现存 header。
//
// 简化算法：找包含 "linctl 模板系统从" 或 "Auto-synced from" 的连续注释块（前 8 行内）。
func findExistingLinctlHeader(content []byte) int {
	const scanLines = 8
	lines := bytes.SplitN(content, []byte("\n"), scanLines+1)

	hasMarker := false
	headerEnd := 0
	for i, ln := range lines {
		if i >= scanLines {
			break
		}
		t := bytes.TrimSpace(ln)
		if !bytes.HasPrefix(t, []byte("//")) && !bytes.HasPrefix(t, []byte("#")) {
			break
		}
		if bytes.Contains(ln, []byte("linctl 模板系统从")) ||
			bytes.Contains(ln, []byte("Auto-synced from")) {
			hasMarker = true
		}
		// +1 for newline; lines slice 不含分隔符
		headerEnd += len(ln) + 1
	}
	if !hasMarker {
		return 0
	}
	// 跳过紧随的空行
	rest := content[headerEnd:]
	for len(rest) > 0 && (rest[0] == '\n' || rest[0] == '\r') {
		headerEnd++
		rest = rest[1:]
	}
	return headerEnd
}

func insertLinctlHeaderFactory(rawCfg map[string]any) (Transform, error) {
	tpl, _ := asString(rawCfg, "template", false)
	return &InsertLinctlHeader{Template: tpl}, nil
}

func init() {
	DefaultRegistry.MustRegister("insertLinctlHeader", insertLinctlHeaderFactory)
}
