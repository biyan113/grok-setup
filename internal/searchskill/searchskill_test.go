package searchskill

import "testing"

func TestInferProvider(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"https://openrouter.ai/api/v1": "openrouter",
		"https://api.x.ai/v1":          "xai",
		"https://gw.example.com/v1":    "openai-compatible",
	}
	for u, want := range cases {
		if got := inferProvider(u); got != want {
			t.Fatalf("inferProvider(%q)=%q want %q", u, got, want)
		}
	}
}
