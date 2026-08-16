# grok-setup

给 [Grok Build](https://x.ai) 接自定义 OpenAI 兼容网关的配置工具。

仓库名 **`grok-setup`**，命令名 **`gsetup`**，不会覆盖官方 `grok`。  
只**合并写入** `~/.grok/config.toml`：已有 MCP、marketplace、UI 会保留。密钥默认走环境变量，不默认 `always-approve`。

本仓库 **不含** 官方 CLI 本体。

[安装](#安装) · [快速开始](#快速开始) · [命令](#命令) · [apply 选项](#apply-选项) · [配置规则](#配置规则) · [排错](#排错)

## 安装

分两步：先装官方 `grok`，再装本仓库的 `gsetup`。

### 1. 官方 Grok Build CLI

文档：[Getting Started](https://docs.x.ai/build) · 安装脚本来自 [x.ai](https://x.ai)。

macOS / Linux / Windows Git Bash：

```bash
curl -fsSL https://x.ai/cli/install.sh | bash
```

指定版本：

```bash
curl -fsSL https://x.ai/cli/install.sh | bash -s 0.1.42
```

Windows PowerShell：

```powershell
irm https://x.ai/cli/install.ps1 | iex
```

指定版本：

```powershell
$env:GROK_VERSION="0.1.42"; irm https://x.ai/cli/install.ps1 | iex
```

校验、升级、启动：

```bash
export PATH="$HOME/.grok/bin:$PATH"   # 当前 shell 找不到 grok 时
grok --version
grok update
grok
```

首次启动会打开浏览器登录 grok.com。CI / 无浏览器环境可改用密钥：

```bash
export XAI_API_KEY="xai-..."
grok
```

也可以用本工具代跑官方脚本（会先确认，再下载到临时文件后执行）：

```bash
gsetup install-cli
```

### 2. gsetup

需要 **Go 1.22+**（[go.dev/dl](https://go.dev/dl/)）。

```bash
go install github.com/biyan113/grok-setup/cmd/gsetup@latest
```

或：

```bash
curl -fsSL https://raw.githubusercontent.com/biyan113/grok-setup/main/install.sh | bash
```

指定版本（代理还没收录 `@latest` 时用这个）：

```bash
go install github.com/biyan113/grok-setup/cmd/gsetup@v0.1.1
```

装完若找不到命令：

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
gsetup version    # 应打印 0.1.1
```

从源码：

```bash
git clone https://github.com/biyan113/grok-setup.git
cd grok-setup
make test && make build    # 当前目录 ./gsetup
make install               # 拷到 ~/.local/bin/gsetup
```

可选：`gsetup search` 安装 grok-search skill（需要 `git`、Node ≥ 18.17、`npm`）。

## 快速开始

官方 `grok` 装好后，加一个网关模型：

```bash
export GATEWAY_API_KEY="你的密钥"
gsetup apply \
  --alias proxy \
  --base-url https://你的网关/v1 \
  --env-key GATEWAY_API_KEY \
  --set-default \
  --privacy \
  --yes
gsetup doctor
```

交互向导（默认入口）：

```bash
gsetup
```

会问：别名（默认 `proxy`）→ `base_url` → 上游模型 ID（默认 `grok-4.6`）→ 环境变量还是手输密钥 → 是否设为 default → 是否开原生搜索 → 是否补隐私字段。确认前只显示掩码，不会打印完整密钥。

先看不写盘：

```bash
gsetup apply --base-url https://gw.example/v1 --env-key GATEWAY_API_KEY --dry-run --yes
```

## 命令

| 命令 | 作用 |
|---|---|
| `gsetup` / `gsetup init` | 交互向导 |
| `gsetup apply` | 非交互合并写入（脚本用，需 `--yes`） |
| `gsetup show` | 脱敏打印当前 `config.toml` |
| `gsetup doctor` | 检查路径、权限、解析、`env_key` 是否已设置 |
| `gsetup backups` | 列出 `~/.grok/backups/` |
| `gsetup restore <文件>` | 从备份恢复；恢复前会再备份当前文件 |
| `gsetup install-cli` | 下载官方安装脚本到临时文件后执行（先确认） |
| `gsetup search` | 安装 [Autsunset/grok-search](https://github.com/Autsunset/grok-search) |
| `gsetup version` | 打印版本 |
| `gsetup help` | 帮助 |

全局可用 `--home DIR` 覆盖 `$GROK_HOME`（默认 `~/.grok`）。

## apply 选项

| 选项 | 默认 | 说明 |
|---|---|---|
| `--alias` | `proxy` | 本地配置名，对应 `[model.名称]` |
| `--model` | `grok-4.6` | 发给上游的模型 ID |
| `--name` | 与 `--model` 相同 | 展示名 |
| `--base-url` | （必填） | `http(s)://…`，通常带 `/v1` |
| `--env-key` | | 从该环境变量读密钥（推荐） |
| `--api-key-stdin` | | 从 stdin 读 `api_key`，不回显 |
| `--set-default` | 关 | 同时改 `[models].default` |
| `--search` | | 写 `web_search` + `supports_backend_search = true` |
| `--no-search` | | 仅当 `web_search` 指向本别名时清掉 |
| `--api-backend` | 不写 | `chat_completions` / `responses` / `messages` |
| `--context-window` | 不写 | 写入 `context_window` |
| `--permission-mode` | 不写 | 仅显式指定时才改 `[ui]` |
| `--privacy` | 关 | **字段缺失时**补关遥测、禁止代码库上传 |
| `--dry-run` | 关 | 打印脱敏结果，不写盘、不备份 |
| `--yes` | 关 | 非交互确认；没有它 `apply` 会中止 |
| `--home` | `$GROK_HOME` 或 `~/.grok` | 配置根目录 |

`--search` 与 `--no-search` 不能同时用。不传两者则搜索相关字段保持原样。

原生搜索按官方文档只需要 `supports_backend_search`，**不依赖** `api_backend = "responses"`。网关真走 Responses 时再加 `--api-backend responses`。

```bash
gsetup apply \
  --base-url https://gw.example/v1 \
  --env-key GATEWAY_API_KEY \
  --search \
  --yes
```

会合并进：

```toml
[models]
web_search = "proxy"

[model.proxy]
model = "grok-4.6"
name = "grok-4.6"
base_url = "https://gw.example/v1"
env_key = "GATEWAY_API_KEY"
supports_backend_search = true
```

## 配置规则

| 项 | 行为 |
|---|---|
| 目标 | `$GROK_HOME/config.toml`，默认 `~/.grok/config.toml` |
| 备份 | 写入前拷到 `~/.grok/backups/config-时间戳.toml` |
| 写入 | 同目录临时文件 → `0600` → 改名替换 |
| 合并 | 只更新你指定的 `[model.别名]` 和点名的字段 |
| 保留 | `[mcp_servers.*]`、`[[marketplace.sources]]`、`[ui]`、其它 `[model.*]` |
| 权限模式 | 默认不写；已有 `always-approve` 也不会被改掉 |
| 解析失败 | 中止，不覆盖 |
| `--privacy` | 只补缺失键，不覆盖已有值 |

`gsetup show` 会掩码 `api_key` / token 等，**`base_url` 保持可见**，方便核对网关地址。

## 安全

| 做 | 不做 |
|---|---|
| 优先 `--env-key`，密钥留在环境变量 | 把含真实 `api_key` 的配置提交到 Git |
| 写盘后 `gsetup doctor` 看权限 | 在聊天、Issue、截图里贴完整密钥 |
| 共享机器用更严的 `permission_mode` | 默认打开 always-approve（本工具也不会替你开） |

`install-cli` 会执行来自 `x.ai` 的脚本；`search` 会把 skill 的 `config.json` 写成 `0600`，覆盖前先备份到 `~/.grok/backups/`。

密钥和网关由你自己申请、保管。泄露或配错造成的损失自行承担。

## 排错

| 现象 | 处理 |
|---|---|
| 找不到 `grok` | `export PATH="$HOME/.grok/bin:$PATH"`，再 `grok --version` |
| 找不到 `gsetup` | `export PATH="$(go env GOPATH)/bin:$PATH"` |
| `@latest` 找不到模块 | 改用 `@v0.1.1`，或 `GOPROXY=direct` |
| `base_url` 无效 | 必须 `http://` 或 `https://`，不能有空格 |
| 现有配置解析失败 | 先修好 TOML，或 `gsetup restore <备份>` |
| 连不上模型 | 查 `base_url`、`env_key` 是否已 `export`、网关控制台 |
| `doctor` 提示 0644 | `chmod 600 ~/.grok/config.toml` |
| Search 报错 / 无搜索 | 去掉 `--search`，或改用 `gsetup search` |
| 想撤销 | `gsetup backups` 然后 `gsetup restore config-….toml` |

官方 CLI 文档在本机：

```text
~/.grok/README.md
~/.grok/docs/user-guide/
```

## 和官方 CLI 的分工

| 命令 | 谁提供 | 做什么 |
|---|---|---|
| `grok` | xAI | 编码 Agent / TUI |
| `gsetup` | 本仓库 | 合并改配置、可选装 CLI / skill |

适合：已有或准备装官方 CLI，要用 grok2api / NewAPI / 自建中转。  
不适合：只想用官方账号登录、完全不需要自定义 `base_url`。那种情况直接 `grok login` 即可。

## 开发

```bash
make test
make vet
make fmt
```

MIT。见 [LICENSE](./LICENSE)。
