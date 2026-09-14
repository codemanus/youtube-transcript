package transcriptmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codychambers/youtube-transcript/backend/internal/churchguideapi"
	"github.com/codychambers/youtube-transcript/backend/internal/transcriptapi"
)

// recordingService plays the transcript service contract, recording the last
// request it received and returning whatever respond computes from it.
type recordingService struct {
	mu      sync.Mutex
	got     transcriptapi.Request
	respond func(transcriptapi.Request) transcriptapi.Response
}

func (s *recordingService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req transcriptapi.Request
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	s.got = req
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.respond(req))
}

func (s *recordingService) request() transcriptapi.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.got
}

// newTestSession starts svc behind an httptest server, builds the MCP server
// from it, and connects a real MCP client to it over an in-memory transport.
func newTestSession(t *testing.T, svc http.Handler) *mcp.ClientSession {
	t.Helper()

	httpSrv := httptest.NewServer(svc)
	t.Cleanup(httpSrv.Close)

	server := NewServer(httpSrv.URL, httpSrv.Client())

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Wait() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func textContent(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("want 1 content block, got %d: %+v", len(res.Content), res.Content)
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content block is %T, want *mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

func TestListTools(t *testing.T) {
	cs := newTestSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "the transcript service should not be called while listing tools", http.StatusInternalServerError)
	}))

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) != 2 {
		t.Fatalf("want 2 tools, got %d", len(res.Tools))
	}

	byName := make(map[string]*mcp.Tool)
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
	}

	transcriptTool, ok := byName[toolName]
	if !ok {
		t.Fatalf("tools missing %q", toolName)
	}
	schema, ok := transcriptTool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("InputSchema is %T, want map[string]any", transcriptTool.InputSchema)
	}
	required, _ := schema["required"].([]any)
	if len(required) != 1 || required[0] != "url" {
		t.Errorf("required = %v, want [\"url\"]", required)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties is %T, want map[string]any", schema["properties"])
	}
	for _, name := range []string{"url", "lang", "includeTimestamps"} {
		if _, ok := props[name]; !ok {
			t.Errorf("properties missing %q", name)
		}
	}

	guideTool, ok := byName[churchGuideToolName]
	if !ok {
		t.Fatalf("tools missing %q", churchGuideToolName)
	}
	guideSchema, ok := guideTool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("InputSchema is %T, want map[string]any", guideTool.InputSchema)
	}
	if required, _ := guideSchema["required"].([]any); len(required) != 0 {
		t.Errorf("%s required = %v, want none", churchGuideToolName, required)
	}
}

func TestCallTool(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]any
		respTitle     string
		respChannel   string
		respGenerated bool
		wantReqLang   string
		wantReqTS     bool
		wantText      string
	}{
		{
			name:          "timestamps default to on",
			args:          map[string]any{"url": "https://youtu.be/abc12345678"},
			respTitle:     "Sunday Sermon",
			respChannel:   "Two Cities Church",
			respGenerated: false,
			wantReqLang:   "",
			wantReqTS:     true,
			wantText: "Video ID: abc12345678\n" +
				"Title: Sunday Sermon\n" +
				"Channel: Two Cities Church\n" +
				"Language: English (en)\n" +
				"Auto-generated: no\n" +
				"URL: https://youtu.be/abc12345678\n" +
				"\n" +
				"[00:00] line one\n[00:05] line two",
		},
		{
			name:          "includeTimestamps false sends false and returns plain text",
			args:          map[string]any{"url": "https://youtu.be/abc12345678", "includeTimestamps": false},
			respTitle:     "Sunday Sermon",
			respChannel:   "Two Cities Church",
			respGenerated: false,
			wantReqLang:   "",
			wantReqTS:     false,
			wantText: "Video ID: abc12345678\n" +
				"Title: Sunday Sermon\n" +
				"Channel: Two Cities Church\n" +
				"Language: English (en)\n" +
				"Auto-generated: no\n" +
				"URL: https://youtu.be/abc12345678\n" +
				"\n" +
				"line one\nline two",
		},
		{
			name:          "lang passed through unchanged",
			args:          map[string]any{"url": "https://youtu.be/abc12345678", "lang": "es,en"},
			respTitle:     "Sunday Sermon",
			respChannel:   "Two Cities Church",
			respGenerated: false,
			wantReqLang:   "es,en",
			wantReqTS:     true,
			wantText: "Video ID: abc12345678\n" +
				"Title: Sunday Sermon\n" +
				"Channel: Two Cities Church\n" +
				"Language: English (en)\n" +
				"Auto-generated: no\n" +
				"URL: https://youtu.be/abc12345678\n" +
				"\n" +
				"[00:00] line one\n[00:05] line two",
		},
		{
			name:          "missing title and channel omit those header lines",
			args:          map[string]any{"url": "https://youtu.be/abc12345678"},
			respTitle:     "",
			respChannel:   "",
			respGenerated: false,
			wantReqLang:   "",
			wantReqTS:     true,
			wantText: "Video ID: abc12345678\n" +
				"Language: English (en)\n" +
				"Auto-generated: no\n" +
				"URL: https://youtu.be/abc12345678\n" +
				"\n" +
				"[00:00] line one\n[00:05] line two",
		},
		{
			name:          "auto-generated renders yes",
			args:          map[string]any{"url": "https://youtu.be/abc12345678"},
			respTitle:     "Sunday Sermon",
			respChannel:   "Two Cities Church",
			respGenerated: true,
			wantReqLang:   "",
			wantReqTS:     true,
			wantText: "Video ID: abc12345678\n" +
				"Title: Sunday Sermon\n" +
				"Channel: Two Cities Church\n" +
				"Language: English (en)\n" +
				"Auto-generated: yes\n" +
				"URL: https://youtu.be/abc12345678\n" +
				"\n" +
				"[00:00] line one\n[00:05] line two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &recordingService{
				respond: func(req transcriptapi.Request) transcriptapi.Response {
					resp := transcriptapi.Response{
						VideoID:      "abc12345678",
						VideoTitle:   tt.respTitle,
						ChannelTitle: tt.respChannel,
						Lang:         "en",
						Language:     "English",
						IsGenerated:  tt.respGenerated,
						Text:         "line one\nline two",
						SnippetCount: 2,
					}
					if req.IncludeTimestamps {
						ts := "[00:00] line one\n[00:05] line two"
						resp.TextTimestamped = &ts
					}
					return resp
				},
			}

			cs := newTestSession(t, svc)

			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      toolName,
				Arguments: tt.args,
			})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if res.IsError {
				t.Fatalf("CallTool returned isError: %s", textContent(t, res))
			}

			gotReq := svc.request()
			if gotReq.URL != tt.args["url"] {
				t.Errorf("request URL = %q, want %q", gotReq.URL, tt.args["url"])
			}
			if gotReq.Lang != tt.wantReqLang {
				t.Errorf("request Lang = %q, want %q", gotReq.Lang, tt.wantReqLang)
			}
			if gotReq.IncludeTimestamps != tt.wantReqTS {
				t.Errorf("request IncludeTimestamps = %v, want %v", gotReq.IncludeTimestamps, tt.wantReqTS)
			}

			if got := textContent(t, res); got != tt.wantText {
				t.Errorf("result text =\n%s\nwant\n%s", got, tt.wantText)
			}
		})
	}
}

func TestCallToolServiceError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantMsg string
	}{
		{
			name:    "400 with error body",
			status:  http.StatusBadRequest,
			body:    `{"error":"url is required"}`,
			wantMsg: "transcript service returned 400: url is required",
		},
		{
			name:    "403 with error body",
			status:  http.StatusForbidden,
			body:    `{"error":"video is age-restricted; transcripts need auth"}`,
			wantMsg: "transcript service returned 403: video is age-restricted; transcripts need auth",
		},
		{
			name:    "404 with error body",
			status:  http.StatusNotFound,
			body:    `{"error":"no transcript for that language (or none available)"}`,
			wantMsg: "transcript service returned 404: no transcript for that language (or none available)",
		},
		{
			name:    "429 with error body",
			status:  http.StatusTooManyRequests,
			body:    `{"error":"rate limit exceeded; try again shortly"}`,
			wantMsg: "transcript service returned 429: rate limit exceeded; try again shortly",
		},
		{
			name:    "503 with error body",
			status:  http.StatusServiceUnavailable,
			body:    `{"error":"could not retrieve transcript (network error or unexpected response from YouTube)"}`,
			wantMsg: "transcript service returned 503: could not retrieve transcript (network error or unexpected response from YouTube)",
		},
		{
			name:    "504 with error body",
			status:  http.StatusGatewayTimeout,
			body:    `{"error":"request timed out"}`,
			wantMsg: "transcript service returned 504: request timed out",
		},
		{
			name:    "non-2xx with unparseable body still reports the status",
			status:  http.StatusInternalServerError,
			body:    "<html>upstream broke</html>",
			wantMsg: "transcript service returned 500: <html>upstream broke</html>",
		},
		{
			name:    "non-2xx with empty body still reports the status",
			status:  http.StatusBadGateway,
			body:    "",
			wantMsg: "transcript service returned 502",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newTestSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))

			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      toolName,
				Arguments: map[string]any{"url": "https://youtu.be/abc12345678"},
			})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if !res.IsError {
				t.Fatalf("want isError, got success: %s", textContent(t, res))
			}
			if got := textContent(t, res); got != tt.wantMsg {
				t.Errorf("error text = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestCallToolMalformedResponse(t *testing.T) {
	cs := newTestSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: map[string]any{"url": "https://youtu.be/abc12345678"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want isError, got success: %s", textContent(t, res))
	}
	if got := textContent(t, res); !strings.Contains(got, "unreadable response") {
		t.Errorf("error text = %q, want it to mention an unreadable response", got)
	}
}

func TestCallToolUnreachable(t *testing.T) {
	// Start and immediately close a real server so baseURL is a valid-looking
	// address that nothing is listening on.
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := httpSrv.URL
	httpSrv.Close()

	server := NewServer(baseURL, http.DefaultClient)

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Wait() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: map[string]any{"url": "https://youtu.be/abc12345678"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want isError, got success: %s", textContent(t, res))
	}

	got := textContent(t, res)
	if !strings.Contains(got, baseURL) {
		t.Errorf("error text %q does not name the configured address %q", got, baseURL)
	}
	if !strings.Contains(got, "LAN/VPN") {
		t.Errorf("error text %q does not hint that the Mac may be off the LAN/VPN", got)
	}
}

func TestCallChurchGuideTool(t *testing.T) {
	cs := newTestSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/church-guide" {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(churchguideapi.Response{
			Title:         "Until Now",
			WeekStartDate: "2026-09-13T00:00:00.000Z",
			Text:          "line one\nline two",
		})
	}))

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      churchGuideToolName,
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("CallTool returned isError: %s", textContent(t, res))
	}

	want := "Title: Until Now\n" +
		"Week of: 2026-09-13T00:00:00.000Z\n" +
		"\n" +
		"line one\nline two"
	if got := textContent(t, res); got != want {
		t.Errorf("result text =\n%s\nwant\n%s", got, want)
	}
}

func TestCallChurchGuideToolServiceError(t *testing.T) {
	cs := newTestSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"groups portal session expired or invalid"}`))
	}))

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      churchGuideToolName,
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want isError, got success: %s", textContent(t, res))
	}
	want := "church guide service returned 401: groups portal session expired or invalid"
	if got := textContent(t, res); got != want {
		t.Errorf("error text = %q, want %q", got, want)
	}
}

func TestCallChurchGuideToolUnreachable(t *testing.T) {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := httpSrv.URL
	httpSrv.Close()

	server := NewServer(baseURL, http.DefaultClient)

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Wait() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      churchGuideToolName,
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want isError, got success: %s", textContent(t, res))
	}

	got := textContent(t, res)
	if !strings.Contains(got, baseURL) {
		t.Errorf("error text %q does not name the configured address %q", got, baseURL)
	}
	if !strings.Contains(got, "LAN/VPN") {
		t.Errorf("error text %q does not hint that the Mac may be off the LAN/VPN", got)
	}
}
