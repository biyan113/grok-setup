package searchskill

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/biyan113/grok-setup/internal/backup"
	"github.com/biyan113/grok-setup/internal/paths"
	"github.com/biyan113/grok-setup/internal/prompt"
	"github.com/biyan113/grok-setup/internal/secret"
	"github.com/biyan113/grok-setup/internal/validate"
)

const DefaultRepo = "https://github.com/Autsunset/grok-search.git"

type fileConfig struct {
	APIURL                   string `json:"apiUrl"`
	APIKey                   string `json:"apiKey"`
	APIProvider              string `json:"apiProvider"`
	SearchEndpoint           string `json:"searchEndpoint"`
	Model                    string `json:"model"`
	ResponsesMaxTurns        int    `json:"responsesMaxTurns,omitempty"`
	ResponsesReasoningEffort string `json:"responsesReasoningEffort,omitempty"`
	DefaultExtra             int    `json:"defaultExtra"`
	SourceChars              int    `json:"sourceChars"`
}

func inferProvider(apiURL string) string {
	u := strings.ToLower(apiURL)
	switch {
	case strings.Contains(u, "openrouter"):
		return "openrouter"
	case strings.Contains(u, "api.x.ai"):
		return "xai"
	default:
		return "openai-compatible"
	}
}

func Run(home, repo string) error {
	io := prompt.New()
	defer io.Close()

	if home == "" {
		var err error
		home, err = paths.Home()
		if err != nil {
			return err
		}
	}
	if repo == "" {
		repo = DefaultRepo
	}

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("需要 git")
	}
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("需要 Node.js >= 18.17")
	}

	dest := paths.SearchSkillDir(home)
	io.Info("skill 目录: %s", dest)
	io.Info("上游仓库: %s", repo)

	if err := cloneOrPull(io, dest, repo); err != nil {
		return err
	}

	cfgPath, err := paths.SearchConfigFile()
	if err != nil {
		return err
	}

	baseURL, err := io.Line("apiUrl (OpenAI 兼容根地址，不要带 /chat/completions): ")
	if err != nil {
		return err
	}
	if err := validate.BaseURL(baseURL); err != nil {
		return err
	}
	key, err := io.Secret("apiKey（输入不可见）: ")
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("apiKey 不能为空")
	}
	endpoint, err := io.Default("searchEndpoint [chat/responses]:", "chat")
	if err != nil {
		return err
	}
	endpoint = strings.ToLower(endpoint)
	if endpoint != "chat" && endpoint != "responses" {
		return fmt.Errorf("searchEndpoint 只能是 chat 或 responses")
	}
	defaultModel := "grok-4.3-fast"
	if endpoint == "responses" {
		defaultModel = "grok-4.6"
	}
	model, err := io.Default("搜索模型 ID:", defaultModel)
	if err != nil {
		return err
	}

	io.Info("将写入 %s", cfgPath)
	io.Info("apiUrl=%s  apiKey=%s  endpoint=%s  model=%s", baseURL, secret.Mask(key), endpoint, model)
	ok, err := io.Confirm("确认写入？（已有文件会先备份）", false)
	if err != nil {
		return err
	}
	if !ok {
		io.Warn("已取消。")
		return nil
	}

	if _, err := os.Stat(cfgPath); err == nil {
		bak, err := backup.CopyFile(home, cfgPath, "grok-search.json")
		if err != nil {
			return fmt.Errorf("备份 grok-search 配置: %w", err)
		}
		if bak != "" {
			io.OK("已备份 → %s", bak)
		}
	}

	fc := fileConfig{
		APIURL:         baseURL,
		APIKey:         key,
		APIProvider:    inferProvider(baseURL),
		SearchEndpoint: endpoint,
		Model:          model,
		DefaultExtra:   6,
		SourceChars:    400,
	}
	if endpoint == "responses" {
		fc.ResponsesMaxTurns = 3
		fc.ResponsesReasoningEffort = "low"
	}

	if err := writeJSON(cfgPath, fc); err != nil {
		return err
	}
	io.OK("已写入 %s", cfgPath)
	io.Info("Grok 会从 ~/.grok/skills/ 自动发现 skill。")
	return nil
}

func cloneOrPull(io *prompt.IO, dest, repo string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		io.Info("已有仓库，git pull --ff-only")
		cmd := exec.Command("git", "-C", dest, "pull", "--ff-only")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			io.Warn("git pull 失败，继续使用现有目录")
		}
	} else if _, err := os.Stat(dest); err == nil {
		if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
			return fmt.Errorf("目录已存在但不是 grok-search: %s", dest)
		}
		io.OK("使用现有非 git 目录")
	} else {
		io.Info("克隆仓库…")
		cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
	}

	if _, err := os.Stat(filepath.Join(dest, "node_modules", "undici")); os.IsNotExist(err) {
		if _, err := exec.LookPath("npm"); err != nil {
			return fmt.Errorf("需要 npm 安装 undici")
		}
		io.Info("npm install --no-fund --no-audit")
		cmd := exec.Command("npm", "install", "--no-fund", "--no-audit")
		cmd.Dir = dest
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm install: %w", err)
		}
	}
	return nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config.json.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	_ = tmp.Chmod(0o600)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
