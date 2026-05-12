package youtubeoembed

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Match the vendored YouTube client so oEmbed is less likely to be blocked the same way.
const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

type payload struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// Fetch returns the video title and channel (author) name from YouTube oEmbed.
// On any error or non-200, returns empty strings (caller should not fail the transcript).
func Fetch(ctx context.Context, videoID string) (title, channel string) {
	watch := "https://www.youtube.com/watch?v=" + videoID
	u := "https://www.youtube.com/oembed?format=json&url=" + url.QueryEscape(watch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", ""
	}
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		return "", ""
	}
	return p.Title, p.AuthorName
}
