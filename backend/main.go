package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codychambers/youtube-transcript/backend/internal/apilog"
	"github.com/codychambers/youtube-transcript/backend/internal/videoid"
	"github.com/codychambers/youtube-transcript/backend/internal/youtubeoembed"
	youtube "github.com/rahadiangg/youtube-transcript-go/youtube"
)

//go:embed all:static
var staticFS embed.FS

const (
	maxBodyBytes       = 16 * 1024
	defaultListen      = ":8080"
	requestTimeout     = 45 * time.Second
	rateLimitPerMinute = 30
)

type transcriptRequest struct {
	URL               string `json:"url"`
	Lang              string `json:"lang"`
	IncludeTimestamps bool   `json:"includeTimestamps"`
}

type transcriptResponse struct {
	VideoID         string  `json:"videoId"`
	VideoTitle      string  `json:"videoTitle,omitempty"`
	ChannelTitle    string  `json:"channelTitle,omitempty"`
	Lang            string  `json:"lang"`
	Language        string  `json:"language"`
	IsGenerated     bool    `json:"isGenerated"`
	Text            string  `json:"text"`
	TextTimestamped *string `json:"textTimestamped,omitempty"`
	SnippetCount    int     `json:"snippetCount"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type logsResponse struct {
	Entries []apilog.Entry `json:"entries"`
}

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = defaultListen
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/logs", handleLogs)
	mux.HandleFunc("POST /api/logs/clear", handleLogsClear)
	mux.HandleFunc("POST /api/transcript", handleTranscript)

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("static embed: %v", err)
	}
	fileServer := http.FileServer(http.FS(static))
	mux.Handle("/", fileServer)

	srv := &http.Server{
		Addr:              addr,
		Handler:           logRequest(recoverMiddleware(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	apilog.Info("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	limit := 200
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	_ = json.NewEncoder(w).Encode(logsResponse{Entries: apilog.Snapshot(limit)})
}

func handleLogsClear(w http.ResponseWriter, r *http.Request) {
	apilog.Clear()
	apilog.Info("log buffer cleared (via POST /api/logs/clear)")
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"ok":true}`)
}

func handleTranscript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !checkRateLimit(r.RemoteAddr) {
		apilog.Warn("rate limited %s", r.RemoteAddr)
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded; try again shortly")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req transcriptRequest
	if err := dec.Decode(&req); err != nil {
		apilog.Warn("transcript bad json from %s: %v", r.RemoteAddr, err)
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	videoID, err := videoid.FromInput(req.URL)
	if err != nil {
		if errors.Is(err, videoid.ErrEmpty) {
			apilog.Warn("transcript missing url from %s", r.RemoteAddr)
			writeError(w, http.StatusBadRequest, "url is required")
			return
		}
		apilog.Warn("transcript invalid url from %s: %q", r.RemoteAddr, req.URL)
		writeError(w, http.StatusBadRequest, "not a valid YouTube URL or video id")
		return
	}

	langs := languageCodes(req.Lang)
	apilog.Info("transcript fetch start video=%s langs=%v client=%s", videoID, langs, r.RemoteAddr)

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	type oembedOut struct {
		title, channel string
	}
	oembedCh := make(chan oembedOut, 1)
	go func() {
		oeCtx, oeCancel := context.WithTimeout(ctx, 8*time.Second)
		defer oeCancel()
		t, c := youtubeoembed.Fetch(oeCtx, videoID)
		oembedCh <- oembedOut{t, c}
	}()

	type result struct {
		ft  *youtube.FetchedTranscript
		err error
	}
	ch := make(chan result, 1)
	go func() {
		api := youtube.NewYouTubeTranscriptApi()
		ft, err := api.Fetch(videoID, langs, false)
		ch <- result{ft, err}
	}()

	var ft *youtube.FetchedTranscript
	select {
	case <-ctx.Done():
		apilog.Error("transcript timeout video=%s client=%s", videoID, r.RemoteAddr)
		writeError(w, http.StatusGatewayTimeout, "request timed out")
		return
	case res := <-ch:
		if res.err != nil {
			status, msg := mapYouTubeError(res.err)
			apilog.Error("transcript failed video=%s http=%d err=%v public=%q", videoID, status, res.err, msg)
			writeError(w, status, msg)
			return
		}
		ft = res.ft
	}

	var videoTitle, channelTitle string
	select {
	case oe := <-oembedCh:
		videoTitle, channelTitle = oe.title, oe.channel
	case <-time.After(8 * time.Second):
	}

	text := joinTranscriptText(ft.Snippets)
	resp := transcriptResponse{
		VideoID:      ft.VideoID,
		VideoTitle:   videoTitle,
		ChannelTitle: channelTitle,
		Lang:         ft.LanguageCode,
		Language:     ft.Language,
		IsGenerated:  ft.IsGenerated,
		Text:         text,
		SnippetCount: len(ft.Snippets),
	}
	if req.IncludeTimestamps {
		ts := joinTranscriptTextTimestamped(ft.Snippets)
		resp.TextTimestamped = &ts
	}

	apilog.Info("transcript ok video=%s lines=%d lang=%s generated=%v title=%q", videoID, len(ft.Snippets), ft.LanguageCode, ft.IsGenerated, videoTitle)
	_ = json.NewEncoder(w).Encode(resp)
}

func languageCodes(lang string) []string {
	s := strings.TrimSpace(strings.ToLower(lang))
	if s == "" {
		return []string{"en"}
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"en"}
	}
	return out
}

func joinTranscriptTextTimestamped(snippets []youtube.FetchedTranscriptSnippet) string {
	var b strings.Builder
	for _, sn := range snippets {
		line := strings.TrimSpace(sn.Text)
		if line == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		sec := int(math.Floor(sn.Start + 1e-9))
		b.WriteString(formatTranscriptTimestamp(sec))
		b.WriteByte(' ')
		b.WriteString(line)
	}
	return b.String()
}

// formatTranscriptTimestamp uses [mm:ss] below one hour, else [hh:mm:ss] (floor seconds).
func formatTranscriptTimestamp(sec int) string {
	if sec < 0 {
		sec = 0
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("[%02d:%02d:%02d]", h, m, s)
	}
	return fmt.Sprintf("[%02d:%02d]", m, s)
}

func joinTranscriptText(snippets []youtube.FetchedTranscriptSnippet) string {
	var b strings.Builder
	for _, sn := range snippets {
		line := strings.TrimSpace(sn.Text)
		if line == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	return b.String()
}

func mapYouTubeError(err error) (int, string) {
	var (
		noTranscript  *youtube.NoTranscriptFound
		transcriptOff *youtube.TranscriptsDisabled
		unavail       *youtube.VideoUnavailable
		unplayable    *youtube.VideoUnplayable
		age           *youtube.AgeRestricted
		ipBlocked     *youtube.IpBlocked
		reqBlocked    *youtube.RequestBlocked
		badID         *youtube.InvalidVideoID
	)
	switch {
	case errors.As(err, &noTranscript):
		return http.StatusNotFound, "no transcript for that language (or none available)"
	case errors.As(err, &transcriptOff):
		return http.StatusNotFound, "captions are disabled for this video"
	case errors.As(err, &unavail):
		return http.StatusNotFound, "video is unavailable"
	case errors.As(err, &unplayable):
		return http.StatusNotFound, "video cannot be played (may be restricted)"
	case errors.As(err, &age):
		return http.StatusForbidden, "video is age-restricted; transcripts need auth"
	case errors.As(err, &ipBlocked) || errors.As(err, &reqBlocked):
		return http.StatusTooManyRequests, "YouTube blocked this request (try again later)"
	case errors.As(err, &badID):
		return http.StatusBadRequest, "invalid video id"
	default:
		// Avoid 502 here: browsers and devtools label it "Bad Gateway", which reads like a broken reverse proxy.
		return http.StatusServiceUnavailable, "could not retrieve transcript (network error or unexpected response from YouTube)"
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/logs", "/api/health", "/api/logs/clear":
			// avoid noise from log polling, health checks, and clear
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") {
				apilog.Info("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				apilog.Error("panic: %v", rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type minuteLimiter struct {
	mu   sync.Mutex
	ipAt map[string][]time.Time
}

var limiter = &minuteLimiter{ipAt: make(map[string][]time.Time)}

func checkRateLimit(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	times := limiter.ipAt[host]
	var kept []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rateLimitPerMinute {
		return false
	}
	kept = append(kept, now)
	limiter.ipAt[host] = kept
	return true
}
