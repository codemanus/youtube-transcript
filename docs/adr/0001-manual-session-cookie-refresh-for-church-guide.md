# Manual session-cookie refresh for Church Guide fetch, not automated login

The Groups Portal (`mycgconnect.com`) authenticates with a long-lived session cookie (~30-day expiry) issued after a one-time email login, not per-request credentials. To fetch the Church Guide, we chose to store a manually-refreshed session cookie (originally an env var, matching the existing `TRANSCRIPT_API_URL` convention; see [ADR-0002](0002-file-based-groups-portal-cookie.md) for why that later moved to a file) rather than storing Cody's real portal password and automating the login flow ourselves.

This avoids putting a real password at rest on the LAN box in exchange for an occasional manual chore: when the cookie expires, Church Guide fetches fail with a clear error (same pattern as the existing "service unreachable" error) until it's refreshed by hand. Acceptable because the tool is already touched weekly and refresh is infrequent (~monthly).
