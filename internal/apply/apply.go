package apply

import (
	"fmt"
	"os"
	"strings"

	"github.com/biyan113/grok-setup/internal/backup"
	"github.com/biyan113/grok-setup/internal/cfg"
	"github.com/biyan113/grok-setup/internal/paths"
	"github.com/biyan113/grok-setup/internal/secret"
	"github.com/biyan113/grok-setup/internal/validate"
)

const DefaultUpstreamModel = "grok-4.6"

type SearchMode int

const (
	SearchLeave SearchMode = iota
	SearchEnable
	SearchDisable
)

type Options struct {
	Home           string
	Alias          string
	Model          string
	Name           string
	BaseURL        string
	EnvKey         string
	APIKey         string
	SetDefault     bool
	Search         SearchMode
	APIBackend     string
	ContextWindow  int64
	PermissionMode string
	Privacy        bool
	DryRun         bool
}

type Result struct {
	Home        string
	ConfigPath  string
	BackupPath  string
	Before      cfg.Doc
	After       cfg.Doc
	Wrote       bool
	CreatedFile bool
}

func Apply(opt Options) (*Result, error) {
	if err := validateOptions(opt); err != nil {
		return nil, err
	}
	home := opt.Home
	if home == "" {
		var err error
		home, err = paths.Home()
		if err != nil {
			return nil, err
		}
	}
	configPath := paths.ConfigFile(home)
	before, err := cfg.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取现有配置失败（已拒绝覆盖，以免毁掉文件）: %w", err)
	}
	created := false
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		created = true
	}

	after := patch(before, opt)
	res := &Result{
		Home:        home,
		ConfigPath:  configPath,
		Before:      before,
		After:       after,
		CreatedFile: created,
	}

	encoded, err := cfg.Encode(after)
	if err != nil {
		return nil, err
	}
	if opt.DryRun {
		return res, nil
	}

	bak, err := backup.CopyFile(home, configPath, "config.toml")
	if err != nil {
		return nil, fmt.Errorf("备份失败: %w", err)
	}
	res.BackupPath = bak
	if err := cfg.WriteAtomic(configPath, encoded); err != nil {
		return nil, fmt.Errorf("写入配置: %w", err)
	}
	res.Wrote = true
	return res, nil
}

func validateOptions(opt Options) error {
	if err := validate.Alias(opt.Alias); err != nil {
		return err
	}
	if err := validate.BaseURL(opt.BaseURL); err != nil {
		return err
	}
	if err := validate.APIBackend(opt.APIBackend); err != nil {
		return err
	}
	if err := validate.PermissionMode(opt.PermissionMode); err != nil {
		return err
	}
	if strings.TrimSpace(opt.EnvKey) == "" && strings.TrimSpace(opt.APIKey) == "" {
		return fmt.Errorf("必须提供 --env-key 或 API Key（推荐环境变量，避免把密钥写入磁盘）")
	}
	if opt.EnvKey != "" {
		if err := validate.EnvKeyName(opt.EnvKey); err != nil {
			return err
		}
	}
	return nil
}

func patch(before cfg.Doc, opt Options) cfg.Doc {
	after := cfg.Clone(before)

	modelID := strings.TrimSpace(opt.Model)
	if modelID == "" {
		modelID = DefaultUpstreamModel
	}
	display := strings.TrimSpace(opt.Name)
	if display == "" {
		display = modelID
	}

	model := cfg.Table(after, "model", opt.Alias)
	model["model"] = modelID
	model["name"] = display
	model["base_url"] = strings.TrimSpace(opt.BaseURL)

	if opt.EnvKey != "" {
		model["env_key"] = opt.EnvKey
		delete(model, "api_key")
	} else {
		model["api_key"] = opt.APIKey
		delete(model, "env_key")
	}

	if opt.APIBackend != "" {
		model["api_backend"] = opt.APIBackend
	}
	if opt.ContextWindow > 0 {
		model["context_window"] = opt.ContextWindow
	}

	models := cfg.Table(after, "models")
	if opt.SetDefault {
		models["default"] = opt.Alias
	}

	switch opt.Search {
	case SearchEnable:
		models["web_search"] = opt.Alias
		model["supports_backend_search"] = true
	case SearchDisable:
		if cfg.StringAt(models, "web_search") == opt.Alias {
			delete(models, "web_search")
		}
		delete(model, "supports_backend_search")
	}

	if opt.PermissionMode != "" {
		cfg.Table(after, "ui")["permission_mode"] = opt.PermissionMode
	}

	if opt.Privacy {
		features := cfg.Table(after, "features")
		if _, ok := features["telemetry"]; !ok {
			features["telemetry"] = false
		}
		telemetry := cfg.Table(after, "telemetry")
		if _, ok := telemetry["trace_upload"]; !ok {
			telemetry["trace_upload"] = false
		}
		if _, ok := telemetry["mixpanel_enabled"]; !ok {
			telemetry["mixpanel_enabled"] = false
		}
		harness := cfg.Table(after, "harness")
		if _, ok := harness["disable_codebase_upload"]; !ok {
			harness["disable_codebase_upload"] = true
		}
	}

	return after
}

func Summary(opt Options) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  别名        %s  →  [model.%s]\n", opt.Alias, opt.Alias)
	modelID := opt.Model
	if modelID == "" {
		modelID = DefaultUpstreamModel
	}
	fmt.Fprintf(&b, "  上游模型    %s\n", modelID)
	fmt.Fprintf(&b, "  base_url    %s\n", opt.BaseURL)
	if opt.EnvKey != "" {
		fmt.Fprintf(&b, "  凭据        env_key = %s\n", opt.EnvKey)
	} else {
		fmt.Fprintf(&b, "  凭据        api_key = %s\n", secret.Mask(opt.APIKey))
	}
	if opt.APIBackend != "" {
		fmt.Fprintf(&b, "  api_backend %s\n", opt.APIBackend)
	}
	switch opt.Search {
	case SearchEnable:
		fmt.Fprintf(&b, "  原生搜索    开启（web_search + supports_backend_search）\n")
	case SearchDisable:
		fmt.Fprintf(&b, "  原生搜索    关闭\n")
	default:
		fmt.Fprintf(&b, "  原生搜索    保持现状\n")
	}
	if opt.SetDefault {
		fmt.Fprintf(&b, "  设为默认    是\n")
	} else {
		fmt.Fprintf(&b, "  设为默认    否（不改 [models].default）\n")
	}
	if opt.PermissionMode != "" {
		fmt.Fprintf(&b, "  权限模式    %s\n", opt.PermissionMode)
	} else {
		fmt.Fprintf(&b, "  权限模式    不改动（不会写入 always-approve）\n")
	}
	return b.String()
}
