package templatesync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBaseCache_PutGet(t *testing.T) {
	dir := t.TempDir()
	c := newBaseCache(dir)

	hash := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	content := []byte("hello world\n")

	// Get on empty cache → miss
	if got, hit, err := c.Get(hash); err != nil || hit || got != nil {
		t.Fatalf("expected miss; got hit=%v err=%v", hit, err)
	}

	// Put then Get
	if err := c.Put(hash, content); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, hit, err := c.Get(hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected hit after Put")
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q want %q", got, content)
	}

	// Path layout sanity: <root>/sha256/<2>/<rest>
	wantPath := filepath.Join(dir, ".linctl/upstream-sync-cache/sha256",
		"01", "23456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if _, err := os.Stat(wantPath); err != nil {
		t.Errorf("expected cache file at %s; err: %v", wantPath, err)
	}
}

func TestBaseCache_RejectInvalidHash(t *testing.T) {
	c := newBaseCache(t.TempDir())

	_, err := c.path("md5:abc")
	if err == nil || !strings.Contains(err.Error(), "sha256:") {
		t.Errorf("expected sha256-only error; got %v", err)
	}

	_, err = c.path("sha256:ab") // 太短
	if err == nil {
		t.Errorf("expected too-short error")
	}
}

func TestBaseCache_PutIdempotent(t *testing.T) {
	dir := t.TempDir()
	c := newBaseCache(dir)
	hash := "sha256:abcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd"

	if err := c.Put(hash, []byte("x")); err != nil {
		t.Fatalf("Put #1: %v", err)
	}
	// 第二次 Put 同 hash 应直接 no-op（不报错；不重写）
	if err := c.Put(hash, []byte("y")); err != nil {
		t.Fatalf("Put #2: %v", err)
	}
	got, _, _ := c.Get(hash)
	// 内容寻址：相同 hash 视为相同内容；第二次 Put 不覆盖
	if string(got) != "x" {
		t.Errorf("Put should be idempotent; got %q", got)
	}
}

func TestBaseCache_Has(t *testing.T) {
	c := newBaseCache(t.TempDir())
	hash := "sha256:dcba9876dcba9876dcba9876dcba9876dcba9876dcba9876dcba9876dcba9876"
	if c.Has(hash) {
		t.Error("Has should be false for unset hash")
	}
	_ = c.Put(hash, []byte("z"))
	if !c.Has(hash) {
		t.Error("Has should be true after Put")
	}
}
