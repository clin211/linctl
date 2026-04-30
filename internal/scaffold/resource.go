package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"

	linas "github.com/clin211/lin/internal/ast"
	"github.com/clin211/lin/internal/pkg/errs"
)

// AddOptions controls the behaviour of AddResource.
type AddOptions struct {
	With        []string // [conversion, validation, proto, errno]
	Without     []string // inverse of With; mutually exclusive
	Ops         []string // [create, update, delete, get, list]
	Version     string   // e.g. "v1"
	Plural      string   // override plural form
	NoInject    bool     // skip AST injection
	SkipImports bool     // skip import statements
}

// AddResource is the core of lin add.
//
// Design source: 02 §4.5 behaviour flow, 03 §1 resource layers, 05 §5 injection order.
func AddResource(ctx *Context, name string, opts AddOptions) error {
	// 1. Validate resource name (must be PascalCase)
	if err := validateResourceName(name); err != nil {
		return err
	}

	// 2. Validate With/Without mutex
	if len(opts.With) > 0 && len(opts.Without) > 0 {
		return errs.New(errs.CodeFlagConflict,
			"scaffold: --with and --without are mutually exclusive").
			WithHint("use either --with or --without, not both")
	}

	// 3. Set resource on context
	ctx.Resource = name
	ctx.Features = computeWith(opts)

	// 4. BuildPlan
	plan, err := BuildPlan(ctx, PlanKindResource)
	if err != nil {
		return err
	}

	// 5. Print summary
	fmt.Printf("✔ resource: %s\n", name)
	fmt.Printf("   + %d files to create\n", len(plan.Creates))
	if !opts.NoInject {
		fmt.Printf("   ✏  %d files to update via AST\n", len(plan.Injects))
	}

	// 6. DryRun: print plan only
	if ctx.DryRun {
		for _, spec := range plan.Creates {
			fmt.Printf("    + %s\n", spec.DestPath)
		}
		for _, spec := range plan.Injects {
			fmt.Printf("    ✏  %s (%s)\n", spec.File, spec.Mutator)
		}
		return nil
	}

	// 7. Execute with rollback on failure
	ts := linas.NewTimestamp()
	var created []string

	// 7a. Render (create new files)
	vars := newTemplateVars(ctx)
	for _, spec := range plan.Creates {
		dstPath := filepath.Join(ctx.RootDir, spec.DestPath)
		if _, err := os.Stat(dstPath); err == nil && !ctx.Force {
			fmt.Printf("   ⊝ skip (exists): %s\n", spec.DestPath)
			continue
		}
		if err := renderOne(ctx, spec, vars); err != nil {
			// Rollback: delete created files
			cleanupCreated(created)
			return fmt.Errorf("create %s: %w", spec.DestPath, err)
		}
		created = append(created, dstPath)
		fmt.Printf("   + %s\n", spec.DestPath)
	}

	// 7b. AST injection
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
			// Rollback: delete created files
			cleanupCreated(created)
			return fmt.Errorf("ast inject: %w", err)
		}
		for _, spec := range plan.Injects {
			fmt.Printf("   ✏  %s\n", spec.File)
		}
	}

	// 8. Ensure .gitignore has backup exclusion
	ensureGitignore(ctx.RootDir, ts)

	// 9. Next steps
	fmt.Printf("📦 Next steps:\n")
	if hasProto(ctx) {
		fmt.Printf("   make protoc\n")
	}
	fmt.Printf("   go mod tidy\n")
	fmt.Printf("   go build ./...\n")

	return nil
}

// buildResourcePlan constructs the Plan for PlanKindResource.
func buildResourcePlan(ctx *Context) (*Plan, error) {
	res := ctx.Resource
	lower := strings.ToLower(res)
	// strcase.ToLowerCamel returns lower-camel, e.g. "PostItem" → "postItem"
	// but for a simple word like "Post" it returns "post"
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

	// 2. biz/v1/post/post.go (interface + struct + New)
	plan.Creates = append(plan.Creates, FileSpec{
		TemplatePath: "resource/biz/biz.go.tpl",
		DestPath:     filepath.Join("internal", appName, "biz", ver, lower, lower+".go"),
		Permissions:  0o644,
	})

	// 3-7. biz verb files
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

	// Optional layers based on With/Without
	with := computeWithMap(ctx.Features)

	// 10. pkg/conversion/post.go (if with:conversion)
	if with["conversion"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/conversion.go.tpl",
			DestPath:     filepath.Join("internal", appName, "pkg", "conversion", lower+".go"),
			Permissions:  0o644,
		})
	}

	// 11. pkg/validation/post.go (if with:validation)
	if with["validation"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/validation.go.tpl",
			DestPath:     filepath.Join("internal", appName, "pkg", "validation", lower+".go"),
			Permissions:  0o644,
		})
	}

	// 12. internal/pkg/errno/post.go (if with:errno)
	if with["errno"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/errno.go.tpl",
			DestPath:     filepath.Join("internal", "pkg", "errno", lower+".go"),
			Permissions:  0o644,
		})
	}

	// 13. pkg/api/<app>/v1/post.proto (if with:proto)
	if with["proto"] {
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/proto.tpl",
			DestPath:     filepath.Join("pkg", "api", appName, "v1", lower+".proto"),
			Permissions:  0o644,
		})
		// Placeholder Go types (allows go build before `make protoc`)
		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: "resource/proto_go.go.tpl",
			DestPath:     filepath.Join("pkg", "api", appName, "v1", lower+"_lin.go"),
			Permissions:  0o644,
		})
	}

	// --- Injects ---

	pascal := strcase.ToCamel(res)
	// strcase.ToCamel("post") = "Post"
	// lowerCamel = "post" (for simple names)
	_ = lowerCamel

	// 1. biz.go: add IBiz.PostV1() method + import + receiver
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

	// 2. store.go: add IStore.Posts() method + receiver
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

	// 3. proto: add import "post.proto"
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

	// 4. errno/register.go: RegisterErrors(PostErrors()...)
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


// validateResourceName checks that name is PascalCase (starts with uppercase, no special chars).
func validateResourceName(name string) error {
	if name == "" {
		return errs.New(errs.CodeBadResourceName, "scaffold: resource name cannot be empty")
	}
	if !unicode.IsUpper(rune(name[0])) {
		return errs.New(errs.CodeBadResourceName,
			fmt.Sprintf("scaffold: resource name %q must start with uppercase (PascalCase)", name)).
			WithHint("example: lin add Post  (not post)")
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

// computeWith returns the effective with-features list, defaulting to all if neither With nor Without is set.
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

// computeWithMap returns a set of enabled features.
func computeWithMap(features []string) map[string]bool {
	m := map[string]bool{}
	for _, f := range features {
		m[f] = true
	}
	return m
}

// hasProto checks whether proto is in features.
func hasProto(ctx *Context) bool {
	for _, f := range ctx.Features {
		if f == "proto" {
			return true
		}
	}
	return false
}

// cleanupCreated removes the list of created files (rollback on failure).
func cleanupCreated(paths []string) {
	for _, p := range paths {
		_ = os.Remove(p)
	}
}

// ensureGitignore ensures .lin/.backup/ is excluded from git.
func ensureGitignore(rootDir, _ string) {
	giPath := filepath.Join(rootDir, ".gitignore")
	data, err := os.ReadFile(giPath)
	if err != nil {
		// Can't read .gitignore, skip
		return
	}
	content := string(data)
	const backupLine = ".lin/.backup/"
	if strings.Contains(content, backupLine) {
		return
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n# lin scaffolding tool runtime files\n" + backupLine + "\n.lin/.last-run.json\n"
	_ = os.WriteFile(giPath, []byte(content), 0o644)
	fmt.Println("   ⚠  updated .gitignore (added .lin/.backup/ exclusion)")
}
