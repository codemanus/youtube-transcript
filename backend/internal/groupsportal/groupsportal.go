// Package groupsportal fetches the current week's Church Guide from the
// Groups Portal (mycgconnect.com). It authenticates with a session cookie
// value (not credentials) supplied by the caller; SessionCookieFromEnv reads
// the cookie Cody refreshes by hand (see docs/adr/0001).
package groupsportal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

// DefaultBaseURL is the Groups Portal's public address.
const DefaultBaseURL = "https://mycgconnect.com"

// SessionCookieEnvVar names the environment variable holding the Groups
// Portal session cookie (the literal Cookie header value, e.g.
// "connect.sid=s%3A...").
const SessionCookieEnvVar = "GROUPS_PORTAL_SESSION_COOKIE"

// ErrSessionExpired is returned when the portal rejects the configured
// session cookie (401/403), meaning it has expired or is invalid.
var ErrSessionExpired = errors.New("groups portal session expired or invalid")

// ErrNoPDF is returned when the current week's resource has no PDF on file.
var ErrNoPDF = errors.New("groups portal: no PDF available for the current week")

// Size caps on external responses we read fully into memory, mirroring the
// youtubeoembed package's bounded read of an external response.
const (
	maxCurrentBytes = 2 * 1024 * 1024  // JSON metadata + editedContent
	maxPDFBytes     = 25 * 1024 * 1024 // a Church Guide PDF is a few pages
)

// SessionCookieFromEnv reads and validates GROUPS_PORTAL_SESSION_COOKIE. It
// returns an error naming the variable when it is unset.
func SessionCookieFromEnv() (string, error) {
	v := strings.TrimSpace(os.Getenv(SessionCookieEnvVar))
	if v == "" {
		return "", fmt.Errorf("%s is not set; copy the Cookie header value from an authenticated browser session at %s", SessionCookieEnvVar, DefaultBaseURL)
	}
	return v, nil
}

// Client fetches the Church Guide from a Groups Portal instance.
type Client struct {
	baseURL string
	cookie  string
	http    *http.Client
}

// NewClient builds a Client for the portal at baseURL, authenticating every
// request with cookie (the literal Cookie header value).
func NewClient(baseURL, cookie string, client *http.Client) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		cookie:  cookie,
		http:    client,
	}
}

// Guide is the current week's Church Guide.
type Guide struct {
	Title         string
	WeekStartDate string
	Text          string
}

// currentResource mirrors the Groups Portal's GET /api/weekly-resources/current response.
type currentResource struct {
	Title         string `json:"title"`
	WeekStartDate string `json:"weekStartDate"`
	PDFObjectKey  string `json:"pdfObjectKey"`
	EditedContent string `json:"editedContent"`
}

type editedPage struct {
	Blocks []editedBlock `json:"blocks"`
}

type editedBlock struct {
	Text            string `json:"text"`
	IsSectionHeader bool   `json:"isSectionHeader"`
}

// FetchGuide fetches the current week's title, week start date, and text
// (built from the portal's own structured editedContent).
func (c *Client) FetchGuide(ctx context.Context) (*Guide, error) {
	res, err := c.fetchCurrent(ctx)
	if err != nil {
		return nil, err
	}

	text, err := buildText(res.EditedContent)
	if err != nil {
		return nil, fmt.Errorf("parse guide content: %w", err)
	}

	return &Guide{
		Title:         res.Title,
		WeekStartDate: res.WeekStartDate,
		Text:          text,
	}, nil
}

// FetchPDF fetches the raw PDF bytes for the current week's Church Guide.
func (c *Client) FetchPDF(ctx context.Context) ([]byte, error) {
	res, err := c.fetchCurrent(ctx)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(res.PDFObjectKey) == "" {
		return nil, ErrNoPDF
	}
	filename := path.Base(res.PDFObjectKey)

	req, err := c.newRequest(ctx, "/api/weekly-resources/download/"+filename)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch guide pdf: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPDFBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read guide pdf: %w", err)
	}
	if len(body) > maxPDFBytes {
		return nil, fmt.Errorf("guide pdf exceeds %d bytes", maxPDFBytes)
	}

	if err := statusError(resp.StatusCode, body); err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) fetchCurrent(ctx context.Context) (*currentResource, error) {
	req, err := c.newRequest(ctx, "/api/weekly-resources/current")
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch current guide: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCurrentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read current guide response: %w", err)
	}
	if len(body) > maxCurrentBytes {
		return nil, fmt.Errorf("current guide response exceeds %d bytes", maxCurrentBytes)
	}

	if err := statusError(resp.StatusCode, body); err != nil {
		return nil, err
	}

	var res currentResource
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("decode current guide response: %w", err)
	}

	return &res, nil
}

func (c *Client) newRequest(ctx context.Context, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", path, err)
	}
	req.Header.Set("Cookie", c.cookie)
	return req, nil
}

func statusError(status int, body []byte) error {
	if status == http.StatusOK {
		return nil
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return ErrSessionExpired
	}
	return fmt.Errorf("groups portal returned %d: %s", status, strings.TrimSpace(string(body)))
}

// buildText joins editedContent's page blocks into plain text, separating
// each section header from what precedes it with a blank line. Empty
// editedContent (the week's guide hasn't been authored yet) yields empty
// text rather than an error — the PDF may still be available as a backup.
func buildText(rawEditedContent string) (string, error) {
	trimmed := strings.TrimSpace(rawEditedContent)
	if trimmed == "" {
		return "", nil
	}

	var pages []editedPage
	if err := json.Unmarshal([]byte(trimmed), &pages); err != nil {
		return "", err
	}

	var b strings.Builder
	first := true
	for _, page := range pages {
		for _, block := range page.Blocks {
			text := strings.TrimSpace(block.Text)
			if text == "" {
				continue
			}
			if !first {
				if block.IsSectionHeader {
					b.WriteString("\n\n")
				} else {
					b.WriteString("\n")
				}
			}
			b.WriteString(text)
			first = false
		}
	}

	return b.String(), nil
}
