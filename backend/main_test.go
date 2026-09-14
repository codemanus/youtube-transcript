package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/codychambers/youtube-transcript/backend/internal/portalcookie"
	"github.com/codychambers/youtube-transcript/backend/internal/transcriptapi"
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

func TestBasicAuthMiddleware(t *testing.T) {
	t.Setenv(churchGuideAdminPasswordEnvVar, "s3cret")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := basicAuthMiddleware(inner)

	t.Run("no credentials", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/church-guide/admin/status", nil))

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
		if rec.Header().Get("WWW-Authenticate") == "" {
			t.Error("missing WWW-Authenticate header")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/church-guide/admin/status", nil)
		req.SetBasicAuth(churchGuideAdminUsername, "wrong")
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("correct credentials", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/church-guide/admin/status", nil)
		req.SetBasicAuth(churchGuideAdminUsername, "s3cret")
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (request should reach the wrapped handler)", rec.Code)
		}
	})
}

func TestChurchGuideAdminSaveCookieRejectsNonJSONContentType(t *testing.T) {
	t.Setenv(portalcookie.PathEnvVar, filepath.Join(t.TempDir(), "cookie"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("portal should not be contacted when Content-Type isn't application/json")
	}))
	defer srv.Close()

	h := churchGuideAdminSaveCookieHandler(srv.URL, srv.Client())

	// A cross-site <form> POST can only produce these Content-Types (never
	// application/json), so rejecting them is the CSRF defense — this test
	// pins that a same-site JSON client isn't accidentally caught by it too.
	for _, ct := range []string{"", "application/x-www-form-urlencoded", "text/plain"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/church-guide/admin/cookie", strings.NewReader(`{"cookie":"connect.sid=x"}`))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %q: status = %d, want %d", ct, rec.Code, http.StatusUnsupportedMediaType)
		}
	}
}

func TestChurchGuideAdminSaveCookieRejectsEmptyWithoutValidating(t *testing.T) {
	t.Setenv(portalcookie.PathEnvVar, filepath.Join(t.TempDir(), "cookie"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("portal should not be contacted for an empty/whitespace-only candidate")
	}))
	defer srv.Close()

	h := churchGuideAdminSaveCookieHandler(srv.URL, srv.Client())

	for _, body := range []string{`{"cookie":""}`, `{"cookie":"   "}`} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/church-guide/admin/cookie", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, rec.Code)
		}
	}
}

func TestChurchGuideAdminSaveCookieRejectedByPortal(t *testing.T) {
	t.Setenv(portalcookie.PathEnvVar, filepath.Join(t.TempDir(), "cookie"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	h := churchGuideAdminSaveCookieHandler(srv.URL, srv.Client())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/church-guide/admin/cookie", strings.NewReader(`{"cookie":"connect.sid=bad"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	var resp transcriptapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if !strings.Contains(resp.Error, "rejected") {
		t.Errorf("error = %q, want it to say the portal rejected the cookie", resp.Error)
	}

	if _, _, configured, err := portalcookie.Read(); err != nil {
		t.Fatalf("Read: %v", err)
	} else if configured {
		t.Error("cookie file should not be written after a portal rejection")
	}
}

func TestChurchGuideAdminSaveCookieUnreachablePortal(t *testing.T) {
	t.Setenv(portalcookie.PathEnvVar, filepath.Join(t.TempDir(), "cookie"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base := srv.URL
	srv.Close() // unreachable by the time the handler calls it

	h := churchGuideAdminSaveCookieHandler(base, http.DefaultClient)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/church-guide/admin/cookie", strings.NewReader(`{"cookie":"connect.sid=x"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var resp transcriptapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if strings.Contains(resp.Error, "rejected") {
		t.Errorf("error = %q, want a distinct 'try again' message, not a rejection message", resp.Error)
	}

	if _, _, configured, err := portalcookie.Read(); err != nil {
		t.Fatalf("Read: %v", err)
	} else if configured {
		t.Error("cookie file should not be written when the portal is unreachable")
	}
}

func TestChurchGuideAdminSaveCookieValidCandidate(t *testing.T) {
	t.Setenv(portalcookie.PathEnvVar, filepath.Join(t.TempDir(), "cookie"))

	const okJSON = `{"title":"Until Now","weekStartDate":"2026-09-13T00:00:00.000Z","pdfObjectKey":"","editedContent":""}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(okJSON))
	}))
	defer srv.Close()

	h := churchGuideAdminSaveCookieHandler(srv.URL, srv.Client())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/church-guide/admin/cookie", strings.NewReader(`{"cookie":"connect.sid=good"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var resp churchGuideAdminStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Configured {
		t.Error("Configured = false in the save response, want true")
	}
	if resp.UpdatedAt == nil {
		t.Error("UpdatedAt = nil in the save response, want a timestamp")
	}

	value, _, configured, err := portalcookie.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !configured || value != "connect.sid=good" {
		t.Errorf("Read() = (%q, configured=%v), want (\"connect.sid=good\", true)", value, configured)
	}

	// A subsequent status call should reflect the save too.
	statusRec := httptest.NewRecorder()
	handleChurchGuideAdminStatus(statusRec, httptest.NewRequest(http.MethodGet, "/api/church-guide/admin/status", nil))
	var statusResp churchGuideAdminStatusResponse
	if err := json.NewDecoder(statusRec.Body).Decode(&statusResp); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if !statusResp.Configured || statusResp.UpdatedAt == nil {
		t.Errorf("status after save = %+v, want configured with an UpdatedAt", statusResp)
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
