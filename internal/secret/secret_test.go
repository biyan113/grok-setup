package secret

import "testing"

func TestMask(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", "(空)"},
		{"abcd", "****"},
		{"sk-abcdefghijklmnop", "sk-a****mnop"},
	}
	for _, tc := range cases {
		if got := Mask(tc.in); got != tc.want {
			t.Fatalf("Mask(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsSensitiveKey(t *testing.T) {
	t.Parallel()
	yes := []string{"api_key", "API-KEY", "token", "password"}
	no := []string{"model", "default", "name", "alias", "base_url"}
	for _, k := range yes {
		if !IsSensitiveKey(k) {
			t.Fatalf("expected sensitive: %s", k)
		}
	}
	for _, k := range no {
		if IsSensitiveKey(k) {
			t.Fatalf("expected not sensitive: %s", k)
		}
	}
}

func TestLooksLikeSecret(t *testing.T) {
	t.Parallel()
	if !LooksLikeSecret("sk-abcdefghijklmnopqrstuv") {
		t.Fatal("sk- prefix should look like a secret")
	}
	if LooksLikeSecret("grok-4.6") {
		t.Fatal("model id should not look like a secret")
	}
}
