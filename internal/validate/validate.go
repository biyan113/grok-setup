package validate

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var aliasRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func Alias(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("模型别名不能为空")
	}
	if !aliasRE.MatchString(name) {
		return fmt.Errorf("模型别名 %q 无效：须字母开头，后接字母/数字/_/-", name)
	}
	return nil
}

func BaseURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("base_url 不能为空")
	}
	if strings.ContainsAny(raw, " \t\n\r") {
		return fmt.Errorf("base_url 不能含空白")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("base_url 无法解析: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base_url 须以 http:// 或 https:// 开头")
	}
	if u.Host == "" {
		return fmt.Errorf("base_url 缺少主机名")
	}
	return nil
}

func APIBackend(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	switch v {
	case "chat_completions", "responses", "messages":
		return nil
	default:
		return fmt.Errorf("api_backend 只能是 chat_completions / responses / messages，收到 %q", v)
	}
}

func PermissionMode(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	switch v {
	case "ask", "default", "auto", "acceptEdits", "plan", "dontAsk",
		"always-approve", "bypassPermissions":
		return nil
	default:
		return fmt.Errorf("未知 permission_mode %q", v)
	}
}

func EnvKeyName(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return fmt.Errorf("环境变量名不能为空")
	}
	for i, r := range v {
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_'
		if i > 0 {
			ok = ok || (r >= '0' && r <= '9')
		}
		if !ok {
			return fmt.Errorf("环境变量名 %q 无效", v)
		}
	}
	return nil
}
