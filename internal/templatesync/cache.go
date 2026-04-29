package templatesync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/clin211/lin/internal/linctlerr"
)

// cacheRoot 是 lin 仓库内 base content 缓存的相对路径（相对 lin/）。
const cacheRoot = ".linctl/upstream-sync-cache"

// baseCache 管理 "上次同步落盘内容" 的内容寻址缓存。
//
// 设计要点：
//   - 写入时键 = sha256:<hex>；存储路径 = .linctl/upstream-sync-cache/<algo>-<hex>
//   - 读取时按 hash 查找；命中即返回内容
//   - 不在缓存中的 base 视为 "无 base 信息"，调用方应降级为非 3-way 路径
//
// 为什么需要缓存：
//   3-way merge 需要 base 内容，而 base = "上次同步时 lin/templates/web-gin/ 写盘的产物"。
//   仅记 hash 是不够的，必须保留实际字节。
type baseCache struct {
	root string // 绝对路径
}

// newBaseCache 构造一个 baseCache。root 为 lin 仓库根目录绝对路径。
func newBaseCache(linRoot string) *baseCache {
	return &baseCache{
		root: filepath.Join(linRoot, cacheRoot),
	}
}

// path 把 sha256:<hex> 形式的哈希映射到磁盘路径。
//
// 实现细节：
//   - 拒绝接受不带 "sha256:" 前缀的哈希（防止与未来其他算法混淆）
//   - 用 "/" 拆为目录前缀（前两位）+ 文件名（剩余位），减少单目录文件数
func (c *baseCache) path(hashWithAlgo string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(hashWithAlgo, prefix) {
		return "", linctlerr.Newf(linctlerr.ErrInternal,
			"baseCache: only sha256: prefixed hashes accepted; got %q", hashWithAlgo)
	}
	hex := hashWithAlgo[len(prefix):]
	if len(hex) < 4 {
		return "", linctlerr.Newf(linctlerr.ErrInternal,
			"baseCache: hex too short: %q", hex)
	}
	return filepath.Join(c.root, "sha256", hex[:2], hex[2:]), nil
}

// Put 把内容写入缓存，键 = hash。已存在时为 no-op（内容寻址保证 idempotent）。
func (c *baseCache) Put(hashWithAlgo string, content []byte) error {
	p, err := c.path(hashWithAlgo)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(p); statErr == nil {
		return nil // 已缓存
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"baseCache: mkdir %s", filepath.Dir(p))
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"baseCache: write tmp %s", tmp)
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"baseCache: rename %s -> %s", tmp, p)
	}
	return nil
}

// Get 按 hash 取内容；不存在返回 (nil, false, nil)。
func (c *baseCache) Get(hashWithAlgo string) ([]byte, bool, error) {
	p, err := c.path(hashWithAlgo)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"baseCache: read %s", p)
	}
	return data, true, nil
}

// Has 是 Get 的轻量版，仅返回是否命中。
func (c *baseCache) Has(hashWithAlgo string) bool {
	p, err := c.path(hashWithAlgo)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Root 返回缓存根目录绝对路径（仅供调试）。
func (c *baseCache) Root() string { return c.root }
