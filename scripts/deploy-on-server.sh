#!/usr/bin/env bash
# ningpm 服务器端部署（Wistop / pm.wistomtop.com）
# - git pull 拉代码
# - 本机 pnpm / go 构建前后端
# - 仅 MySQL 跑在 Docker（ningpm-mysql :3308）
# - API 用 systemd 托管
#
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
SERVICE="ningpm-api.service"

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
      sed -n '2,12p' "$0"
      exit 0
      ;;
    *)
      echo "未知参数: $arg" >&2
      exit 2
      ;;
  esac
done

export PATH="/usr/local/go/bin:/usr/local/bin:$(npm prefix -g 2>/dev/null)/bin:${PATH:-/usr/bin:/bin}"
log() { printf '[ningpm-deploy] %s\n' "$*"; }
die() { printf '[ningpm-deploy][FATAL] %s\n' "$*" >&2; exit 1; }
need_cmd() { command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"; }

cd "$APP" || die "目录不存在: $APP"
VERSION="$(tr -d '[:space:]' < VERSION 2>/dev/null || echo 0.0.0)"

# 停掉旧的「API 进容器」形态（若存在）
docker rm -f ningpm-api >/dev/null 2>&1 || true

verify() {
  log "==> verify"
  curl -fsS "http://127.0.0.1:${API_PORT}/api/health" | grep -q . || die "api health 失败"
  docker ps --filter name=ningpm-mysql --format '{{.Names}} {{.Status}} {{.Ports}}' | grep -q ningpm-mysql || die "mysql 容器未运行"
  code="$(curl -s -o /dev/null -w '%{http_code}' "https://${DOMAIN}/" || true)"
  log "https_${DOMAIN}=${code}"
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
    die "尚未 clone。执行: git clone ${REMOTE_URL_CLEAN} ${APP}"
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
if [ ! -f .env ] || ! grep -q '^NINGPM_DB_DRIVER=mysql' .env; then
  cp deploy/.env.example .env
  secret="$(openssl rand -hex 24)"
  mysql_pw="$(openssl rand -hex 12)"
  root_pw="$(openssl rand -hex 12)"
  sed -i "s/^NINGPM_JWT_SECRET=.*/NINGPM_JWT_SECRET=${secret}/" .env
  sed -i "s/^MYSQL_PASSWORD=.*/MYSQL_PASSWORD=${mysql_pw}/" .env
  sed -i "s/^MYSQL_ROOT_PASSWORD=.*/MYSQL_ROOT_PASSWORD=${root_pw}/" .env
  sed -i "s|^NINGPM_DB_DSN=.*|NINGPM_DB_DSN=ningpm:${mysql_pw}@tcp(127.0.0.1:3308)/ningpm?parseTime=true\&charset=utf8mb4\&loc=UTC\&multiStatements=true|" .env
  log "已生成/重建 .env（MySQL + 随机密钥）"
fi
sed -i "s/^NINGPM_VERSION=.*/NINGPM_VERSION=${VERSION}/" .env
grep -q '^NINGPM_DB_DRIVER=mysql' .env || die ".env 未配置 NINGPM_DB_DRIVER=mysql"
# shellcheck disable=SC1091
set -a
# 去掉可能的 export 前缀，按 KEY=VAL 加载
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    ''|\#*) continue ;;
  esac
  export "$line"
done < .env
set +a

# ---------- mysql (docker only) ----------
need_cmd docker
log "==> docker compose up mysql（仅数据库）"
docker compose -f deploy/docker-compose.yml --env-file .env up -d mysql
log "等待 MySQL 就绪…"
for i in $(seq 1 40); do
  if docker exec ningpm-mysql mysqladmin ping -h127.0.0.1 -uningpm -p"${MYSQL_PASSWORD}" --silent 2>/dev/null; then
    log "MYSQL_OK"
    break
  fi
  if [ "$i" -eq 40 ]; then
    docker logs --tail 40 ningpm-mysql || true
    die "MySQL 未就绪"
  fi
  sleep 2
done

# ---------- web ----------
if [ "$SKIP_WEB" -eq 0 ]; then
  log "==> build web（宿主 pnpm）"
  need_cmd node
  export COREPACK_ENABLE_DOWNLOAD_PROMPT=0
  if ! command -v pnpm >/dev/null 2>&1; then
    log "安装 pnpm"
    if command -v corepack >/dev/null 2>&1; then
      corepack enable || true
      corepack prepare pnpm@9.15.9 --activate || true
    fi
    if ! command -v pnpm >/dev/null 2>&1; then
      npm i -g pnpm@9
      hash -r 2>/dev/null || true
      export PATH="$(npm prefix -g)/bin:${PATH}"
    fi
  fi
  if command -v pnpm >/dev/null 2>&1; then
    PNPM=(pnpm)
  else
    PNPM=(npx --yes pnpm@9)
  fi
  "${PNPM[@]}" install --frozen-lockfile || "${PNPM[@]}" install
  "${PNPM[@]}" --filter @ningpm/web exec vite build
  [ -f apps/web/dist/index.html ] || die "缺少 apps/web/dist/index.html"
  log "BUILD_WEB_OK"
else
  log "==> skip web"
fi

# ---------- api (host go build + systemd) ----------
if [ "$SKIP_API" -eq 0 ]; then
  need_cmd go
  log "==> build api（宿主 go）"
  export GOPROXY="$GOPROXY_URL"
  mkdir -p apps/api/bin data/files logs
  (
    cd apps/api
    go build -o bin/api ./cmd/server
  )
  [ -x apps/api/bin/api ] || die "缺少 apps/api/bin/api"
  log "BUILD_API_OK"

  log "==> systemd ${SERVICE} :${API_PORT}"
  install -m 0644 "$APP/scripts/${SERVICE}" "/etc/systemd/system/${SERVICE}"
  systemctl daemon-reload
  systemctl enable "$SERVICE" >/dev/null 2>&1 || true
  systemctl restart "$SERVICE"
  sleep 2
  curl -fsS "http://127.0.0.1:${API_PORT}/api/health" >/dev/null || {
    journalctl -u "$SERVICE" -n 40 --no-pager || true
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
