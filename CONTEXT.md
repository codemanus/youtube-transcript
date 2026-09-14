# youtube-transcript

A small LAN/VPN utility that fetches text content from sources Cody can't easily script access to otherwise (YouTube captions, login-gated portals), for use as input to downstream automation (Cowork pipelines).

## Language

**Transcript**:
The caption text for a single YouTube video, optionally timestamped, fetched via `POST /api/transcript` or the `get_youtube_transcript` MCP tool.

**Church Guide**:
The weekly Community Group discussion document (intro questions, Scripture reference, discussion questions) for Two Cities Church, sourced from the Groups Portal. This repo's responsibility is fetching and extracting it (text primary, PDF backup) — writing it into the Obsidian vault note is a downstream (Cowork) concern, not this repo's.
_Avoid_: CG Guide, Weekly Guide, This Week PDF — use "Church Guide" to match the vault/Cowork pipeline's existing vocabulary.

**Groups Portal**:
The login-gated web app at `mycgconnect.com`, used by Two Cities Church CG leaders for weekly resources. Exposes a `/this-week` page with Guide/Notes/PDF views; the Church Guide is fetched from its PDF download endpoint. Authenticated via a long-lived session cookie, not per-request login.
_Avoid_: the portal (on its own, when another portal could be meant), CG Connect.
