# YouTube transcript (Go + SvelteKit)

Small LAN/VPN tool: paste a YouTube URL or video ID, get caption text for notes and follow-ups.

## Layout

- [`backend/`](backend/) — one Go process: `POST /api/transcript`, `GET /api/health`, and static UI from `embed`.
- [`frontend/`](frontend/) — SvelteKit with `@sveltejs/adapter-static`. In dev, Vite proxies `/api` to the Go server.

## Prerequisites

- Go **1.25+** (required by [`youtube-transcript-go`](https://github.com/rahadiangg/youtube-transcript-go))
- Node **20+** and npm (only for building the UI)

## Build (recommended)

From the repo root, produce `dist/youtube-transcript` with the UI embedded:

```bash
make build
```

This runs `npm ci` and `vite build` in `frontend/`, copies `frontend/build/` into `backend/static/`, then compiles the Go binary.

For a **Linux amd64** artifact on another machine (fits small Proxmox LXCs; build on your laptop/CI to save disk on the guest):

```bash
make release-linux
```

Copy `dist/youtube-transcript-linux-amd64` to the server, e.g. `/usr/local/bin/youtube-transcript`.

## Updating from GitHub

After you have a **git clone** on the machine (laptop or LXC), you can refresh from `origin` and rebuild in one step.

### On your dev machine (macOS / Linux desktop)

```bash
cd /path/to/youtube-transcript
make update
```

Runs `git pull --ff-only` then `make build` (npm + Vite + embedded UI + native Go binary).

### On the LXC (clone lives on the guest, has Go 1.25+ and Node/npm)

**Option A — Makefile only** (install the binary yourself or you already use systemd with a fixed path):

```bash
cd /opt/youtube-transcript   # or wherever you cloned
make update-release-host
BIN=$(make -s print-release-binary)
sudo install -m 755 "$BIN" /usr/local/bin/youtube-transcript
sudo systemctl restart youtube-transcript
```

Use `make update-release-host` so the binary matches **amd64 vs arm64** automatically (`release-for-host` writes `dist/youtube-transcript-linux-<arch>`).

**Option B — One script** (pull, build, install, restart):

```bash
cd /opt/youtube-transcript
SYSTEMD_UNIT=youtube-transcript ./scripts/update-service.sh
```

Optional: `INSTALL_BIN=/usr/local/bin/youtube-transcript` (default), `GIT_BRANCH=main` to merge `origin/main` explicitly after `git fetch`. Run **without** `sudo` on the whole script so `git pull` uses your SSH keys; the script uses `sudo` only for `install` and `systemctl`.

**Option C — Build on laptop, copy binary only** (saves RAM/disk on tiny LXC): on the laptop run `git pull && make release-linux` (or `release-linux-arm64`), then `scp dist/youtube-transcript-linux-amd64 user@lxc:/tmp/` and install there.

## Configuration

| Environment variable | Meaning | Default |
|----------------------|---------|---------|
| `LISTEN_ADDR` | `host:port` for HTTP | `:8080` |

Bind to a LAN address if you do not want the app on every interface, for example:

`LISTEN_ADDR=192.168.1.50:8080`

## Run

```bash
./dist/youtube-transcript
```

Open `http://<host>:8080/`.

### Development

Terminal 1 — API + static (placeholder UI until you sync a frontend build):

```bash
make run-backend
```

Terminal 2 — Svelte dev server (**API calls are proxied in dev** via [`src/hooks.server.ts`](frontend/src/hooks.server.ts) to `http://127.0.0.1:8080`):

```bash
cd frontend && npm run dev
```

Set `TRANSCRIPT_API_ORIGIN` if your Go server uses another URL in dev.

If the browser shows **502** on `/api/...` in dev, the **hooks proxy** could not reach Go at `TRANSCRIPT_API_ORIGIN` / `http://127.0.0.1:8080` — start `make run-backend` first. If Go is running and you still get errors, check the JSON `error` field: **503** usually means YouTube/network from the API.

## API

`POST /api/transcript`

```json
{
  "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "lang": "en",
  "includeTimestamps": false
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `url` | yes | YouTube watch URL, short URL, embed path, or bare 11-character video ID |
| `lang` | no | Language code or comma-separated fallbacks (default `en`) |
| `includeTimestamps` | no | If `true`, response includes `textTimestamped`: each non-empty cue line is `[mm:ss]` or `[hh:mm:ss]` (floor of start time in seconds) followed by the caption text |

**Success `200`** JSON fields:

| Field | Description |
|-------|-------------|
| `videoId` | 11-character ID |
| `videoTitle` | Human-readable title from YouTube oEmbed (omitted if oEmbed fails) |
| `channelTitle` | Channel / author name from oEmbed (omitted if unavailable) |
| `lang`, `language`, `isGenerated` | Caption track metadata |
| `text` | Plain transcript (newline-separated cues) |
| `textTimestamped` | Present only when `includeTimestamps` was `true` |
| `snippetCount` | Number of caption segments |

Timestamps use the **start** of each cue in seconds (floored). Duration is not repeated on each line.

**Debug (trusted networks):** `GET /api/logs?limit=120` returns recent server log lines (same entries as `journald`’s `[api]` lines, capped in memory). `POST /api/logs/clear` empties that buffer. The web UI can show this under **Show server log**.

## Troubleshooting (transcript failures)

- **Log shows** `Get "https://www.youtube.com/watch?...": EOF` **or** similar before any HTTP status: YouTube closed the TCP/TLS connection early (common with minimal or HTTP/2 clients). This repo vendors [`youtube-transcript-go`](backend/third_party/youtube-transcript-go) with a **browser-like User-Agent**, **extra headers**, and **HTTP/1.1-only** transport; rebuild/restart the Go server after updates.
- **Sanity-check outbound HTTPS** from the same machine as Go:

  ```bash
  curl -fsSIL -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36" \
    "https://www.youtube.com/watch?v=dQw4w9WgXcQ" | head
  ```

  You should see `HTTP/2 200` or `HTTP/1.1 200` and headers — not immediate disconnect. If `curl` fails too, the issue is network/VPN/firewall/ISP, not this app.

- **Verify the video ID** (11 characters). Typos (`0` vs `O`, `G` vs `Q`) yield the wrong video or odd failures.

## systemd (Debian LXC)

Example unit — adjust `User` and `LISTEN_ADDR`:

```ini
[Unit]
Description=YouTube transcript helper
After=network.target

[Service]
Type=simple
Environment=LISTEN_ADDR=:8080
ExecStart=/usr/local/bin/youtube-transcript
Restart=on-failure
RestartSec=5
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

Install and enable:

```bash
sudo install -m 755 dist/youtube-transcript-linux-amd64 /usr/local/bin/youtube-transcript
sudo nano /etc/systemd/system/youtube-transcript.service
sudo systemctl daemon-reload
sudo systemctl enable --now youtube-transcript
journalctl -u youtube-transcript -f
```

## Manual checks

On a host that can reach YouTube normally:

```bash
make build
LISTEN_ADDR=:8080 ./dist/youtube-transcript
curl -sS http://127.0.0.1:8080/api/health
curl -sS -X POST http://127.0.0.1:8080/api/transcript \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","lang":"en"}' | jq .
```

Expect `snippetCount` and a long `text` field for videos with captions. Try an invalid host (`https://example.com/...`) and expect HTTP `400`.


See repository root (add your own `LICENSE` if needed).
