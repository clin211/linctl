// Package ast 实现 lin v2 的基于 AST 的代码注入。
//
// injector 完全依据 Go AST 结构（接口名、struct receiver、顶层函数）
// 定位插入点 —— 不依赖任何注释标记或锚点区域。
//
// 设计来源：lin/docs/features/05-registration-strategy.md。
package ast

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// backupEnv 允许测试通过该环境变量重定向备份根目录。
const backupEnv = "LINCTL_BACKUP_DIR"

// projectBackupBase 返回项目级别的备份根目录（位于用户级缓存下，
// 因此备份文件不会污染用户的项目目录）。
//
// 目录结构：<UserCacheDir>/linctl/backups/<basename(rootDir)>-<short-hash>
// 测试可通过 LINCTL_BACKUP_DIR 环境变量整体覆写根目录。
func projectBackupBase(rootDir string) (string, error) {
	if env := os.Getenv(backupEnv); env != "" {
		return env, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	abs, absErr := filepath.Abs(rootDir)
	if absErr != nil {
		abs = rootDir
	}
	h := sha256.Sum256([]byte(abs))
	name := filepath.Base(abs) + "-" + hex.EncodeToString(h[:6])
	return filepath.Join(cacheDir, "linctl", "backups", name), nil
}

// Injector 编排中心文件的备份、变更与回滚。
type Injector struct {
	rootDir   string
	timestamp string
	backed    []backedFile
}

type backedFile struct {
	relPath    string
	backupPath string
}

// NewInjector 为指定的项目根目录创建一个 Injector。
func NewInjector(rootDir string) *Injector {
	return &Injector{
		rootDir:   rootDir,
		timestamp: NewTimestamp(),
	}
}

// Inject 执行 plan.Specs 中的所有 InjectSpec。
//
// 算法：
//  1. 备份所有目标文件。
//  2. 按顺序执行各个 mutator。
//  3. 任一步骤失败则恢复全部备份并返回包装后的错误。
//  4. 全部成功后清理备份目录。
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

// Restore 将所有已备份文件恢复到原始路径。
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

// applySpec 将 spec 分发到对应的 mutator。
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

// MutatorKind 标识 AST mutator 的类型。
type MutatorKind string

const (
	MutatorKindInterface MutatorKind = "interface"
	MutatorKindProto     MutatorKind = "proto"
	MutatorKindRegister  MutatorKind = "register"
)

// InjectSpec 描述一次 AST 注入。
type InjectSpec struct {
	File    string
	Mutator MutatorKind
	Payload any
}

// InjectPlan 聚合一次注入操作中的所有 spec。
type InjectPlan struct {
	Specs []InjectSpec
}

// ParseFile 将一个 Go 源文件解析为 dst.File。
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

// WriteFile 将 dst.File 重新序列化并写回指定路径。
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

// BackupFile 将 src（相对 rootDir 的路径）拷贝到用户级备份缓存：
// <projectBackupBase>/<ts>/<rel>。返回备份后文件的路径。
func BackupFile(rootDir, relPath, ts string) (string, error) {
	src := filepath.Join(rootDir, relPath)
	base, err := projectBackupBase(rootDir)
	if err != nil {
		return "", errs.Wrap(errs.CodeASTBackupFailed,
			"ast: locate user cache dir", err)
	}
	dstPath := filepath.Join(base, ts, relPath)

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

// RestoreFromBackup 将 backupPath 恢复回 originalPath。
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

// CleanupBackup 删除指定时间戳对应的备份目录，
// 并裁剪历史备份只保留最近的 3 份。
func CleanupBackup(rootDir, ts string) error {
	base, err := projectBackupBase(rootDir)
	if err != nil {
		return errs.Wrap(errs.CodeASTBackupFailed,
			"ast: locate user cache dir", err)
	}
	backupDir := filepath.Join(base, ts)
	if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
		return errs.Wrap(errs.CodeASTBackupFailed,
			fmt.Sprintf("ast: cleanup backup %s", ts), err)
	}
	pruneOldBackups(rootDir)
	return nil
}

// NewTimestamp 返回一个适合用作备份目录名的时间戳。
func NewTimestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

// pruneOldBackups 仅保留最近的 3 个备份目录。
func pruneOldBackups(rootDir string) {
	parent, err := projectBackupBase(rootDir)
	if err != nil {
		return
	}
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
