#!/usr/bin/env bash
# Update the Groups Portal session cookie used for Church Guide fetches.
#
# The cookie itself can't be obtained automatically (see docs/adr/0001-...):
# it must come from a logged-in browser session. In DevTools -> Network,
# click any mycgconnect.com/api/... request, then Headers -> Request
# Headers -> copy the full "cookie" value (starts with "connect.sid=").
#
# Writes the raw cookie value straight to the cookie file (see
# docs/adr/0002-...) — the running service reads this file fresh on every
# Church Guide request, so a restart is not required for the new value to
# take effect.
#
# Environment:
#   COOKIE_FILE   Path to the cookie file (default: /var/lib/youtube-transcript/groups-portal-cookie)
#   SYSTEMD_UNIT  If set, run: sudo systemctl restart "$SYSTEMD_UNIT" after writing (harmless, not required)
#
# Usage:
#   ./scripts/update-groups-portal-cookie.sh
#   (prompts for the cookie value; input is not echoed, not passed as an
#   argument, so it never lands in shell history or `ps`)
#
set -euo pipefail

COOKIE_FILE="${COOKIE_FILE:-/var/lib/youtube-transcript/groups-portal-cookie}"

echo "Paste the Groups Portal Cookie header value (starts with connect.sid=), then press Enter:" >&2
read -rs COOKIE_VALUE
echo >&2

if [[ -z "$COOKIE_VALUE" ]]; then
	echo "error: empty value, nothing written." >&2
	exit 1
fi

TMP="$(mktemp)"
chmod 600 "$TMP"
trap 'rm -f "$TMP"' EXIT

printf '%s' "$COOKIE_VALUE" > "$TMP"

sudo install -d -m 700 "$(dirname "$COOKIE_FILE")"

# install/cp write the destination in place; a concurrent read (the running
# service reads this file on every Church Guide request) could observe a
# partially-written value mid-copy. Stage in the same directory instead, then
# mv — a rename within one directory is atomic, so a concurrent read always
# sees either the old value or the new one in full, never a partial write.
STAGED="${COOKIE_FILE}.new"
sudo install -m 600 "$TMP" "$STAGED"
sudo mv -f "$STAGED" "$COOKIE_FILE"

echo "==> wrote cookie to ${COOKIE_FILE}"

if [[ -n "${SYSTEMD_UNIT:-}" ]]; then
	echo "==> systemctl restart ${SYSTEMD_UNIT} (not required, but harmless)"
	sudo systemctl restart "$SYSTEMD_UNIT"
	sudo systemctl --no-pager -l status "$SYSTEMD_UNIT" || true
else
	echo "==> done (the running service will pick up the new cookie on its next request)"
fi
