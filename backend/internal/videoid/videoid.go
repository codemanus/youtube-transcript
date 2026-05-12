package videoid

import (
	"net/url"
	"regexp"
	"strings"
)

var bareID = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

// FromInput extracts a YouTube video ID from a URL or returns the input if it is already a bare ID.
func FromInput(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ErrEmpty
	}
	if bareID.MatchString(s) {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", ErrInvalidURL
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", ErrInvalidURL
	}

	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))

	if host == "youtu.be" {
		seg := strings.Trim(u.Path, "/")
		seg = strings.SplitN(seg, "/", 2)[0]
		if bareID.MatchString(seg) {
			return seg, nil
		}
		return "", ErrInvalidURL
	}

	if !strings.HasSuffix(host, "youtube.com") && !strings.HasSuffix(host, "youtube-nocookie.com") {
		return "", ErrInvalidURL
	}

	path := strings.Trim(u.Path, "/")

	if strings.HasPrefix(path, "watch") {
		v := u.Query().Get("v")
		if bareID.MatchString(v) {
			return v, nil
		}
	}

	for _, prefix := range []string{"embed/", "shorts/", "live/", "v/"} {
		if strings.HasPrefix(path, prefix) {
			rest := strings.TrimPrefix(path, prefix)
			id := strings.SplitN(rest, "/", 2)[0]
			if bareID.MatchString(id) {
				return id, nil
			}
		}
	}

	return "", ErrInvalidURL
}
