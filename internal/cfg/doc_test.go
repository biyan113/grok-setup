package cfg

import (
	"strings"
	"testing"
)

func TestParseRoundTripPreservesExtraSections(t *testing.T) {
	t.Parallel()
	raw := []byte(`
[cli]
installer = "internal"

[ui]
permission_mode = "ask"
yolo = false

[models]
default = "grok-4.6"

[mcp_servers.kimi-cu]
command = "/usr/bin/kimi"
enabled = true

[[marketplace.sources]]
name = "xAI Official"
git = "https://github.com/xai-org/plugin-marketplace.git"
`)
	doc, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := GetTable(doc, "mcp_servers", "kimi-cu"); !ok {
		t.Fatal("mcp_servers.kimi-cu missing")
	}
	out, err := Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	mcp, ok := GetTable(again, "mcp_servers", "kimi-cu")
	if !ok {
		t.Fatalf("lost mcp after encode:\n%s", out)
	}
	if StringAt(mcp, "command") != "/usr/bin/kimi" {
		t.Fatalf("command = %q", StringAt(mcp, "command"))
	}
}

func TestMergeDoesNotClobberSiblingKeys(t *testing.T) {
	t.Parallel()
	dst, err := Parse([]byte(`
[ui]
permission_mode = "ask"
max_thoughts_width = 120

[model.old]
model = "keep-me"
`))
	if err != nil {
		t.Fatal(err)
	}
	src := Empty()
	Table(src, "model", "proxy")["base_url"] = "https://gw.example/v1"
	merged := Merge(dst, src)

	ui, _ := GetTable(merged, "ui")
	if StringAt(ui, "permission_mode") != "ask" {
		t.Fatalf("permission_mode overwritten: %#v", ui)
	}
	if _, ok := GetTable(merged, "model", "old"); !ok {
		t.Fatal("existing model.old dropped")
	}
	proxy, ok := GetTable(merged, "model", "proxy")
	if !ok || StringAt(proxy, "base_url") != "https://gw.example/v1" {
		t.Fatalf("model.proxy missing: %#v", merged["model"])
	}
}

func TestRedactMasksAPIKey(t *testing.T) {
	t.Parallel()
	doc := Empty()
	Table(doc, "model", "proxy")["api_key"] = "sk-abcdefghijklmnopqrstuv"
	red := Redact(doc)
	proxy, _ := GetTable(red, "model", "proxy")
	got := StringAt(proxy, "api_key")
	if strings.Contains(got, "sk-abcdefghijklmnop") {
		t.Fatalf("api_key not masked: %s", got)
	}
	if !strings.Contains(got, "****") {
		t.Fatalf("expected mask, got %s", got)
	}
}

func TestRedactKeepsBaseURL(t *testing.T) {
	t.Parallel()
	doc := Empty()
	Table(doc, "model", "proxy")["base_url"] = "https://gw.example.com/v1"
	red := Redact(doc)
	proxy, _ := GetTable(red, "model", "proxy")
	if StringAt(proxy, "base_url") != "https://gw.example.com/v1" {
		t.Fatalf("base_url should stay visible, got %q", StringAt(proxy, "base_url"))
	}
}
