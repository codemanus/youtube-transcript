# Vendored `youtube-transcript-go`

This is a copy of [`github.com/rahadiangg/youtube-transcript-go`](https://github.com/rahadiangg/youtube-transcript-go) v0.1.0 with HTTP client hardening in `youtube/client.go`:

- **Browser-like `User-Agent`** and **Accept / Sec-Fetch-\* / Referer** (watch page vs InnerTube POST vs caption fetch).
- **HTTP/1.1 only** (`TLSNextProto` empty) because YouTube often resets **HTTP/2** connections for non-browser clients, which surfaces as **`EOF`** on the first `GET` of the watch URL.
- **No manual `Accept-Encoding`** so `net/http` can negotiate **gzip** and decompress correctly with a custom User-Agent.

`go.mod` uses:

`replace github.com/rahadiangg/youtube-transcript-go => ./third_party/youtube-transcript-go`

Update by replacing the tree from upstream and re-applying the changes in `youtube/client.go` (and adjusting `youtube/client_test.go` expectations if needed).
