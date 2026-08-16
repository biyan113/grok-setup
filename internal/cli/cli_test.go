package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if code := Main([]string{"version"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestApplyDryRunThroughCLI(t *testing.T) {
	home := t.TempDir()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := Main([]string{
		"apply",
		"--home", home,
		"--alias", "proxy",
		"--base-url", "https://gw.example.com/v1",
		"--env-key", "GATEWAY_API_KEY",
		"--set-default",
		"--search",
		"--dry-run",
		"--yes",
	})
	_ = w.Close()
	os.Stdout = oldStdout
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	out := string(buf[:n])
	if !strings.Contains(out, "GATEWAY_API_KEY") {
		t.Fatalf("expected env_key in output:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, "config.toml")); !os.IsNotExist(err) {
		t.Fatal("dry-run should not write")
	}
}

func TestUnknownCommand(t *testing.T) {
	if code := Main([]string{"nope"}); code == 0 {
		t.Fatal("expected non-zero")
	}
}
