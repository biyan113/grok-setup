package wizard

import (
	"fmt"
	"os"

	"github.com/biyan113/grok-setup/internal/apply"
	"github.com/biyan113/grok-setup/internal/cfg"
	"github.com/biyan113/grok-setup/internal/paths"
	"github.com/biyan113/grok-setup/internal/prompt"
	"github.com/biyan113/grok-setup/internal/validate"
)

func Run(home string, dryRun bool) error {
	io := prompt.New()
	defer io.Close()

	if home == "" {
		var err error
		home, err = paths.Home()
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(io.Err)
	fmt.Fprintln(io.Err, "Grok 配置向导")
	fmt.Fprintln(io.Err, "合并写入 ~/.grok/config.toml，不会整文件覆盖。")
	fmt.Fprintln(io.Err, "默认不改权限模式，也不会把密钥打到屏幕上。")
	fmt.Fprintln(io.Err)

	configPath := paths.ConfigFile(home)
	if _, err := cfg.Load(configPath); err != nil {
		return fmt.Errorf("现有配置无法解析，已中止: %w", err)
	}
	if _, err := os.Stat(configPath); err == nil {
		io.OK("已发现现有配置: %s", configPath)
		io.Info("只会增补/更新你指定的 [model.*]，MCP / marketplace / UI 会保留。")
	} else {
		io.Info("尚无配置文件，将新建: %s", configPath)
	}

	alias, err := io.Default("模型别名 (default / [model.名称]):", "proxy")
	if err != nil {
		return err
	}
	if err := validate.Alias(alias); err != nil {
		return err
	}

	var baseURL string
	for {
		baseURL, err = io.Line("base_url (例如 https://api.example.com/v1): ")
		if err != nil {
			return err
		}
		if err := validate.BaseURL(baseURL); err != nil {
			io.Warn("%v", err)
			continue
		}
		break
	}

	modelID, err := io.Default("上游模型 ID:", apply.DefaultUpstreamModel)
	if err != nil {
		return err
	}

	useEnv, err := io.Confirm("用环境变量保存密钥？（推荐，不写入磁盘）", true)
	if err != nil {
		return err
	}

	opt := apply.Options{
		Home:    home,
		Alias:   alias,
		Model:   modelID,
		BaseURL: baseURL,
		DryRun:  dryRun,
	}

	if useEnv {
		envKey, err := io.Default("环境变量名:", "GATEWAY_API_KEY")
		if err != nil {
			return err
		}
		if err := validate.EnvKeyName(envKey); err != nil {
			return err
		}
		opt.EnvKey = envKey
		if os.Getenv(envKey) == "" {
			io.Warn("当前进程里 %s 还是空的。写入配置后，请在 shell 里 export 再启动 grok。", envKey)
		} else {
			io.OK("已检测到 %s", envKey)
		}
	} else {
		key, err := io.Secret("api_key（输入不可见）: ")
		if err != nil {
			return err
		}
		if key == "" {
			return fmt.Errorf("api_key 不能为空")
		}
		again, err := io.Confirm("再输入一次以核对？", true)
		if err != nil {
			return err
		}
		if again {
			key2, err := io.Secret("再次输入 api_key: ")
			if err != nil {
				return err
			}
			if key != key2 {
				return fmt.Errorf("两次输入不一致，已取消")
			}
		}
		opt.APIKey = key
	}

	setDefault, err := io.Confirm("把这个别名设成 [models].default？", false)
	if err != nil {
		return err
	}
	opt.SetDefault = setDefault

	enableSearch, err := io.Confirm("开启原生 Search（web_search + supports_backend_search）？网关需支持服务端搜索", false)
	if err != nil {
		return err
	}
	if enableSearch {
		opt.Search = apply.SearchEnable
		backend, err := io.Default("api_backend（可选，官方搜索不强制 responses）:", "")
		if err != nil {
			return err
		}
		if backend != "" {
			if err := validate.APIBackend(backend); err != nil {
				return err
			}
			opt.APIBackend = backend
		}
	}

	privacy, err := io.Confirm("在缺失时补上关遥测 / 禁止代码库上传？（不覆盖已有值）", true)
	if err != nil {
		return err
	}
	opt.Privacy = privacy

	fmt.Fprintln(io.Err)
	io.Info("即将合并写入:")
	fmt.Fprint(io.Err, apply.Summary(opt))
	if dryRun {
		io.Info("dry-run：不会写盘")
	}

	ok, err := io.Confirm("确认写入？", false)
	if err != nil {
		return err
	}
	if !ok {
		io.Warn("已取消，未做任何修改。")
		return nil
	}

	res, err := apply.Apply(opt)
	if err != nil {
		return err
	}
	if res.BackupPath != "" {
		io.OK("已备份 → %s", res.BackupPath)
	}
	if res.Wrote {
		io.OK("已合并写入 %s", res.ConfigPath)
	}

	redacted, err := cfg.Encode(cfg.Redact(res.After))
	if err == nil {
		fmt.Fprintln(io.Err)
		io.Info("写入后预览（敏感字段已掩码）:")
		fmt.Fprintln(io.Err, "────────────────────────────────────────")
		fmt.Fprint(io.Err, string(redacted))
		fmt.Fprintln(io.Err, "────────────────────────────────────────")
	}
	io.Info("官方 CLI 仍用命令 grok。本工具命令是 gsetup，不会覆盖它。")
	io.Info("装官方 CLI: gsetup install-cli")
	io.Info("装 grok-search skill: gsetup search")
	return nil
}
