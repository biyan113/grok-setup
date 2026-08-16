package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biyan113/grok-setup/internal/cfg"
	"github.com/biyan113/grok-setup/internal/paths"
)

func TestApplyMergesAndPreservesMCP(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	existing := []byte(`
[cli]
installer = "internal"
auto_update = true

[ui]
permission_mode = "ask"
yolo = false

[models]
default = "grok-4.6"

[mcp_servers.kimi-cu]
command = "/Applications/KimiCU.app/Contents/MacOS/kimi-cu"
enabled = true

[[marketplace.sources]]
name = "xAI Official"
git = "https://github.com/xai-org/plugin-marketplace.git"
`)
	if err := os.WriteFile(paths.ConfigFile(home), existing, 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := Apply(Options{
		Home:       home,
		Alias:      "proxy",
		Model:      "grok-4.6",
		BaseURL:    "https://gw.example.com/v1",
		EnvKey:     "GATEWAY_API_KEY",
		SetDefault: true,
		Search:     SearchEnable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.BackupPath == "" {
		t.Fatal("expected backup")
	}

	doc, err := cfg.Load(paths.ConfigFile(home))
	if err != nil {
		t.Fatal(err)
	}
	mcp, ok := cfg.GetTable(doc, "mcp_servers", "kimi-cu")
	if !ok || cfg.StringAt(mcp, "command") == "" {
		t.Fatalf("MCP section lost: %#v", doc["mcp_servers"])
	}
	if _, ok := cfg.GetTable(doc, "marketplace"); !ok {
		// array of tables lives under marketplace.sources
		if doc["marketplace"] == nil {
			t.Fatalf("marketplace lost:\n%v", mustEncode(t, doc))
		}
	}
	ui, _ := cfg.GetTable(doc, "ui")
	if cfg.StringAt(ui, "permission_mode") != "ask" {
		t.Fatalf("permission_mode should stay ask, got %#v", ui)
	}
	proxy, ok := cfg.GetTable(doc, "model", "proxy")
	if !ok {
		t.Fatal("model.proxy missing")
	}
	if cfg.StringAt(proxy, "env_key") != "GATEWAY_API_KEY" {
		t.Fatalf("env_key = %q", cfg.StringAt(proxy, "env_key"))
	}
	if _, exists := proxy["api_key"]; exists {
		t.Fatal("api_key should not be written when env_key is set")
	}
	if v, _ := proxy["supports_backend_search"].(bool); !v {
		t.Fatal("supports_backend_search not set")
	}
	models, _ := cfg.GetTable(doc, "models")
	if cfg.StringAt(models, "default") != "proxy" {
		t.Fatalf("default = %q", cfg.StringAt(models, "default"))
	}
	if cfg.StringAt(models, "web_search") != "proxy" {
		t.Fatalf("web_search = %q", cfg.StringAt(models, "web_search"))
	}
}

func TestApplyDoesNotWritePermissionModeByDefault(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	res, err := Apply(Options{
		Home:    home,
		Alias:   "proxy",
		BaseURL: "https://gw.example.com/v1",
		EnvKey:  "K",
	})
	if err != nil {
		t.Fatal(err)
	}
	ui, ok := cfg.GetTable(res.After, "ui")
	if ok {
		if _, exists := ui["permission_mode"]; exists {
			t.Fatalf("should not invent permission_mode: %#v", ui)
		}
	}
}

func TestApplyDryRunDoesNotWrite(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	_, err := Apply(Options{
		Home:    home,
		Alias:   "proxy",
		BaseURL: "https://gw.example.com/v1",
		EnvKey:  "K",
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.ConfigFile(home)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote a file")
	}
}

func TestApplyPrefersEnvKeyOverInline(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ConfigFile(home), []byte(`
[model.proxy]
api_key = "old-secret-should-go"
model = "old"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Apply(Options{
		Home:    home,
		Alias:   "proxy",
		BaseURL: "https://gw.example.com/v1",
		EnvKey:  "NEW_KEY",
	})
	if err != nil {
		t.Fatal(err)
	}
	proxy, _ := cfg.GetTable(res.After, "model", "proxy")
	if _, ok := proxy["api_key"]; ok {
		t.Fatalf("old api_key still present: %#v", proxy)
	}
}

func TestApplyRejectsBadURL(t *testing.T) {
	t.Parallel()
	_, err := Apply(Options{Alias: "proxy", BaseURL: "not-a-url", EnvKey: "K"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplySearchDisableOnlyTouchesOwnAlias(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.WriteFile(paths.ConfigFile(home), []byte(`
[models]
web_search = "other"

[model.other]
supports_backend_search = true
`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Apply(Options{
		Home:    home,
		Alias:   "proxy",
		BaseURL: "https://gw.example.com/v1",
		EnvKey:  "K",
		Search:  SearchDisable,
	})
	if err != nil {
		t.Fatal(err)
	}
	models, _ := cfg.GetTable(res.After, "models")
	if cfg.StringAt(models, "web_search") != "other" {
		t.Fatalf("should leave others' web_search, got %q", cfg.StringAt(models, "web_search"))
	}
}

func TestSummaryNeverContainsRawKey(t *testing.T) {
	t.Parallel()
	s := Summary(Options{
		Alias:   "proxy",
		APIKey:  "sk-SUPER-SECRET-VALUE-123456",
		BaseURL: "https://gw.example.com/v1",
	})
	if strings.Contains(s, "SUPER-SECRET") {
		t.Fatalf("summary leaked key:\n%s", s)
	}
}

func mustEncode(t *testing.T, doc cfg.Doc) string {
	t.Helper()
	b, err := cfg.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestWriteUses0600(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	_, err := Apply(Options{
		Home:    home,
		Alias:   "proxy",
		BaseURL: "https://gw.example.com/v1",
		EnvKey:  "K",
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected 0600-class perms, got %o", st.Mode().Perm())
	}
}
