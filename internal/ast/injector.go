// Package ast implements AST-based code injection for lin v2.
//
// The injector locates insertion points purely from Go AST structure
// (interface names, struct receivers, top-level functions) — there are
// no comment markers or anchor regions involved.
//
// Design source: lin/docs/features/05-registration-strategy.md.
package ast

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/clin211/lin/internal/pkg/errs"
)

// Injector orchestrates backup, mutation, and rollback of central files.
type Injector struct {
	rootDir   string
	timestamp string
	backed    []backedFile
}

type backedFile struct {
	relPath    string
	backupPath string
}

// NewInjector creates an Injector for the given project root.
func NewInjector(rootDir string) *Injector {
	return &Injector{
		rootDir:   rootDir,
		timestamp: NewTimestamp(),
	}
}

// Inject executes all InjectSpecs in plan.Specs.
//
// Algorithm:
//  1. Backup all target files.
//  2. Execute each mutator in order.
//  3. On any failure, restore all backed-up files and return a wrapped error.
//  4. On success, clean up the backup directory.
func (inj *Injector) Inject(plan InjectPlan) error {
	seen := map[string]bool{}
	for _, spec := range plan.Specs {
		if seen[spec.File] {
			continue
		}
		seen[spec.File] = true
		abs := filepath.Join(inj.rootDir, spec.File)
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			continue
		}
		bp, err := BackupFile(inj.rootDir, spec.File, inj.timestamp)
		if err != nil {
			return errs.Wrap(errs.CodeASTBackupFailed,
				fmt.Sprintf("ast: backup failed for %s", spec.File), err)
		}
		inj.backed = append(inj.backed, backedFile{relPath: spec.File, backupPath: bp})
	}

	for _, spec := range plan.Specs {
		absFile := filepath.Join(inj.rootDir, spec.File)
		if err := inj.applySpec(absFile, spec); err != nil {
			_ = inj.Restore()
			return fmt.Errorf("inject %s (%s): %w", spec.File, spec.Mutator, err)
		}
	}

	_ = CleanupBackup(inj.rootDir, inj.timestamp)
	return nil
}

// Restore restores all backed-up files to their original paths.
func (inj *Injector) Restore() error {
	var firstErr error
	for _, bf := range inj.backed {
		origPath := filepath.Join(inj.rootDir, bf.relPath)
		if err := RestoreFromBackup(bf.backupPath, origPath); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// applySpec dispatches to the appropriate mutator.
func (inj *Injector) applySpec(file string, spec InjectSpec) error {
	switch spec.Mutator {
	case MutatorKindInterface:
		p, ok := spec.Payload.(InterfacePayload)
		if !ok {
			return errs.New(errs.CodeASTApplyError,
				fmt.Sprintf("ast: bad payload type for interface mutator in %s", file))
		}
		return AddInterfaceMethod(file, p)

	case MutatorKindProto:
		p, ok := spec.Payload.(ProtoPayload)
		if !ok {
			return errs.New(errs.CodeASTApplyError,
				fmt.Sprintf("ast: bad payload type for proto mutator in %s", file))
		}
		p.File = file
		return AddProtoImport(file, p)

	case MutatorKindRegister:
		p, ok := spec.Payload.(RegisterPayload)
		if !ok {
			return errs.New(errs.CodeASTApplyError,
				fmt.Sprintf("ast: bad payload type for register mutator in %s", file))
		}
		return AppendRegistration(file, p)

	default:
		return errs.New(errs.CodeASTApplyError,
			fmt.Sprintf("ast: unknown mutator kind %q", spec.Mutator))
	}
}

// MutatorKind identifies the AST mutator type.
type MutatorKind string

const (
	MutatorKindInterface MutatorKind = "interface"
	MutatorKindProto     MutatorKind = "proto"
	MutatorKindRegister  MutatorKind = "register"
)

// InjectSpec describes a single AST injection.
type InjectSpec struct {
	File    string
	Mutator MutatorKind
	Payload any
}

// InjectPlan groups all injection specs for one operation.
type InjectPlan struct {
	Specs []InjectSpec
}

// ParseFile parses a Go source file into a dst.File.
func ParseFile(path string) (*dst.File, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(errs.CodeASTParseError, fmt.Sprintf("ast: read %s", path), err)
	}
	f, err := decorator.Parse(src)
	if err != nil {
		return nil, errs.Wrap(errs.CodeASTParseError, fmt.Sprintf("ast: parse %s", path), err)
	}
	return f, nil
}

// WriteFile serialises a dst.File back to the given path.
func WriteFile(path string, f *dst.File) error {
	pr, pw := io.Pipe()
	var writeErr error
	go func() {
		r := decorator.NewRestorer()
		writeErr = r.Fprint(pw, f)
		_ = pw.CloseWithError(writeErr)
	}()
	out, err := io.ReadAll(pr)
	if err != nil {
		return errs.Wrap(errs.CodeASTApplyError, fmt.Sprintf("ast: restore %s", path), err)
	}
	if writeErr != nil {
		return errs.Wrap(errs.CodeASTApplyError, fmt.Sprintf("ast: restore %s", path), writeErr)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return errs.Wrap(errs.CodeASTApplyError, fmt.Sprintf("ast: write %s", path), err)
	}
	return nil
}

// BackupFile copies src (relative to rootDir) to .lin/.backup/<ts>/<rel>.
// Returns the backup path.
func BackupFile(rootDir, relPath, ts string) (string, error) {
	src := filepath.Join(rootDir, relPath)
	dstPath := filepath.Join(rootDir, ".lin", ".backup", ts, relPath)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return "", errs.Wrap(errs.CodeASTBackupFailed,
			fmt.Sprintf("ast: backup mkdir for %s", relPath), err)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return "", errs.Wrap(errs.CodeASTBackupFailed,
			fmt.Sprintf("ast: backup read %s", src), err)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return "", errs.Wrap(errs.CodeASTBackupFailed,
			fmt.Sprintf("ast: backup write %s", dstPath), err)
	}
	return dstPath, nil
}

// RestoreFromBackup copies backupPath back to originalPath.
func RestoreFromBackup(backupPath, originalPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return errs.Wrap(errs.CodeASTBackupFailed,
			fmt.Sprintf("ast: restore read %s", backupPath), err)
	}
	if err := os.WriteFile(originalPath, data, 0o644); err != nil {
		return errs.Wrap(errs.CodeASTBackupFailed,
			fmt.Sprintf("ast: restore write %s", originalPath), err)
	}
	return nil
}

// CleanupBackup removes the backup directory for a given timestamp.
// Also prunes oldest backups keeping only the 3 most recent.
func CleanupBackup(rootDir, ts string) error {
	backupDir := filepath.Join(rootDir, ".lin", ".backup", ts)
	if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
		return errs.Wrap(errs.CodeASTBackupFailed,
			fmt.Sprintf("ast: cleanup backup %s", ts), err)
	}
	pruneOldBackups(rootDir)
	return nil
}

// NewTimestamp returns a timestamp string suitable for backup directory names.
func NewTimestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

// pruneOldBackups keeps only the 3 most recent backup directories.
func pruneOldBackups(rootDir string) {
	parent := filepath.Join(rootDir, ".lin", ".backup")
	entries, err := os.ReadDir(parent)
	if err != nil {
		return
	}
	if len(entries) <= 3 {
		return
	}
	for _, e := range entries[:len(entries)-3] {
		_ = os.RemoveAll(filepath.Join(parent, e.Name()))
	}
}
