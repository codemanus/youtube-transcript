package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func getBody(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	return rec.Code, string(body)
}

func TestStaticHandlerServesUIBuild(t *testing.T) {
	ui := fstest.MapFS{
		"index.html":          {Data: []byte("<h1>real ui</h1>")},
		"_app/immutable/a.js": {Data: []byte("js")},
	}
	h := staticHandler(ui)

	if code, body := getBody(t, h, "/"); code != http.StatusOK || !strings.Contains(body, "real ui") {
		t.Errorf("GET / = %d %q, want 200 with the UI build", code, body)
	}
	if code, body := getBody(t, h, "/_app/immutable/a.js"); code != http.StatusOK || body != "js" {
		t.Errorf("GET asset = %d %q, want 200 js", code, body)
	}
}

func TestStaticHandlerServesPlaceholderWithoutUIBuild(t *testing.T) {
	ui := fstest.MapFS{".gitkeep": {}}
	h := staticHandler(ui)

	if code, body := getBody(t, h, "/"); code != http.StatusOK || body != string(placeholderHTML) {
		t.Errorf("GET / = %d %q, want 200 with the placeholder page", code, body)
	}
	if code, _ := getBody(t, h, "/church-guide/"); code != http.StatusNotFound {
		t.Errorf("GET /church-guide/ = %d, want 404", code)
	}
}

func TestFormatTranscriptTimestamp(t *testing.T) {
	tests := []struct {
		sec  int
		want string
	}{
		{0, "[00:00]"},
		{59, "[00:59]"},
		{60, "[01:00]"},
		{3599, "[59:59]"},
		{3600, "[01:00:00]"},
		{3661, "[01:01:01]"},
	}
	for _, tt := range tests {
		if got := formatTranscriptTimestamp(tt.sec); got != tt.want {
			t.Errorf("formatTranscriptTimestamp(%d) = %q, want %q", tt.sec, got, tt.want)
		}
	}
}
