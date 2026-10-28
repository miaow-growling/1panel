#!/bin/bash
# ============================================================================
# 1Panel 社区多节点版 —— 一键构建脚本
#
# 用法:
#   ./scripts/build-multinode.sh [amd64|arm64|all] [--skip-frontend]
#
# 产物 (build/ 目录):
#   1panel-core-linux-<arch>   主控 (含多节点管理/远程代理/一键安装)
#   1panel-agent-linux-<arch>  被控 (官方单机模式 + PANEL_NODE_MODE 远程模式)
#
# 升级已安装官方社区 v2 的服务器:
#   1. scp 两个产物到服务器 /tmp/
#   2. systemctl stop 1panel-core 1panel-agent   # 服务名以实际为准
#   3. 覆盖 /usr/local/bin/1panel-core 与 /usr/local/bin/1panel-agent
#   4. systemctl start 1panel-agent 1panel-core  # 启动时自动建 host_nodes 表
#   数据与配置不动; 换回官方二进制即可回退。
#
# 注意: 不要用面板内"在线升级"(1pctl upgrade), 会把二进制换回官方版。
# ============================================================================
set -euo pipefail
cd "$(dirname "$0")/.."

ARCH="${1:-amd64}"
SKIP_FRONTEND="${2:-}"

case "$ARCH" in
  amd64|arm64|all) ;;
  *) echo "用法: $0 [amd64|arm64|all] [--skip-frontend]"; exit 1 ;;
esac

# ---------- 前端 ----------
if [ "$SKIP_FRONTEND" != "--skip-frontend" ]; then
  echo "==> [1/2] 构建前端 (输出到 core/cmd/server/web)"
  (cd frontend \
    && { [ -d node_modules ] || npm install --no-audit --no-fund; } \
    && npx vite build --mode production)
  # emptyOutDir 会清掉 web/ 里的 go embed 声明, 从 git 恢复
  ./build-after-frontend.sh
else
  echo "==> [1/2] 跳过前端构建"
  [ -f core/cmd/server/web/web.go ] || { echo "错误: web.go 不存在, 请先完整构建一次前端"; exit 1; }
fi

# ---------- 后端 ----------
build_one() {
  local name="$1" goarch="$2"
  echo "==> [2/2] 构建 linux/$goarch"
  (cd core  && CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
      go build -trimpath -ldflags '-s -w' -o "../build/1panel-core-linux-$name" ./cmd/server)
  (cd agent && CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
      go build -trimpath -ldflags '-s -w' -o "../build/1panel-agent-linux-$name" ./cmd/server)
}

mkdir -p build
case "$ARCH" in
  all) build_one amd64 amd64; build_one arm64 arm64 ;;
  *)   build_one "$ARCH" "$ARCH" ;;
esac

echo ""
echo "==> 构建完成:"
ls -lh build/1panel-*linux-* | awk '{print "   " $9 "  (" $5 ")"}'

# ---------- 生成带版本号的 1pctl（面板版本显示用） ----------
# 版本号自动取当前基线所属的官方最新 tag，拼上 -community-multinode
BASE_TAG=$(git describe --tags --abbrev=0 HEAD 2>/dev/null || echo "v2.4.1")
VERSION="${BASE_TAG}-community-multinode"
cat > build/1pctl <<PCTL
BASE_DIR=/opt
ORIGINAL_PORT=9999
ORIGINAL_VERSION=${VERSION}
ORIGINAL_USERNAME=admin
ORIGINAL_PASSWORD=admin123
ORIGINAL_ENTRANCE=entrance
LANGUAGE=zh
PCTL
chmod 644 build/1pctl
echo "==> 已生成 build/1pctl (版本显示: ${VERSION})"
