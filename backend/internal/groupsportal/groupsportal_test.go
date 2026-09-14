package groupsportal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const currentJSON = `{
	"title": "Until Now",
	"weekStartDate": "2026-09-13T00:00:00.000Z",
	"pdfObjectKey": "public/weekly-resources/1789314491895.pdf",
	"editedContent": "[{\"pageNum\":1,\"blocks\":[{\"text\":\"UNTIL NOW\",\"isSectionHeader\":false},{\"text\":\"Intro Questions\",\"isSectionHeader\":true},{\"text\":\"1. Icebreaker?\",\"isSectionHeader\":false},{\"text\":\"   \",\"isSectionHeader\":false}]}]"
}`

func TestFetchGuide(t *testing.T) {
	var gotPath, gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(currentJSON))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "connect.sid=abc123", srv.Client())
	guide, err := c.FetchGuide(context.Background())
	if err != nil {
		t.Fatalf("FetchGuide: %v", err)
	}

	if gotPath != "/api/weekly-resources/current" {
		t.Errorf("request path = %q, want /api/weekly-resources/current", gotPath)
	}
	if gotCookie != "connect.sid=abc123" {
		t.Errorf("Cookie header = %q, want %q", gotCookie, "connect.sid=abc123")
	}

	if guide.Title != "Until Now" {
		t.Errorf("Title = %q, want %q", guide.Title, "Until Now")
	}
	if guide.WeekStartDate != "2026-09-13T00:00:00.000Z" {
		t.Errorf("WeekStartDate = %q, want %q", guide.WeekStartDate, "2026-09-13T00:00:00.000Z")
	}

	wantText := "UNTIL NOW\n\nIntro Questions\n1. Icebreaker?"
	if guide.Text != wantText {
		t.Errorf("Text =\n%q\nwant\n%q", guide.Text, wantText)
	}
}

func TestFetchGuideSessionExpired(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
		}))

		c := NewClient(srv.URL, "connect.sid=stale", srv.Client())
		_, err := c.FetchGuide(context.Background())
		srv.Close()

		if !errors.Is(err, ErrSessionExpired) {
			t.Errorf("status %d: err = %v, want ErrSessionExpired", status, err)
		}
	}
}

func TestFetchGuideUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base := srv.URL
	srv.Close()

	c := NewClient(base, "connect.sid=abc", http.DefaultClient)
	_, err := c.FetchGuide(context.Background())
	if err == nil {
		t.Fatal("FetchGuide: want error for unreachable portal, got nil")
	}
	if errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, want a non-session error for an unreachable portal", err)
	}
}

func TestFetchGuideMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "connect.sid=abc", srv.Client())
	_, err := c.FetchGuide(context.Background())
	if err == nil {
		t.Fatal("FetchGuide: want error for malformed JSON, got nil")
	}
}

func TestFetchPDF(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 fake bytes")
	var gotPath, gotCookie string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/weekly-resources/current":
			_, _ = w.Write([]byte(currentJSON))
		case "/api/weekly-resources/download/1789314491895.pdf":
			gotPath = r.URL.Path
			gotCookie = r.Header.Get("Cookie")
			_, _ = w.Write(pdfBytes)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "connect.sid=abc123", srv.Client())
	data, err := c.FetchPDF(context.Background())
	if err != nil {
		t.Fatalf("FetchPDF: %v", err)
	}

	if gotPath != "/api/weekly-resources/download/1789314491895.pdf" {
		t.Errorf("download path = %q, want /api/weekly-resources/download/1789314491895.pdf", gotPath)
	}
	if gotCookie != "connect.sid=abc123" {
		t.Errorf("Cookie header = %q, want %q", gotCookie, "connect.sid=abc123")
	}
	if string(data) != string(pdfBytes) {
		t.Errorf("data = %q, want %q", data, pdfBytes)
	}
}

func TestFetchGuideEmptyEditedContent(t *testing.T) {
	const emptyContentJSON = `{
		"title": "Until Now",
		"weekStartDate": "2026-09-13T00:00:00.000Z",
		"pdfObjectKey": "public/weekly-resources/1789314491895.pdf",
		"editedContent": ""
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(emptyContentJSON))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "connect.sid=abc123", srv.Client())
	guide, err := c.FetchGuide(context.Background())
	if err != nil {
		t.Fatalf("FetchGuide: want no error for empty editedContent (PDF is still the backup), got %v", err)
	}
	if guide.Text != "" {
		t.Errorf("Text = %q, want empty", guide.Text)
	}
	if guide.Title != "Until Now" {
		t.Errorf("Title = %q, want %q", guide.Title, "Until Now")
	}
}

func TestFetchPDFMissingObjectKey(t *testing.T) {
	const noKeyJSON = `{"title":"Until Now","weekStartDate":"2026-09-13T00:00:00.000Z","pdfObjectKey":"","editedContent":"[]"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/weekly-resources/current" {
			_, _ = w.Write([]byte(noKeyJSON))
			return
		}
		t.Errorf("unexpected download request for path %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "connect.sid=abc123", srv.Client())
	_, err := c.FetchPDF(context.Background())
	if err == nil {
		t.Fatal("FetchPDF: want error when pdfObjectKey is missing, got nil")
	}
}

func TestFetchPDFOversized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/weekly-resources/current":
			_, _ = w.Write([]byte(currentJSON))
		case "/api/weekly-resources/download/1789314491895.pdf":
			buf := make([]byte, maxPDFBytes+1)
			_, _ = w.Write(buf)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "connect.sid=abc123", srv.Client())
	_, err := c.FetchPDF(context.Background())
	if err == nil {
		t.Fatal("FetchPDF: want error for an oversized response, got nil")
	}
}

func TestFetchPDFSessionExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "connect.sid=stale", srv.Client())
	_, err := c.FetchPDF(context.Background())
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, want ErrSessionExpired", err)
	}
}
