#!/usr/bin/env bash
set -euo pipefail

# Updates embedded Tor sources/configs/go wrappers using upstream release branch.
# By default uses release-0.4.8 from the Tor GitLab mirror.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TOR_REPO_URL="${TOR_REPO_URL:-https://gitlab.torproject.org/tpo/core/tor.git}"
TOR_BRANCH="${TOR_BRANCH:-release-0.4.8}"
NOBUILD="${NOBUILD:-1}"

printf '==> go-libtor tor updater\n'
printf '    repo:   %s\n' "$TOR_REPO_URL"
printf '    branch: %s\n' "$TOR_BRANCH"
printf '    nobuild:%s\n' "$NOBUILD"

PY_BIN=""
if command -v python3 >/dev/null 2>&1; then
  PY_BIN="python3"
elif command -v python >/dev/null 2>&1; then
  PY_BIN="python"
else
  echo "error: python3/python not found; cannot patch build/wrap.go" >&2
  exit 1
fi

# Patch build/wrap.go clone target for this run.
TOR_REPO_URL="$TOR_REPO_URL" TOR_BRANCH="$TOR_BRANCH" "$PY_BIN" - <<'PY'
from pathlib import Path
import os, re

p = Path('build/wrap.go')
s = p.read_text()
repo = os.environ.get('TOR_REPO_URL', 'https://gitlab.torproject.org/tpo/core/tor.git')
branch = os.environ.get('TOR_BRANCH', 'release-0.4.8')
pat = r'exec\.Command\("git", "clone", "--depth", "1", "--branch", "[^"]+", "[^"]+"\)'
rep = f'exec.Command("git", "clone", "--depth", "1", "--branch", "{branch}", "{repo}")'
ns, n = re.subn(pat, rep, s, count=1)
if n != 1:
    raise SystemExit('failed to patch tor clone command in build/wrap.go')
p.write_text(ns)
PY

printf '==> running wrapper generator\n'
if [[ "$NOBUILD" == "1" ]]; then
  go run build/wrap.go -nobuild
else
  go run build/wrap.go
fi

printf '==> done. review changes with: git status && git diff --stat\n'
