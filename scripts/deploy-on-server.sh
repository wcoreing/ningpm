#!/usr/bin/env bash
# ningpm 服务器端部署（Wistop / pm.wistomtop.com）
# 用法：
#   bash /www/wwwroot/ningpm/scripts/deploy-on-server.sh
#   GITEE_TOKEN=xxx bash scripts/deploy-on-server.sh
#   bash scripts/deploy-on-server.sh --skip-pull
#   bash scripts/deploy-on-server.sh --cert
set -euo pipefail

APP="${APP_DIR:-/www/wwwroot/ningpm}"
BRANCH="${DEPLOY_BRANCH:-master}"
REMOTE_URL_CLEAN="https://gitee.com/wei3ning/ningpm.git"
DOMAIN="${SITE_HOST:-pm.wistomtop.com}"
API_PORT="${API_PORT:-8091}"
GOPROXY_URL="${GOPROXY:-https://goproxy.cn,direct}"

SKIP_PULL=0
SKIP_WEB=0
SKIP_API=0
DO_CERT=0
VERIFY_ONLY=0

for arg in "$@"; do
  case "$arg" in
    --skip-pull) SKIP_PULL=1 ;;
    --skip-web) SKIP_WEB=1 ;;
    --skip-api) SKIP_API=1 ;;
    --cert) DO_CERT=1 ;;
    --verify-only) VERIFY_ONLY=1 ;;
    -h|--help)
      sed -n '2,10p' "$0"
      exit 0
      ;;
    *)
      echo "未知参数: $arg" >&2
      exit 2
      ;;
  esac
done

export PATH="/usr/local/go/bin:/usr/local/bin:${PATH:-/usr/bin:/bin}"
log() { printf '[ningpm-deploy] %s\n' "$*"; }
die() { printf '[ningpm-deploy][FATAL] %s\n' "$*" >&2; exit 1; }
need_cmd() { command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"; }

cd "$APP" || die "目录不存在: $APP"
VERSION="$(tr -d '[:space:]' < VERSION 2>/dev/null || echo 0.0.0)"

verify() {
  log "==> verify"
  curl -fsS "http://127.0.0.1:${API_PORT}/api/health" | grep -q . || die "api health 失败"
  code="$(curl -s -o /dev/null -w '%{http_code}' "https://${DOMAIN}/" || true)"
  log "https_${DOMAIN}=${code}"
  docker ps --filter name=ningpm-api --format '{{.Names}} {{.Status}} {{.Ports}}'
  log "VERIFY_OK v${VERSION}"
}

if [ "$VERIFY_ONLY" -eq 1 ]; then
  verify
  exit 0
fi

# ---------- pull ----------
if [ "$SKIP_PULL" -eq 0 ]; then
  need_cmd git
  if [ ! -d .git ]; then
    die "尚未 git init/clone。本机先 push 到 Gitee，或: git clone ${REMOTE_URL_CLEAN} ${APP}"
  fi
  log "==> git pull ${BRANCH}"
  CLEANED_REMOTE=0
  DEPLOY_TOKEN="${GITEE_TOKEN:-${GIT_TOKEN:-${GITHUB_TOKEN:-}}}"
  if [ -n "$DEPLOY_TOKEN" ]; then
    git remote set-url origin "https://oauth2:${DEPLOY_TOKEN}@gitee.com/wei3ning/ningpm.git"
    CLEANED_REMOTE=1
  else
    git remote set-url origin "$REMOTE_URL_CLEAN" || true
  fi
  cleanup_remote() {
    if [ "$CLEANED_REMOTE" -eq 1 ]; then
      git remote set-url origin "$REMOTE_URL_CLEAN" || true
      CLEANED_REMOTE=0
    fi
  }
  trap cleanup_remote EXIT
  git fetch origin
  # 兼容 main / master
  if git rev-parse --verify "origin/${BRANCH}" >/dev/null 2>&1; then
    git reset --hard "origin/${BRANCH}"
  elif git rev-parse --verify origin/main >/dev/null 2>&1; then
    git reset --hard origin/main
  else
    die "远程没有 ${BRANCH} 或 main"
  fi
  cleanup_remote
  trap - EXIT
  git log -1 --oneline
  VERSION="$(tr -d '[:space:]' < VERSION)"
else
  log "==> skip pull"
fi

# ---------- env ----------
if [ ! -f .env ]; then
  cp deploy/.env.example .env
  secret="$(openssl rand -hex 24)"
  sed -i "s/^NINGPM_JWT_SECRET=.*/NINGPM_JWT_SECRET=${secret}/" .env
  log "已生成 .env"
fi
sed -i "s/^NINGPM_VERSION=.*/NINGPM_VERSION=${VERSION}/" .env

# ---------- web ----------
if [ "$SKIP_WEB" -eq 0 ]; then
  log "==> build web"
  need_cmd node
  need_cmd pnpm || npm i -g pnpm@9
  export COREPACK_ENABLE_DOWNLOAD_PROMPT=0
  pnpm install --frozen-lockfile || pnpm install
  pnpm --filter @ningpm/web exec vite build
  [ -f apps/web/dist/index.html ] || die "缺少 apps/web/dist/index.html"
  log "BUILD_WEB_OK"
else
  log "==> skip web"
fi

# ---------- api (docker) ----------
if [ "$SKIP_API" -eq 0 ]; then
  need_cmd docker
  log "==> docker compose up api（SQLite → volume ningpm_data，端口 ${API_PORT}）"
  export NINGPM_VERSION="$VERSION"
  export GOPROXY="$GOPROXY_URL"
  docker compose -f deploy/docker-compose.yml --env-file .env up -d --build api
  sleep 2
  curl -fsS "http://127.0.0.1:${API_PORT}/api/health" >/dev/null || {
    docker logs --tail 50 ningpm-api || true
    die "api 未就绪"
  }
  log "API_OK"
else
  log "==> skip api"
fi

# ---------- nginx ----------
log "==> nginx ${DOMAIN}"
if [ -f "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ]; then
  cp deploy/nginx/pm.wistomtop.com.conf "/etc/nginx/sites-available/${DOMAIN}"
else
  cp deploy/nginx/pm.wistomtop.com.http-only.conf "/etc/nginx/sites-available/${DOMAIN}"
  log "尚无证书，使用 HTTP 占位；跑: $0 --cert"
fi
ln -sf "/etc/nginx/sites-available/${DOMAIN}" "/etc/nginx/sites-enabled/${DOMAIN}"
nginx -t
systemctl reload nginx

if [ "$DO_CERT" -eq 1 ]; then
  need_cmd certbot
  log "==> certbot ${DOMAIN}"
  # 确保 HTTP 配置可做 ACME
  cp deploy/nginx/pm.wistomtop.com.http-only.conf "/etc/nginx/sites-available/${DOMAIN}"
  ln -sf "/etc/nginx/sites-available/${DOMAIN}" "/etc/nginx/sites-enabled/${DOMAIN}"
  nginx -t && systemctl reload nginx
  certbot certonly --nginx -d "${DOMAIN}" --non-interactive --agree-tos \
    --email admin@wistomtop.com --keep-until-expiring
  cp deploy/nginx/pm.wistomtop.com.conf "/etc/nginx/sites-available/${DOMAIN}"
  nginx -t && systemctl reload nginx
  log "CERT_OK"
fi

verify
log "DEPLOY_DONE https://${DOMAIN} v${VERSION}"
