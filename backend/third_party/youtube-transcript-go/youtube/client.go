package youtube

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Browser-like User-Agent; YouTube often rejects or truncates the default Go client string.
const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

func newYouTubeTransport() *http.Transport {
	t := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// YouTube frequently resets HTTP/2 connections for script-like clients; stick to HTTP/1.1.
		TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
	}
	return t
}

func setWatchPageHeaders(req *http.Request) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func setInnertubePOSTHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Referer", "https://www.youtube.com/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
}

// Caption / timedtext GET (not the watch HTML page).
func setCaptionGETHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Referer", "https://www.youtube.com/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
}

func headersForGET(urlStr string) func(*http.Request) {
	if strings.Contains(urlStr, "youtube.com/watch?") || strings.Contains(urlStr, "youtube.com/watch#") {
		return setWatchPageHeaders
	}
	return setCaptionGETHeaders
}

// Client is an HTTP client for making requests to YouTube.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new HTTP client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout:   45 * time.Second,
			Transport: newYouTubeTransport(),
		},
	}
}

// Get performs a GET request and returns the response body.
func (c *Client) Get(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	headersForGET(url)(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 429 {
		return "", NewIpBlocked("") // VideoID will be set by caller
	}

	if resp.StatusCode >= 400 {
		return "", NewYouTubeRequestFailed("", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	return string(body), nil
}

// Post performs a POST request with JSON body and returns the response body.
func (c *Client) Post(url string, jsonBody string) (string, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	setInnertubePOSTHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 429 {
		return "", NewIpBlocked("") // VideoID will be set by caller
	}

	if resp.StatusCode >= 400 {
		return "", NewYouTubeRequestFailed("", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	return string(body), nil
}

// GetWithCookies performs a GET request with cookies.
func (c *Client) GetWithCookies(url string, cookies []*http.Cookie) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	headersForGET(url)(req)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 429 {
		return "", NewIpBlocked("") // VideoID will be set by caller
	}

	if resp.StatusCode >= 400 {
		return "", NewYouTubeRequestFailed("", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	return string(body), nil
}

// SetProxy sets a proxy for the HTTP client.
func (c *Client) SetProxy(proxyURL string) error {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}

	t := newYouTubeTransport()
	t.Proxy = http.ProxyURL(parsedURL)
	c.httpClient.Transport = t
	return nil
}

// GetCookies returns cookies for a given URL.
func (c *Client) GetCookies(u *url.URL) []*http.Cookie {
	if jar, ok := c.httpClient.Jar.(interface {
		GetCookies(*url.URL) []*http.Cookie
	}); ok {
		return jar.GetCookies(u)
	}
	return nil
}

// SetCookie sets a cookie for a given domain.
func (c *Client) SetCookie(cookie *http.Cookie) {
	if jar, ok := c.httpClient.Jar.(interface {
		SetCookie(*http.Cookie)
	}); ok {
		jar.SetCookie(cookie)
	}
}
