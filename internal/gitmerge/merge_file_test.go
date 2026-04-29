package gitmerge

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMergeFile_NonConflict(t *testing.T) {
	ctx := context.Background()
	base := []byte("a\nb\nc\n")
	ours := []byte("a\nb\nc\nd\n")    // 加了 d
	theirs := []byte("z\na\nb\nc\n")  // 加了 z

	res, err := MergeFile(ctx, Files{Base: base, Ours: ours, Theirs: theirs}, StrategyAuto)
	if err != nil {
		t.Fatalf("MergeFile error: %v", err)
	}
	if res.HasConflict {
		t.Errorf("expected no conflict; got %s", string(res.Content))
	}
	want := "z\na\nb\nc\nd\n"
	if string(res.Content) != want {
		t.Errorf("content = %q; want %q", string(res.Content), want)
	}
}

func TestMergeFile_Conflict_AutoStrategy(t *testing.T) {
	ctx := context.Background()
	base := []byte("line1\nline2\nline3\n")
	ours := []byte("line1\nMINE\nline3\n")
	theirs := []byte("line1\nTHEIRS\nline3\n")

	res, err := MergeFile(ctx, Files{
		Base:   base,
		Ours:   ours,
		Theirs: theirs,
		Labels: Labels{Ours: "lin-side", Base: "v0.3.0", Theirs: "miniblog-v4"},
	}, StrategyAuto)
	if err != nil {
		t.Fatalf("MergeFile error: %v", err)
	}
	if !res.HasConflict {
		t.Errorf("expected conflict; got clean output:\n%s", string(res.Content))
	}
	if !bytes.Contains(res.Content, []byte("<<<<<<<")) {
		t.Errorf("expected '<<<<<<<' marker in output:\n%s", string(res.Content))
	}
	if !bytes.Contains(res.Content, []byte("=======")) {
		t.Errorf("expected '=======' marker in output")
	}
	if !bytes.Contains(res.Content, []byte(">>>>>>>")) {
		t.Errorf("expected '>>>>>>>' marker in output")
	}
}

func TestMergeFile_Strategy_Ours(t *testing.T) {
	ctx := context.Background()
	base := []byte("a\nb\nc\n")
	ours := []byte("a\nMINE\nc\n")
	theirs := []byte("a\nTHEIRS\nc\n")

	res, err := MergeFile(ctx, Files{Base: base, Ours: ours, Theirs: theirs}, StrategyOurs)
	if err != nil {
		t.Fatalf("MergeFile error: %v", err)
	}
	// strategy=ours：冲突自动取 ours，不应留 markers
	if bytes.Contains(res.Content, []byte("<<<<<<<")) {
		t.Errorf("ours strategy should not leave markers:\n%s", string(res.Content))
	}
	if !bytes.Contains(res.Content, []byte("MINE")) {
		t.Errorf("expected MINE in output:\n%s", string(res.Content))
	}
}

func TestMergeFile_Strategy_Theirs(t *testing.T) {
	ctx := context.Background()
	base := []byte("a\nb\nc\n")
	ours := []byte("a\nMINE\nc\n")
	theirs := []byte("a\nTHEIRS\nc\n")

	res, err := MergeFile(ctx, Files{Base: base, Ours: ours, Theirs: theirs}, StrategyTheirs)
	if err != nil {
		t.Fatalf("MergeFile error: %v", err)
	}
	if !bytes.Contains(res.Content, []byte("THEIRS")) {
		t.Errorf("expected THEIRS in output:\n%s", string(res.Content))
	}
}

func TestHasConflictMarkers(t *testing.T) {
	tests := map[string]bool{
		"<<<<<<< ours\nfoo\n=======\nbar\n>>>>>>> theirs\n": true,
		"normal content\nno conflict here\n":                false,
		"\n<<<<<<< ours\n":                                  true,
		"<<<<<<<":                                           true,
	}
	for content, want := range tests {
		got := HasConflictMarkers([]byte(content))
		if got != want {
			t.Errorf("HasConflictMarkers(%q) = %v; want %v", content, got, want)
		}
	}
}

func TestMergeFallback_Conflict(t *testing.T) {
	// 直接测试 fallback 实现，绕开系统 git
	base := []byte("line1\nline2\nline3\n")
	ours := []byte("line1\nMINE\nline3\n")
	theirs := []byte("line1\nTHEIRS\nline3\n")

	res, err := mergeWithFallback(Files{
		Base:   base,
		Ours:   ours,
		Theirs: theirs,
		Labels: Labels{Ours: "lin-side", Base: "v0.3.0", Theirs: "miniblog-v4"},
	}, StrategyAuto)
	if err != nil {
		t.Fatalf("mergeWithFallback error: %v", err)
	}
	if !res.HasConflict {
		t.Errorf("expected conflict; got:\n%s", string(res.Content))
	}
	if res.Backend != "diff3-fallback" {
		t.Errorf("expected backend=diff3-fallback; got %s", res.Backend)
	}
	if !strings.Contains(string(res.Content), "<<<<<<<") {
		t.Errorf("expected markers; got:\n%s", string(res.Content))
	}
}

func TestMergeFallback_NonConflict(t *testing.T) {
	base := []byte("a\nb\nc\n")
	ours := []byte("a\nb\nc\nd\n")
	theirs := []byte("z\na\nb\nc\n")

	res, err := mergeWithFallback(Files{Base: base, Ours: ours, Theirs: theirs}, StrategyAuto)
	if err != nil {
		t.Fatalf("mergeWithFallback error: %v", err)
	}
	if res.HasConflict {
		t.Errorf("unexpected conflict:\n%s", string(res.Content))
	}
	want := "z\na\nb\nc\nd\n"
	if string(res.Content) != want {
		t.Errorf("content = %q; want %q", string(res.Content), want)
	}
}

func TestMergeFallback_BothEqualEdit(t *testing.T) {
	// 双方做了相同改动 → 自动采纳
	base := []byte("a\nb\nc\n")
	ours := []byte("a\nB\nc\n")
	theirs := []byte("a\nB\nc\n")

	res, err := mergeWithFallback(Files{Base: base, Ours: ours, Theirs: theirs}, StrategyAuto)
	if err != nil {
		t.Fatalf("mergeWithFallback error: %v", err)
	}
	if res.HasConflict {
		t.Errorf("unexpected conflict for identical edits")
	}
	if string(res.Content) != "a\nB\nc\n" {
		t.Errorf("content = %q; want %q", string(res.Content), "a\nB\nc\n")
	}
}

func TestMergeFile_NilInput(t *testing.T) {
	_, err := MergeFile(context.Background(), Files{Base: nil, Ours: []byte{}, Theirs: []byte{}}, StrategyAuto)
	if err == nil {
		t.Errorf("expected error for nil Base")
	}
}

func TestIsAvailable(t *testing.T) {
	// 在大多数 dev 机器上 git 都可用；CI 上也通常有
	if !IsAvailable() {
		t.Skip("git not available in this environment")
	}
}
