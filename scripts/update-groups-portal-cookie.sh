#!/usr/bin/env bash
# Update the Groups Portal session cookie used for Church Guide fetches, and
# restart the service so it picks up the new value.
#
# The cookie itself can't be obtained automatically (see docs/adr/0001-...):
# it must come from a logged-in browser session. In DevTools -> Network,
# click any mycgconnect.com/api/... request, then Headers -> Request
# Headers -> copy the full "cookie" value (starts with "connect.sid=").
#
# Writes to an env file rather than the systemd unit directly: unit file
# Environment= values go through systemd's %-specifier expansion, which
# corrupts a URL-encoded cookie (e.g. %3A). An EnvironmentFile='s contents
# are not expanded, so no escaping is needed here.
#
# Environment:
#   ENV_FILE      Path to the env file (default: /etc/youtube-transcript.env)
#   SYSTEMD_UNIT  If set, run: sudo systemctl restart "$SYSTEMD_UNIT" after writing
#
# Usage:
#   SYSTEMD_UNIT=youtube-transcript ./scripts/update-groups-portal-cookie.sh
#   (prompts for the cookie value; input is not echoed, not passed as an
#   argument, so it never lands in shell history or `ps`)
#
# One-time setup: the systemd unit needs
#   EnvironmentFile=/etc/youtube-transcript.env
# (or your ENV_FILE path) added under [Service], then `daemon-reload`.
#
set -euo pipefail

ENV_FILE="${ENV_FILE:-/etc/youtube-transcript.env}"

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

if [[ -f "$ENV_FILE" ]]; then
	grep -v '^GROUPS_PORTAL_SESSION_COOKIE=' "$ENV_FILE" > "$TMP" || true
fi
echo "GROUPS_PORTAL_SESSION_COOKIE=${COOKIE_VALUE}" >> "$TMP"

sudo install -m 600 "$TMP" "$ENV_FILE"

echo "==> wrote GROUPS_PORTAL_SESSION_COOKIE to ${ENV_FILE}"

if [[ -n "${SYSTEMD_UNIT:-}" ]]; then
	echo "==> systemctl restart ${SYSTEMD_UNIT}"
	sudo systemctl restart "$SYSTEMD_UNIT"
	sudo systemctl --no-pager -l status "$SYSTEMD_UNIT" || true
else
	echo "==> done (set SYSTEMD_UNIT to restart a service, e.g. SYSTEMD_UNIT=youtube-transcript)"
fi
