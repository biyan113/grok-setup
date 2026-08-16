package installcli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/biyan113/grok-setup/internal/paths"
	"github.com/biyan113/grok-setup/internal/prompt"
)

const OfficialURL = "https://x.ai/cli/install.sh"

func Run(out io.Writer) error {
	ui := prompt.New()
	defer ui.Close()

	ui.Info("将下载官方安装脚本: %s", OfficialURL)
	ui.Warn("这会执行来自 x.ai 的脚本，把 grok 装到 ~/.grok/bin。")
	ok, err := ui.Confirm("继续下载并执行？", false)
	if err != nil {
		return err
	}
	if !ok {
		ui.Warn("已取消。")
		return nil
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(OfficialURL)
	if err != nil {
		return fmt.Errorf("下载安装脚本: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载安装脚本: HTTP %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "grok-official-install-*.sh")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	n, err := ioCopy(tmp, resp.Body)
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o700); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	ui.OK("已下载 %d 字节 → %s", n, tmpName)
	ui.Info("可用 less %s 先看再重跑。现在执行…", tmpName)

	cmd := exec.Command("bash", tmpName)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("官方安装脚本失败: %w", err)
	}

	home, err := paths.Home()
	if err != nil {
		return err
	}
	bin := paths.OfficialBin(home)
	if _, err := os.Stat(bin); err == nil {
		ui.OK("官方 grok: %s", bin)
		ui.Info("若当前 shell 找不到命令: export PATH=%q:$PATH", filepath.Dir(bin))
	}
	return nil
}

func ioCopy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}
