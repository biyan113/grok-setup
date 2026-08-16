package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/biyan113/grok-setup/internal/apply"
	"github.com/biyan113/grok-setup/internal/backup"
	"github.com/biyan113/grok-setup/internal/cfg"
	"github.com/biyan113/grok-setup/internal/doctor"
	"github.com/biyan113/grok-setup/internal/installcli"
	"github.com/biyan113/grok-setup/internal/paths"
	"github.com/biyan113/grok-setup/internal/searchskill"
	"github.com/biyan113/grok-setup/internal/wizard"
)

const Version = "0.1.1"

const usage = `Grok（命令名 gsetup）— 给官方 Grok Build 合并写入自定义模型，而不是整文件覆盖。

用法:
  gsetup                  交互向导（默认）
  gsetup init             同上
  gsetup apply [选项]     非交互写入
  gsetup show             脱敏打印当前 ~/.grok/config.toml
  gsetup doctor           检查配置、权限、env_key
  gsetup backups          列出备份
  gsetup restore <文件>   从备份恢复 config.toml
  gsetup install-cli      下载并执行官方 x.ai 安装脚本
  gsetup search           可选安装 Autsunset/grok-search skill
  gsetup version

apply 选项:
  --home DIR              覆盖 GROK_HOME（默认 ~/.grok）
  --alias NAME            本地配置名（默认 proxy）
  --model ID              上游模型 ID（默认 grok-4.6）
  --name TEXT             展示名
  --base-url URL          OpenAI 兼容根地址（必填）
  --env-key NAME          从环境变量读密钥（推荐）
  --api-key-stdin         从 stdin 读 api_key（不回显）
  --set-default           同时改 [models].default
  --search                开启原生 web_search
  --no-search             关闭本别名上的原生搜索
  --api-backend NAME      chat_completions | responses | messages
  --context-window N      写入 context_window
  --permission-mode MODE  仅在你显式指定时才改 [ui]
  --privacy               缺失时补关遥测 / 禁止代码库上传
  --dry-run               只打印将写入的脱敏结果
  --yes                   非交互，不再确认

本工具不会覆盖官方 grok 命令。
`

func Main(args []string) int {
	if len(args) == 0 {
		args = []string{"init"}
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	case "version", "-v", "--version":
		fmt.Println(Version)
		return 0
	case "init", "wizard":
		return wrap(runInit(rest))
	case "apply":
		return wrap(runApply(rest))
	case "show":
		return wrap(runShow(rest))
	case "doctor":
		return wrap(runDoctor(rest))
	case "backups":
		return wrap(runBackups(rest))
	case "restore":
		return wrap(runRestore(rest))
	case "install-cli":
		return wrap(installcli.Run(os.Stdout))
	case "search":
		return wrap(runSearch(rest))
	default:
		if strings.HasPrefix(cmd, "-") {
			return wrap(fmt.Errorf("未知选项 %s\n\n%s", cmd, usage))
		}
		return wrap(fmt.Errorf("未知命令 %s\n\n%s", cmd, usage))
	}
}

func wrap(err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintf(os.Stderr, "✗  %v\n", err)
	return 1
}

func homeFlag(fs *flag.FlagSet) *string {
	return fs.String("home", "", "GROK_HOME 覆盖")
}

func resolveHome(v string) (string, error) {
	if v != "" {
		return v, nil
	}
	return paths.Home()
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	home := homeFlag(fs)
	dry := fs.Bool("dry-run", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return wizard.Run(*home, *dry)
}

func runApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	opt := apply.Options{}
	home := homeFlag(fs)
	alias := fs.String("alias", "proxy", "")
	model := fs.String("model", apply.DefaultUpstreamModel, "")
	name := fs.String("name", "", "")
	baseURL := fs.String("base-url", "", "")
	envKey := fs.String("env-key", "", "")
	apiKeyStdin := fs.Bool("api-key-stdin", false, "")
	setDefault := fs.Bool("set-default", false, "")
	search := fs.Bool("search", false, "")
	noSearch := fs.Bool("no-search", false, "")
	backend := fs.String("api-backend", "", "")
	ctxWin := fs.String("context-window", "", "")
	perm := fs.String("permission-mode", "", "")
	privacy := fs.Bool("privacy", false, "")
	dry := fs.Bool("dry-run", false, "")
	yes := fs.Bool("yes", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opt.Home = *home
	opt.Alias = *alias
	opt.Model = *model
	opt.Name = *name
	opt.BaseURL = *baseURL
	opt.EnvKey = *envKey
	opt.SetDefault = *setDefault
	opt.APIBackend = *backend
	opt.PermissionMode = *perm
	opt.Privacy = *privacy
	opt.DryRun = *dry
	if *search && *noSearch {
		return fmt.Errorf("--search 与 --no-search 不能同时用")
	}
	switch {
	case *search:
		opt.Search = apply.SearchEnable
	case *noSearch:
		opt.Search = apply.SearchDisable
	}
	if *ctxWin != "" {
		n, err := strconv.ParseInt(*ctxWin, 10, 64)
		if err != nil || n <= 0 {
			return fmt.Errorf("无效 --context-window")
		}
		opt.ContextWindow = n
	}
	if *apiKeyStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		opt.APIKey = strings.TrimSpace(string(data))
	}

	if !*yes && !*dry {
		fmt.Fprintln(os.Stderr, "即将合并写入:")
		fmt.Fprint(os.Stderr, apply.Summary(opt))
		fmt.Fprintln(os.Stderr, "非交互请加 --yes。已中止。")
		return fmt.Errorf("缺少 --yes")
	}

	res, err := apply.Apply(opt)
	if err != nil {
		return err
	}
	if res.BackupPath != "" {
		fmt.Fprintf(os.Stderr, "✓  备份 %s\n", res.BackupPath)
	}
	redacted, err := cfg.Encode(cfg.Redact(res.After))
	if err != nil {
		return err
	}
	if opt.DryRun {
		fmt.Fprintln(os.Stderr, "dry-run 结果（未写盘）:")
	} else {
		fmt.Fprintf(os.Stderr, "✓  已写入 %s\n", res.ConfigPath)
	}
	os.Stdout.Write(redacted)
	return nil
}

func runShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	homeFlag := homeFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	home, err := resolveHome(*homeFlag)
	if err != nil {
		return err
	}
	path := paths.ConfigFile(home)
	doc, err := cfg.Load(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("没有配置文件: %s", path)
	}
	out, err := cfg.Encode(cfg.Redact(doc))
	if err != nil {
		return err
	}
	os.Stdout.Write(out)
	return nil
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	homeFlag := homeFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	home, err := resolveHome(*homeFlag)
	if err != nil {
		return err
	}
	return doctor.Run(home, os.Stdout)
}

func runBackups(args []string) error {
	fs := flag.NewFlagSet("backups", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	homeFlag := homeFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	home, err := resolveHome(*homeFlag)
	if err != nil {
		return err
	}
	ents, err := backup.List(home)
	if err != nil {
		return err
	}
	if len(ents) == 0 {
		fmt.Println("没有备份。")
		return nil
	}
	for _, e := range ents {
		fmt.Printf("%s\t%d\t%s\n", e.ModTime.Format("2006-01-02 15:04:05"), e.Size, e.Path)
	}
	return nil
}

func runRestore(args []string) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	homeFlag := homeFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("用法: gsetup restore <备份文件>")
	}
	home, err := resolveHome(*homeFlag)
	if err != nil {
		return err
	}
	src := fs.Arg(0)
	if !filepath.IsAbs(src) {
		cand := filepath.Join(paths.BackupDir(home), src)
		if _, err := os.Stat(cand); err == nil {
			src = cand
		}
	}
	// backup current first
	cur := paths.ConfigFile(home)
	if bak, err := backup.CopyFile(home, cur, "config.toml"); err != nil {
		return err
	} else if bak != "" {
		fmt.Fprintf(os.Stderr, "✓  恢复前已备份当前文件 → %s\n", bak)
	}
	if err := backup.Restore(src, cur); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "✓  已恢复 → %s\n", cur)
	return nil
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	homeFlag := homeFlag(fs)
	repo := fs.String("repo", searchskill.DefaultRepo, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return searchskill.Run(*homeFlag, *repo)
}
