package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyAndRestore(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	src := filepath.Join(home, "config.toml")
	if err := os.WriteFile(src, []byte("hello = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dst, err := CopyFile(home, src, "config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if dst == "" {
		t.Fatal("expected backup path")
	}
	if err := os.WriteFile(src, []byte("hello = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Restore(dst, src); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello = 1\n" {
		t.Fatalf("got %q", got)
	}
}

func TestCopyMissing(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	dst, err := CopyFile(home, filepath.Join(home, "nope.toml"), "config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if dst != "" {
		t.Fatalf("expected empty, got %s", dst)
	}
}
