#!/usr/bin/env bash
# Start local ningpm API detached (survives IDE/agent shell exit).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/apps/api/bin/ningpm-api"
LOG="$ROOT/apps/api/data/api.log"
PIDFILE="$ROOT/apps/api/data/api.pid"
export NINGPM_VERSION="$(tr -d '[:space:]' < "$ROOT/VERSION")"
export NINGPM_DB_PATH="${NINGPM_DB_PATH:-$ROOT/apps/api/data/ningpm.db}"
export NINGPM_HTTP_ADDR="${NINGPM_HTTP_ADDR:-127.0.0.1:8080}"
export GOROOT="${GOROOT_OVERRIDE:-/opt/homebrew/opt/go/libexec}"
export PATH="$GOROOT/bin:$PATH"

mkdir -p "$(dirname "$LOG")" "$ROOT/apps/api/bin"
if [[ ! -x "$BIN" ]]; then
  (cd "$ROOT/apps/api" && go build -o "$BIN" ./cmd/server)
fi

# stop previous
if [[ -f "$PIDFILE" ]]; then
  old=$(cat "$PIDFILE" || true)
  if [[ -n "${old:-}" ]] && kill -0 "$old" 2>/dev/null; then
    kill "$old" 2>/dev/null || true
    sleep 0.3
  fi
fi
PORT="${NINGPM_HTTP_ADDR##*:}"
if lsof -t -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
  kill $(lsof -t -iTCP:"$PORT" -sTCP:LISTEN) 2>/dev/null || true
  sleep 0.3
fi

NINGPM_API_BIN="$BIN" NINGPM_API_LOG="$LOG" NINGPM_API_PIDFILE="$PIDFILE" NINGPM_API_ROOT="$ROOT" python3 - <<'PY'
import os, sys, time
bin_path = os.environ["NINGPM_API_BIN"]
log_path = os.environ["NINGPM_API_LOG"]
pid_path = os.environ["NINGPM_API_PIDFILE"]
root = os.environ["NINGPM_API_ROOT"]
env = os.environ.copy()
if os.fork() > 0:
    time.sleep(0.6)
    sys.exit(0)
os.setsid()
if os.fork() > 0:
    sys.exit(0)
os.chdir(root)
os.umask(0)
devnull = open(os.devnull, "rb")
log = open(log_path, "ab", buffering=0)
os.dup2(devnull.fileno(), 0)
os.dup2(log.fileno(), 1)
os.dup2(log.fileno(), 2)
with open(pid_path, "w", encoding="utf-8") as f:
    f.write(str(os.getpid()))
os.execve(bin_path, [bin_path], env)
PY

if ! curl -fsS "http://$NINGPM_HTTP_ADDR/api/health" >/dev/null; then
  echo "api failed to start; see $LOG" >&2
  exit 1
fi
echo "ningpm-api $(cat "$PIDFILE") on $NINGPM_HTTP_ADDR (log $LOG)"
