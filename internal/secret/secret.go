package secret

import (
	"regexp"
	"strings"
	"unicode"
)

var sensitiveKey = regexp.MustCompile(`(?i)^(api[_-]?key|secret|token|password|passwd|authorization|auth[_-]?token|access[_-]?key|access[_-]?token)$`)

func IsSensitiveKey(key string) bool {
	return sensitiveKey.MatchString(strings.TrimSpace(key))
}

// Mask keeps a short prefix/suffix so the user can verify they typed something,
// without printing the credential.
func Mask(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(空)"
	}
	n := 0
	for range s {
		n++
	}
	if n <= 8 {
		return "****"
	}
	runes := []rune(s)
	return string(runes[:4]) + "****" + string(runes[len(runes)-4:])
}

func LooksLikeSecret(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 12 {
		return false
	}
	if strings.HasPrefix(s, "sk-") || strings.HasPrefix(s, "xai-") || strings.HasPrefix(s, "gsk_") {
		return true
	}
	letters, digits := 0, 0
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsDigit(r):
			digits++
		}
	}
	return letters > 0 && digits > 0 && len(s) >= 20
}
