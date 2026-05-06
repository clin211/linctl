package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"

	linas "github.com/clin211/linctl/internal/ast"
	"github.com/clin211/linctl/internal/pkg/errs"
)

// AddOptions 控制 AddResource 的行为。
type AddOptions struct {
	With        []string // [conversion, validation, proto, errno]
	Without     []string // 与 With 互斥
	Ops         []string // [create, update, delete, get, list]
	Version     string   // 如 "v1"
	Plural      string   // 自定义复数形式
	NoInject    bool     // 跳过 AST 注入
	SkipImports bool     // 跳过 import 语句
}

// AddResource 是 linctl add 的核心。
//
// 设计来源：02 §4.5 行为流程，03 §1 资源分层，05 §5 注入顺序。
func AddResource(ctx *Context, name string, opts AddOptions) error {
	// 1. 校验资源名（必须为 PascalCase）
	if err := validateResourceName(name); err != nil {
		return err
	}

	// 2. 校验 With/Without 互斥
	if len(opts.With) > 0 && len(opts.Without) > 0 {
		return errs.New(errs.CodeFlagConflict,
			"scaffold: --with and --without are mutually exclusive").
			WithHint("use either --with or --without, not both")
	}

	// 3. 将 resource 写入 context
	ctx.Resource = name
	ctx.Features = computeWith(opts)

	// 4. 构建 Plan
	plan, err := BuildPlan(ctx, PlanKindResource)
	if err != nil {
		return err
	}

	// 5. 打印摘要
	fmt.Printf("✔ resource: %s\n", name)
	fmt.Printf("   + %d files to create\n", len(plan.Creates))
	if !opts.NoInject {
		fmt.Printf("   ✏  %d files to update via AST\n", len(plan.Injects))
	}

	// 6. DryRun：仅打印计划
	if ctx.DryRun {
		for _, spec := range plan.Creates {
			fmt.Printf("    + %s\n", spec.DestPath)
		}
		for _, spec := range plan.Injects {
			fmt.Printf("    ✏  %s (%s)\n", spec.File, spec.Mutator)
		}
		return nil
	}

	// 7. 执行（失败时回滚）
	var created []string

	// 7a. 渲染（创建新文件）
	vars := newTemplateVars(ctx)
	for _, spec := range plan.Creates {
		dstPath := filepath.Join(ctx.RootDir, spec.DestPath)
		if _, err := os.Stat(dstPath); err == nil && !ctx.Force {
			fmt.Printf("   ⊝ skip (exists): %s\n", spec.DestPath)
			continue
		}
		if err := renderOne(ctx, spec, vars); err != nil {
			// 回滚：删除已创建的文件
			cleanupCreated(created)
			return fmt.Errorf("create %s: %w", spec.DestPath, err)
		}
		created = append(created, dstPath)
		fmt.Printf("   + %s\n", spec.DestPath)
	}

	// 7b. AST 注入
	if !opts.NoInject && len(plan.Injects) > 0 {
		injector := linas.NewInjector(ctx.RootDir)
		injectPlan := linas.InjectPlan{}
		for _, spec := range plan.Injects {
			injectPlan.Specs = append(injectPlan.Specs, linas.InjectSpec{
				File:    spec.File,
				Mutator: linas.MutatorKind(spec.Mutator),
				Payload: spec.Payload,
			})
		}
		if err := injector.Inject(injectPlan); err != nil {
			// 回滚：删除已创建的文件
			cleanupCreated(created)
			return fmt.Errorf("ast inject: %w", err)
		}
		for _, spec := range plan.Injects {
			fmt.Printf("   ✏  %s\n", spec.File)
		}
	}

	// 8. 提示后续步骤
	fmt.Printf("📦 Next steps:\n")
	if hasProto(ctx) {
		fmt.Printf("   make protoc\n")
	}
	fmt.Printf("   go mod tidy\n")
	fmt.Printf("   go build ./...\n")

	return nil
}

// buildResourcePlan 构建 PlanKindResource 对应的 Plan。
func buildResourcePlan(ctx *Context) (*Plan, error) {
	res := ctx.Resource
	lower := strings.ToLower(res)
	// strcase.ToLowerCamel 返回 lower-camel，如 "PostItem" → "postItem"
	// 对于 "Post" 这种简单单词直接返回 "post"
	lowerCamel := strcase.ToLowerCamel(res)
	appName := ctx.AppName
	ver := "v1"

	plan := &Plan{Kind: PlanKindResource}

	// --- Creates ---

	// 1. handler/post.go
	plan.Creates = append(plan.Creates, FileSpec{
		TemplatePath: "resource/handler.go.tpl",
		DestPath:     filepath.Join("internal", appName, "handler", lower+".go"),
		Permissions:  0o644,
	})

	// 2. biz/v1/post/post.go（接口 + 结构体 + New 函数）
	plan.Creates = append(plan.Creates, FileSpec{
		TemplatePath: "resource/biz/biz.go.tpl",
		DestPath:     filepath.Join("internal", appName, "biz", ver, lower, lower+".go"),
		Permissions:  0o644,
	})

	// 3-7. biz 的各 verb 文件
	for _, verb := range []string{"create", "update", "delete", "get", "list"} {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/biz/verb_" + verb + ".go.tpl",
			DestPath:     filepath.Join("internal", appName, "biz", ver, lower, verb+".go"),
			Permissions:  0o644,
		})
	}

	// 8. store/post.go
	plan.Creates = append(plan.Creates, FileSpec{
		TemplatePath: "resource/store.go.tpl",
		DestPath:     filepath.Join("internal", appName, "store", lower+".go"),
		Permissions:  0o644,
	})

	// 9. model/post.gen.go
	plan.Creates = append(plan.Creates, FileSpec{
		TemplatePath: "resource/model.gen.go.tpl",
		DestPath:     filepath.Join("internal", appName, "model", lower+".gen.go"),
		Permissions:  0o644,
	})

	// 根据 With/Without 决定是否生成可选层
	with := computeWithMap(ctx.Features)

	// 10. pkg/conversion/post.go（with:conversion 时生成）
	if with["conversion"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/conversion.go.tpl",
			DestPath:     filepath.Join("internal", appName, "pkg", "conversion", lower+".go"),
			Permissions:  0o644,
		})
	}

	// 11. pkg/validation/post.go（with:validation 时生成）
	if with["validation"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/validation.go.tpl",
			DestPath:     filepath.Join("internal", appName, "pkg", "validation", lower+".go"),
			Permissions:  0o644,
		})
	}

	// 12. internal/pkg/errno/post.go（with:errno 时生成）
	if with["errno"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/errno.go.tpl",
			DestPath:     filepath.Join("internal", "pkg", "errno", lower+".go"),
			Permissions:  0o644,
		})
	}

	// 13. pkg/api/<app>/v1/post.proto（with:proto 时生成）
	if with["proto"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/proto.tpl",
			DestPath:     filepath.Join("pkg", "api", appName, "v1", lower+".proto"),
			Permissions:  0o644,
		})
	}

	// --- Injects ---

	pascal := strcase.ToCamel(res)
	// strcase.ToCamel("post") = "Post"
	// lowerCamel = "post"（针对简单单词）
	_ = lowerCamel

	// 1. biz.go：注入 IBiz.PostV1() 方法 + import + receiver
	bizFile := filepath.Join("internal", appName, "biz", "biz.go")
	bizImportAlias := lower + ver
	bizImportPath := ctx.Module + "/internal/" + appName + "/biz/" + ver + "/" + lower
	plan.Injects = append(plan.Injects, InjectSpec{
		File:    bizFile,
		Mutator: MutatorInterface,
		Payload: linas.InterfacePayload{
			InterfaceName: "IBiz",
			StructName:    "biz",
			Method:        pascal + "V1",
			ReturnType:    bizImportAlias + "." + pascal + "Biz",
			ImportAlias:   bizImportAlias,
			ImportPath:    bizImportPath,
			ImplBody:      "return " + bizImportAlias + ".New(b.store)",
		},
	})

	// 2. store.go：注入 IStore.Posts() 方法 + receiver
	storeFile := filepath.Join("internal", appName, "store", "store.go")
	plural := inflection.Plural(pascal)
	structStore := "datastore"
	if ctx.Storage == "memory" {
		structStore = "memoryStore"
	}
	plan.Injects = append(plan.Injects, InjectSpec{
		File:    storeFile,
		Mutator: MutatorInterface,
		Payload: linas.InterfacePayload{
			InterfaceName: "IStore",
			StructName:    structStore,
			Method:        plural,
			ReturnType:    pascal + "Store",
			ImplBody:      "return new" + pascal + "Store(b)",
		},
	})

	// 3. proto：追加 import "post.proto"
	if with["proto"] {
		protoFile := filepath.Join("pkg", "api", appName, "v1", appName+".proto")
		plan.Injects = append(plan.Injects, InjectSpec{
			File:    protoFile,
			Mutator: MutatorProto,
			Payload: linas.ProtoPayload{
				Import: lower + ".proto",
			},
		})
	}

	// 4. errno/register.go：追加 RegisterErrors(PostErrors()...)
	if with["errno"] {
		registerFile := filepath.Join("internal", "pkg", "errno", "register.go")
		plan.Injects = append(plan.Injects, InjectSpec{
			File:    registerFile,
			Mutator: MutatorRegister,
			Payload: linas.RegisterPayload{
				FunctionName: "RegisterAll",
				Statement:    "RegisterErrors(" + pascal + "Errors()...)",
			},
		})
	}

	return plan, nil
}


// validateResourceName 校验资源名是否为 PascalCase（首字母大写，仅字母数字）。
func validateResourceName(name string) error {
	if name == "" {
		return errs.New(errs.CodeBadResourceName, "scaffold: resource name cannot be empty")
	}
	if !unicode.IsUpper(rune(name[0])) {
		return errs.New(errs.CodeBadResourceName,
			fmt.Sprintf("scaffold: resource name %q must start with uppercase (PascalCase)", name)).
			WithHint("example: linctl add Post  (not post)")
	}
	for _, ch := range name {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) {
			return errs.New(errs.CodeBadResourceName,
				fmt.Sprintf("scaffold: resource name %q contains invalid character %q", name, ch)).
				WithHint("resource names must be PascalCase letters/digits only")
		}
	}
	return nil
}

// computeWith 返回最终生效的 with 列表；当 With/Without 都未设置时返回默认全集。
func computeWith(opts AddOptions) []string {
	defaultWith := []string{"conversion", "validation", "proto", "errno"}
	if len(opts.With) > 0 {
		return opts.With
	}
	if len(opts.Without) > 0 {
		result := []string{}
		excludeSet := map[string]bool{}
		for _, w := range opts.Without {
			excludeSet[w] = true
		}
		for _, f := range defaultWith {
			if !excludeSet[f] {
				result = append(result, f)
			}
		}
		return result
	}
	return defaultWith
}

// computeWithMap 将启用的特性列表转换为 set。
func computeWithMap(features []string) map[string]bool {
	m := map[string]bool{}
	for _, f := range features {
		m[f] = true
	}
	return m
}

// hasProto 判断 features 中是否启用了 proto。
func hasProto(ctx *Context) bool {
	for _, f := range ctx.Features {
		if f == "proto" {
			return true
		}
	}
	return false
}

// cleanupCreated 删除已创建的文件列表（失败回滚）。
func cleanupCreated(paths []string) {
	for _, p := range paths {
		_ = os.Remove(p)
	}
}
