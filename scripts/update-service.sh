#!/usr/bin/env bash
# Pull latest from GitHub, rebuild, install binary, optionally restart systemd.
# Run as a normal user (with git credentials); sudo is only used for install + systemctl.
#
# Environment:
#   INSTALL_BIN   Path to install the binary (default: /usr/local/bin/youtube-transcript)
#   SYSTEMD_UNIT  If set, run: sudo systemctl restart "$SYSTEMD_UNIT" after install
#   GIT_BRANCH    After git fetch, merge origin/GIT_BRANCH with ff-only (optional; default is git pull --ff-only)
#
# Example (on LXC, repo at /opt/youtube-transcript):
#   cd /opt/youtube-transcript && SYSTEMD_UNIT=youtube-transcript ./scripts/update-service.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	echo "error: not a git clone — clone the repo first, then run this script from it." >&2
	exit 1
fi

INSTALL_BIN="${INSTALL_BIN:-/usr/local/bin/youtube-transcript}"

echo "==> git fetch"
git fetch origin

if [[ -n "${GIT_BRANCH:-}" ]]; then
	echo "==> git merge --ff-only origin/${GIT_BRANCH}"
	git merge --ff-only "origin/${GIT_BRANCH}"
else
	echo "==> git pull --ff-only"
	git pull --ff-only
fi

echo "==> make release-for-host"
make release-for-host

OUT="$(make -s print-release-binary)"
if [[ ! -f "$OUT" ]]; then
	echo "error: expected binary missing: $OUT" >&2
	exit 1
fi

echo "==> install -> ${INSTALL_BIN}"
sudo install -m 755 "$OUT" "$INSTALL_BIN"

if [[ -n "${SYSTEMD_UNIT:-}" ]]; then
	echo "==> systemctl restart ${SYSTEMD_UNIT}"
	sudo systemctl restart "$SYSTEMD_UNIT"
	sudo systemctl --no-pager -l status "$SYSTEMD_UNIT" || true
else
	echo "==> done (set SYSTEMD_UNIT to restart a service, e.g. SYSTEMD_UNIT=youtube-transcript)"
fi
