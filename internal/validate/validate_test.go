package validate

import "testing"

func TestAlias(t *testing.T) {
	t.Parallel()
	ok := []string{"proxy", "Steve", "work-grok", "MyProxy"}
	bad := []string{"", "1abc", "has space", "中文", "-dash"}
	for _, v := range ok {
		if err := Alias(v); err != nil {
			t.Fatalf("Alias(%q) unexpected error: %v", v, err)
		}
	}
	for _, v := range bad {
		if err := Alias(v); err == nil {
			t.Fatalf("Alias(%q) expected error", v)
		}
	}
}

func TestBaseURL(t *testing.T) {
	t.Parallel()
	if err := BaseURL("https://api.example.com/v1"); err != nil {
		t.Fatal(err)
	}
	if err := BaseURL("http://127.0.0.1:8080/v1"); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"", "ftp://x", "https://", "https://host /v1", "example.com/v1"} {
		if err := BaseURL(v); err == nil {
			t.Fatalf("BaseURL(%q) expected error", v)
		}
	}
}

func TestAPIBackend(t *testing.T) {
	t.Parallel()
	if err := APIBackend(""); err != nil {
		t.Fatal(err)
	}
	if err := APIBackend("responses"); err != nil {
		t.Fatal(err)
	}
	if err := APIBackend("rpc"); err == nil {
		t.Fatal("expected error")
	}
}
