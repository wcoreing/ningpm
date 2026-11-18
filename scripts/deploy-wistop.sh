#!/usr/bin/env bash
# 本机触发：SSH 到 Wistop 执行 git pull + 宿主构建（仅 MySQL 在 Docker）
#
#   ./scripts/deploy-wistop.sh
#   ./scripts/deploy-wistop.sh --cert
#   ./scripts/deploy-wistop.sh status
#   GITEE_TOKEN=xxx ./scripts/deploy-wistop.sh
set -euo pipefail

REMOTE_HOST="${REMOTE_HOST:-root@8.138.250.141}"
REMOTE_DIR="${REMOTE_DIR:-/www/wwwroot/ningpm}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/mytag_ed25519}"
DOMAIN="pm.wistomtop.com"

DO_CERT=0
ACTION=deploy
for arg in "$@"; do
  case "$arg" in
    --cert) DO_CERT=1 ;;
    status) ACTION=status ;;
    -h|--help) sed -n '2,10p' "$0"; exit 0 ;;
  esac
done

SSH=(ssh -i "$SSH_KEY" -o BatchMode=yes -o ConnectTimeout=25 -o StrictHostKeyChecking=accept-new)
log() { printf '==> %s\n' "$*"; }

case "$ACTION" in
  status)
    "${SSH[@]}" "$REMOTE_HOST" "cd ${REMOTE_DIR} && bash scripts/deploy-on-server.sh --verify-only"
    ;;
  deploy)
    args=()
    [ "$DO_CERT" -eq 1 ] && args+=(--cert)
    token_export=""
    if [ -n "${GITEE_TOKEN:-}" ]; then
      token_export="GITEE_TOKEN='${GITEE_TOKEN}' "
    fi
    log "远程部署 ${DOMAIN}（git pull + 本机构建 + Docker MySQL）"
    "${SSH[@]}" "$REMOTE_HOST" "cd ${REMOTE_DIR} && chmod +x scripts/*.sh && ${token_export}bash scripts/deploy-on-server.sh ${args[*]:-}"
    log "DONE https://${DOMAIN}"
    ;;
esac
