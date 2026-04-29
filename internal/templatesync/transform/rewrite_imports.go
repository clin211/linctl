package transform

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"github.com/clin211/linctl/internal/linctlerr"
	"golang.org/x/tools/go/ast/astutil"
)

// RewriteImports 把 Go 源码中的 oldPath import 改写为 newPath。
//
// 实现采用 Go 标准库 AST + golang.org/x/tools/astutil（与 Kubebuilder / goimports
// 的实现路线对齐），相比 sed/regex：
//   - 不会误改字符串字面量中包含 oldPath 的内容
//   - 不会误改注释中提到 oldPath 的内容
//   - 自动维护 import 分组与对齐
//
// 对非 .go 文件返回原内容（no-op），便于在 sync.yaml 的 defaultTransforms
// 中无差别启用。
//
// sync.yaml 配置：
//
//	- kind: rewriteImports
//	  from: github.com/clin211/miniblog-v4
//	  to:   '{{`{{ .Project.Module }}`}}'
type RewriteImports struct {
	From string
	To   string
}

// Kind implements Transform.
func (r *RewriteImports) Kind() string { return "rewriteImports" }

// Apply implements Transform.
func (r *RewriteImports) Apply(_ context.Context, rc *RuntimeContext, content []byte) ([]byte, error) {
	if !strings.HasSuffix(rc.SrcPath, ".go") {
		return content, nil
	}

	from := r.From
	to := r.To

	// 允许 sync.yaml 中用 `{{ .Upstream.ModuleOld }}` 引用 manifest 字段
	if rc.Manifest != nil {
		from = expandManifestRefs(from, rc.Manifest)
		to = expandManifestRefs(to, rc.Manifest)
	}

	if from == "" {
		return nil, linctlerr.New(linctlerr.ErrConfigInvalid,
			"rewriteImports.from is empty after manifest expansion")
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rc.SrcPath, content, parser.ParseComments)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err,
			"rewriteImports: parse %s", rc.SrcPath)
	}

	changed := rewriteAllImports(fset, f, from, to)
	if !changed {
		return content, nil
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, f); err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err,
			"rewriteImports: format %s", rc.SrcPath)
	}
	return buf.Bytes(), nil
}

// rewriteAllImports 遍历 file 中的所有 import，对前缀匹配的 path 做替换。
// 返回是否发生过改动。
//
// 例如 from=github.com/clin211/miniblog-v4：
//   - github.com/clin211/miniblog-v4 → newPath
//   - github.com/clin211/miniblog-v4/internal/pkg → newPath/internal/pkg
//   - github.com/clin211/miniblog-v4/pkg/log → newPath/pkg/log
//
// 注意：astutil.RewriteImport 内部会修改 file.Imports，但我们边遍历边改可能
// 导致漏改后续条目；所以先收集所有要改的 (oldVal, newVal) 对，再统一应用。
func rewriteAllImports(fset *token.FileSet, file *ast.File, from, to string) bool {
	type pair struct {
		oldVal, newVal string
	}
	var todo []pair
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		raw := imp.Path.Value
		if len(raw) < 2 {
			continue
		}
		val := raw[1 : len(raw)-1] // strip surrounding quotes
		if val == from {
			todo = append(todo, pair{val, to})
			continue
		}
		if strings.HasPrefix(val, from+"/") {
			tail := val[len(from):]
			todo = append(todo, pair{val, to + tail})
		}
	}
	for _, p := range todo {
		astutil.RewriteImport(fset, file, p.oldVal, p.newVal)
	}
	return len(todo) > 0
}

// expandManifestRefs 替换形如 `{{ .Upstream.ModuleOld }}` 的占位为 manifest 真实值。
//
// 仅支持有限几个固定占位，避免引入完整 text/template 依赖：
//   - {{ .Upstream.ModuleOld }}
//   - {{ .Upstream.BinaryNameOld }}
//   - {{ .Upstream.ComponentNameOld }}
//   - {{ .Upstream.Name }}
//
// 注意：sync.yaml 中给 lin 模板用的占位（如反引号包裹的 `{{ .Project.Module }}`）
// 会被 YAML 反转义为字面量，本函数保留原样不动。
func expandManifestRefs(s string, m ManifestAccessor) string {
	if m == nil || s == "" {
		return s
	}
	r := strings.NewReplacer(
		"{{ .Upstream.ModuleOld }}", m.UpstreamModuleOld(),
		"{{.Upstream.ModuleOld}}", m.UpstreamModuleOld(),
		"{{ .Upstream.BinaryNameOld }}", m.UpstreamBinaryNameOld(),
		"{{.Upstream.BinaryNameOld}}", m.UpstreamBinaryNameOld(),
		"{{ .Upstream.ComponentNameOld }}", m.UpstreamComponentNameOld(),
		"{{.Upstream.ComponentNameOld}}", m.UpstreamComponentNameOld(),
		"{{ .Upstream.Name }}", m.UpstreamName(),
		"{{.Upstream.Name}}", m.UpstreamName(),
	)
	return r.Replace(s)
}

// rewriteImportsFactory 是从 sync.yaml 配置块构造 RewriteImports 的 Factory。
func rewriteImportsFactory(rawCfg map[string]any) (Transform, error) {
	from, err := asString(rawCfg, "from", true)
	if err != nil {
		return nil, err
	}
	to, err := asString(rawCfg, "to", true)
	if err != nil {
		return nil, err
	}
	return &RewriteImports{From: from, To: to}, nil
}

func init() {
	DefaultRegistry.MustRegister("rewriteImports", rewriteImportsFactory)
}
