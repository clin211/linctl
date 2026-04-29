package templatesync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/clin211/lin/internal/linctlerr"
)

// 当前 lockfile 格式版本。
const lockSchemaVersion = "1"

// Lockfile 记录 lin 仓库内"模板上游同步"的当前状态。
//
// 文件位置：lin/.linctl/upstream-sync.lock.json（提交进 git）。
// 用途：① 算 plan 时知道每文件的"base hash"做 3-way merge；② CI 断言上游漂移。
type Lockfile struct {
	SchemaVersion string                 `json:"schemaVersion"`
	SyncManifest  string                 `json:"syncManifest"`         // sync.yaml 相对项目根的路径
	LastSyncAt    time.Time              `json:"lastSyncAt"`
	LinctlVersion string                 `json:"linctlVersion"`
	Upstream      LockUpstream           `json:"upstream"`
	Files         map[string]LockedFile  `json:"files"`

	// 私有字段：记录 lockfile 自身路径，便于 Save 写回。
	path string
}

// LockUpstream 记录上游版本信息。
type LockUpstream struct {
	Name               string `json:"name"`
	RootPath           string `json:"rootPath"`
	LastSyncedCommit   string `json:"lastSyncedCommit,omitempty"`   // 上游 git commit（如可用）
	LastSyncedTreeHash string `json:"lastSyncedTreeHash,omitempty"` // 备用：聚合 tree hash
}

// LockedFile 是 lockfile 中单文件的记录。
type LockedFile struct {
	Src               string    `json:"src,omitempty"`
	Owner             Owner     `json:"owner"`
	SrcHashAtSync     string    `json:"srcHashAtSync,omitempty"`     // 上次同步时的源文件 hash（base）
	DstHashAtSync     string    `json:"dstHashAtSync,omitempty"`     // 上次同步后写盘的产物 hash
	TransformsApplied []string  `json:"transformsApplied,omitempty"` // 应用过的 transform kind 列表
	LastSyncAt        time.Time `json:"lastSyncAt,omitempty"`
}

// LoadLockfile 从给定路径加载 lockfile；不存在时返回一个空 Lockfile（不报错）。
//
// path 通常是 <linRoot>/.linctl/upstream-sync.lock.json。
func LoadLockfile(path string) (*Lockfile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"resolve lockfile path %s", path)
	}

	lf := &Lockfile{
		SchemaVersion: lockSchemaVersion,
		Files:         make(map[string]LockedFile, 64),
		path:          abs,
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return lf, nil // 首次运行；空 lockfile
		}
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"read lockfile %s", path)
	}

	if err := json.Unmarshal(data, lf); err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrConfigInvalid, err,
			"parse lockfile %s", path)
	}

	if lf.SchemaVersion != lockSchemaVersion {
		return nil, linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"lockfile schemaVersion=%q; expected %q (consider lockfile migration)",
			lf.SchemaVersion, lockSchemaVersion)
	}
	if lf.Files == nil {
		lf.Files = make(map[string]LockedFile, 64)
	}
	lf.path = abs
	return lf, nil
}

// Save 把 lockfile 写回磁盘（原子写：tmp + rename）。
//
// 写入前对 Files 按 dst key 排序输出，保证 git diff 友好。
func (lf *Lockfile) Save() error {
	if lf == nil {
		return linctlerr.New(linctlerr.ErrInternal, "lockfile: nil receiver")
	}
	if lf.path == "" {
		return linctlerr.New(linctlerr.ErrInternal,
			"lockfile: path not set; use LoadLockfile or set explicitly via SetPath")
	}

	if err := os.MkdirAll(filepath.Dir(lf.path), 0o755); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"mkdir %s", filepath.Dir(lf.path))
	}

	// 重排 Files 为有序 slice 然后再 marshal，避免 JSON 输出顺序漂移。
	type fileEntry struct {
		Dst string
		LockedFile
	}
	keys := make([]string, 0, len(lf.Files))
	for k := range lf.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 用 OrderedMap 风格：直接编码到 JSON 时 keys 已 sort，需要手工序列化
	// → 简单做法：用 map 直接 marshal（json 包 Go 1.12+ 自带 key-sorted 输出）
	out := struct {
		SchemaVersion string                `json:"schemaVersion"`
		SyncManifest  string                `json:"syncManifest"`
		LastSyncAt    time.Time             `json:"lastSyncAt"`
		LinctlVersion string                `json:"linctlVersion"`
		Upstream      LockUpstream          `json:"upstream"`
		Files         map[string]LockedFile `json:"files"`
	}{
		SchemaVersion: lf.SchemaVersion,
		SyncManifest:  lf.SyncManifest,
		LastSyncAt:    lf.LastSyncAt,
		LinctlVersion: lf.LinctlVersion,
		Upstream:      lf.Upstream,
		Files:         lf.Files,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return linctlerr.Wrapf(linctlerr.ErrInternal, err, "marshal lockfile")
	}

	tmp := lf.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"write tmp lockfile %s", tmp)
	}
	if err := os.Rename(tmp, lf.path); err != nil {
		_ = os.Remove(tmp)
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"rename %s -> %s", tmp, lf.path)
	}
	return nil
}

// SetPath 显式设置 lockfile 落盘路径（用于刚 New 出来未走 Load 的场景）。
func (lf *Lockfile) SetPath(p string) {
	lf.path = p
}

// Get 按 dst 查询单个 LockedFile；不存在时返回 (zero, false)。
func (lf *Lockfile) Get(dst string) (LockedFile, bool) {
	v, ok := lf.Files[dst]
	return v, ok
}

// Update 更新或插入单个 dst 的 LockedFile 记录。
func (lf *Lockfile) Update(dst string, e LockedFile) {
	if lf.Files == nil {
		lf.Files = make(map[string]LockedFile, 64)
	}
	lf.Files[dst] = e
}

// Delete 移除单个 dst 的记录。
func (lf *Lockfile) Delete(dst string) {
	delete(lf.Files, dst)
}

// AllDsts 返回 lockfile 已知的全部 dst（字典序）。
func (lf *Lockfile) AllDsts() []string {
	out := make([]string, 0, len(lf.Files))
	for k := range lf.Files {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
