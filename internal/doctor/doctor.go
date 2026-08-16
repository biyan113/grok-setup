package doctor

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/biyan113/grok-setup/internal/cfg"
	"github.com/biyan113/grok-setup/internal/paths"
	"github.com/biyan113/grok-setup/internal/validate"
)

func Run(home string, out io.Writer) error {
	if home == "" {
		var err error
		home, err = paths.Home()
		if err != nil {
			return err
		}
	}
	configPath := paths.ConfigFile(home)
	fmt.Fprintf(out, "GROK_HOME     %s\n", home)
	fmt.Fprintf(out, "config.toml   %s\n", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintln(out, "状态          配置文件不存在（官方 grok 仍可用内置默认）")
	} else {
		st, err := os.Stat(configPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "权限          %04o\n", st.Mode().Perm())
		if st.Mode().Perm()&0o077 != 0 {
			fmt.Fprintln(out, "警告          配置文件对组/其他人可读，建议 chmod 600")
		}
		doc, err := cfg.Load(configPath)
		if err != nil {
			fmt.Fprintf(out, "解析          失败: %v\n", err)
			return err
		}
		fmt.Fprintln(out, "解析          成功")
		if models, ok := cfg.GetTable(doc, "models"); ok {
			fmt.Fprintf(out, "default       %s\n", cfg.StringAt(models, "default"))
			if ws := cfg.StringAt(models, "web_search"); ws != "" {
				fmt.Fprintf(out, "web_search    %s\n", ws)
			}
		}
		if ui, ok := cfg.GetTable(doc, "ui"); ok {
			if pm := cfg.StringAt(ui, "permission_mode"); pm != "" {
				fmt.Fprintf(out, "permission    %s\n", pm)
				if pm == "always-approve" || pm == "bypassPermissions" {
					fmt.Fprintln(out, "警告          always-approve 会让代理少打断地改文件/跑命令")
				}
			}
		}
		for _, name := range cfg.ModelNames(doc) {
			m, _ := cfg.GetTable(doc, "model", name)
			fmt.Fprintf(out, "\n[model.%s]\n", name)
			fmt.Fprintf(out, "  model     %s\n", cfg.StringAt(m, "model"))
			if u := cfg.StringAt(m, "base_url"); u != "" {
				if err := validate.BaseURL(u); err != nil {
					fmt.Fprintf(out, "  base_url  %s  (%v)\n", u, err)
				} else {
					fmt.Fprintf(out, "  base_url  %s\n", u)
				}
			}
			if k := cfg.StringAt(m, "env_key"); k != "" {
				if os.Getenv(k) == "" {
					fmt.Fprintf(out, "  env_key   %s  （当前进程未设置）\n", k)
				} else {
					fmt.Fprintf(out, "  env_key   %s  （已设置）\n", k)
				}
			}
			if _, ok := m["api_key"]; ok {
				fmt.Fprintln(out, "  api_key   已写入文件（建议改成 env_key）")
			}
			if v, ok := m["supports_backend_search"].(bool); ok && v {
				fmt.Fprintln(out, "  search    supports_backend_search = true")
			}
		}
	}

	fmt.Fprintln(out)
	if p, err := exec.LookPath("grok"); err == nil {
		fmt.Fprintf(out, "官方 grok     %s\n", p)
	} else if _, err := os.Stat(paths.OfficialBin(home)); err == nil {
		fmt.Fprintf(out, "官方 grok     %s（不在 PATH）\n", paths.OfficialBin(home))
	} else {
		fmt.Fprintln(out, "官方 grok     未找到。可执行: gsetup install-cli")
	}
	return nil
}
