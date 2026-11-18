#!/usr/bin/env bash
# 本机 → Wistop 部署入口（pm.wistomtop.com）
# 优先：本机 push Gitee → 服务器 git pull + 构建
# 兜底：rsync 代码后 --skip-pull
#
# 用法：
#   ./scripts/deploy-wistop.sh              # rsync + 服务器构建（含证书若已有）
#   ./scripts/deploy-wistop.sh --from-git   # 服务器 git pull（需已 clone 且能访问 Gitee）
#   ./scripts/deploy-wistop.sh --cert       # 额外申请/续期证书
#   ./scripts/deploy-wistop.sh status
#
# 环境变量：
#   REMOTE_HOST  默认 root@8.138.250.141
#   REMOTE_DIR   默认 /www/wwwroot/ningpm
#   SSH_KEY      默认 ~/.ssh/mytag_ed25519
#   GITEE_TOKEN  可选，传给服务器 pull

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="${REMOTE_HOST:-root@8.138.250.141}"
REMOTE_DIR="${REMOTE_DIR:-/www/wwwroot/ningpm}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/mytag_ed25519}"
DOMAIN="pm.wistomtop.com"
VERSION="$(tr -d '[:space:]' < "$ROOT/VERSION")"

FROM_GIT=0
DO_CERT=0
ACTION=deploy

for arg in "$@"; do
  case "$arg" in
    --from-git) FROM_GIT=1 ;;
    --cert) DO_CERT=1 ;;
    status) ACTION=status ;;
    -h|--help)
      sed -n '2,18p' "$0"
      exit 0
      ;;
  esac
done

SSH=(ssh -i "$SSH_KEY" -o BatchMode=yes -o ConnectTimeout=25 -o StrictHostKeyChecking=accept-new)
RSYNC=(rsync -az --delete -e "ssh -i ${SSH_KEY} -o BatchMode=yes -o ConnectTimeout=25")

log() { printf '==> %s\n' "$*"; }
ssh_r() { "${SSH[@]}" "$REMOTE_HOST" "$@"; }

rsync_code() {
  log "rsync → ${REMOTE_HOST}:${REMOTE_DIR}"
  ssh_r "mkdir -p ${REMOTE_DIR}"
  "${RSYNC[@]}" \
    --exclude '.git/' \
    --exclude 'node_modules/' \
    --exclude 'apps/web/node_modules/' \
    --exclude 'apps/api/bin/' \
    --exclude 'apps/api/tmp/' \
    --exclude 'apps/api/data/' \
    --exclude 'apps/web/dist/' \
    --exclude '.pnpm-store/' \
    --exclude '.env' \
    --exclude '*.db' \
    --exclude '.DS_Store' \
    "$ROOT/" "$REMOTE_HOST:${REMOTE_DIR}/"
}

remote_deploy() {
  local args=()
  if [ "$FROM_GIT" -eq 1 ]; then
    args=()
  else
    args=(--skip-pull)
  fi
  [ "$DO_CERT" -eq 1 ] && args+=(--cert)
  local token_export=""
  if [ -n "${GITEE_TOKEN:-}" ]; then
    token_export="GITEE_TOKEN='${GITEE_TOKEN}' "
  fi
  log "远程执行 deploy-on-server.sh ${args[*]:-}"
  ssh_r "cd ${REMOTE_DIR} && chmod +x scripts/deploy-on-server.sh && ${token_export}bash scripts/deploy-on-server.sh ${args[*]:-}"
}

status() {
  ssh_r "cd ${REMOTE_DIR} 2>/dev/null && bash scripts/deploy-on-server.sh --verify-only || echo '尚未部署'"
  ssh_r "docker volume ls | grep ningpm || true; ls /etc/nginx/sites-enabled/${DOMAIN} 2>/dev/null || true"
}

case "$ACTION" in
  status) status ;;
  deploy)
    if [ "$FROM_GIT" -eq 1 ]; then
      log "使用服务器 git pull（需 ${REMOTE_DIR} 已 clone）"
    else
      rsync_code
    fi
    remote_deploy
    log "DONE https://${DOMAIN} v${VERSION}"
    ;;
esac
