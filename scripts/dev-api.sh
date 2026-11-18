#!/usr/bin/env bash
# Run Gin API with a consistent Homebrew Go toolchain.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export GOROOT="${GOROOT_OVERRIDE:-/opt/homebrew/opt/go/libexec}"
export NINGPM_VERSION="$(tr -d '[:space:]' < "$ROOT/VERSION")"
export NINGPM_DB_PATH="${NINGPM_DB_PATH:-$ROOT/apps/api/data/ningpm.db}"
cd "$ROOT/apps/api"
exec go run ./cmd/server
