#!/usr/bin/env bash
# 一行安装 gsetup：
#   curl -fsSL https://raw.githubusercontent.com/biyan113/grok-setup/main/install.sh | bash
set -euo pipefail

REPO="github.com/biyan113/grok-setup"
BIN="gsetup"

if ! command -v go >/dev/null 2>&1; then
  echo "需要 Go 1.22+。先安装: https://go.dev/dl/" >&2
  exit 1
fi

echo "go install ${REPO}/cmd/${BIN}@latest"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"
go install "${REPO}/cmd/${BIN}@latest"

dest="${GOBIN}/${BIN}"
if [[ -x "$dest" ]]; then
  echo "已安装: $dest"
else
  echo "go install 完成，但没在 ${GOBIN} 找到 ${BIN}" >&2
  echo "把下面加入 PATH:  export PATH=\"${GOBIN}:\$PATH\"" >&2
  exit 1
fi

case ":${PATH}:" in
  *":${GOBIN}:"*) ;;
  *)
    echo "当前 PATH 里没有 ${GOBIN}，新开终端前先执行:"
    echo "  export PATH=\"${GOBIN}:\$PATH\""
    ;;
esac

echo
echo "启动向导:  ${BIN}"
echo "查看帮助:  ${BIN} help"
echo "本命令不会覆盖官方 grok。"
