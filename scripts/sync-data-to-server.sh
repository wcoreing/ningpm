#!/usr/bin/env bash
# 将本机 SQLite 业务数据 + files 同步到 Wistop 生产 MySQL。
# 用法：./scripts/sync-data-to-server.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="${REMOTE_HOST:-root@8.138.250.141}"
REMOTE_DIR="${REMOTE_DIR:-/www/wwwroot/ningpm}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/mytag_ed25519}"
DB_LOCAL="${NINGPM_DB_PATH:-$ROOT/apps/api/data/ningpm.db}"
FILES_LOCAL="${NINGPM_FILES_DIR:-$ROOT/apps/api/data/files}"
TMP_DIR="$(mktemp -d)"
SQL_OUT="$TMP_DIR/ningpm-data.sql"

SSH=(ssh -i "$SSH_KEY" -o BatchMode=yes -o ConnectTimeout=25 -o StrictHostKeyChecking=accept-new)
SCP=(scp -i "$SSH_KEY" -o BatchMode=yes -o ConnectTimeout=25 -o StrictHostKeyChecking=accept-new)
RSYNC=(rsync -az -e "ssh -i $SSH_KEY -o BatchMode=yes -o ConnectTimeout=25 -o StrictHostKeyChecking=accept-new")

log() { printf '==> %s\n' "$*"; }
die() { printf '[FATAL] %s\n' "$*" >&2; exit 1; }

[ -f "$DB_LOCAL" ] || die "本地库不存在: $DB_LOCAL"
need_cmd() { command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"; }
need_cmd sqlite3
need_cmd python3

cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

log "从 SQLite 导出业务数据 → MySQL SQL"
python3 - "$DB_LOCAL" "$SQL_OUT" <<'PY'
import sqlite3, sys, re

db_path, out_path = sys.argv[1], sys.argv[2]
# 依赖顺序（先父后子）
tables = [
    "users",
    "projects",
    "project_members",
    "cycles",
    "modules",
    "issues",
    "comments",
    "resources",
    "issue_attachments",
    "project_files",
    "notifications",
]

conn = sqlite3.connect(db_path)
conn.row_factory = sqlite3.Row
cur = conn.cursor()

def esc(v):
    if v is None:
        return "NULL"
    if isinstance(v, (int, float)):
        return str(v)
    s = str(v).replace("\\", "\\\\").replace("'", "''")
    return "'" + s + "'"

existing = {r[0] for r in cur.execute("SELECT name FROM sqlite_master WHERE type='table'")}
lines = [
    "SET NAMES utf8mb4;",
    "SET FOREIGN_KEY_CHECKS=0;",
]
for t in reversed(tables):
    if t in existing:
        lines.append(f"TRUNCATE TABLE `{t}`;")

for t in tables:
    if t not in existing:
        continue
    cols = [r[1] for r in cur.execute(f"PRAGMA table_info(`{t}`)")]
    rows = list(cur.execute(f"SELECT * FROM `{t}`"))
    lines.append(f"-- {t}: {len(rows)} rows")
    if not rows:
        continue
    col_sql = ",".join(f"`{c}`" for c in cols)
    for row in rows:
        vals = ",".join(esc(row[c]) for c in cols)
        lines.append(f"INSERT INTO `{t}` ({col_sql}) VALUES ({vals});")

lines.append("SET FOREIGN_KEY_CHECKS=1;")
with open(out_path, "w", encoding="utf-8") as f:
    f.write("\n".join(lines) + "\n")
print(f"wrote {out_path} ({sum(1 for l in lines if l.startswith('INSERT'))} inserts)")
conn.close()
PY

log "上传 SQL 到服务器"
"${SCP[@]}" "$SQL_OUT" "${REMOTE_HOST}:${REMOTE_DIR}/data/ningpm-import.sql"

log "同步附件目录 files/"
mkdir -p "$FILES_LOCAL"
"${SSH[@]}" "$REMOTE_HOST" "mkdir -p ${REMOTE_DIR}/data/files"
"${RSYNC[@]}" --delete "$FILES_LOCAL/" "${REMOTE_HOST}:${REMOTE_DIR}/data/files/"

log "导入 MySQL（覆盖业务表）并重启 API"
"${SSH[@]}" "$REMOTE_HOST" bash -s <<'REMOTE'
set -euo pipefail
APP=/www/wwwroot/ningpm
cd "$APP"
# shellcheck disable=SC1091
set -a
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in ''|\#*) continue ;; esac
  export "$line"
done < .env
set +a
[ -n "${MYSQL_PASSWORD:-}" ] || { echo "缺少 MYSQL_PASSWORD"; exit 1; }
docker exec -i ningpm-mysql mysql -uningpm -p"${MYSQL_PASSWORD}" ningpm < data/ningpm-import.sql
systemctl restart ningpm-api.service
sleep 2
curl -fsS "http://127.0.0.1:${API_PORT:-8091}/api/health"
echo
curl -fsS "http://127.0.0.1:${API_PORT:-8091}/api/version"
echo
docker exec ningpm-mysql mysql -uningpm -p"${MYSQL_PASSWORD}" -N -e "SELECT 'users',COUNT(*) FROM ningpm.users UNION ALL SELECT 'projects',COUNT(*) FROM ningpm.projects UNION ALL SELECT 'issues',COUNT(*) FROM ningpm.issues UNION ALL SELECT 'modules',COUNT(*) FROM ningpm.modules;"
REMOTE

log "DATA_SYNC_OK"
