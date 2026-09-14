// Package portalcookie stores the Groups Portal session cookie as a file on
// disk, read fresh on every Church Guide request rather than once at process
// startup. See docs/adr/0002 for why this breaks from the app's usual
// env-var configuration convention.
package portalcookie

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultPath is where the cookie lives when PathEnvVar is unset.
const DefaultPath = "/var/lib/youtube-transcript/groups-portal-cookie"

// PathEnvVar overrides DefaultPath, for local development.
const PathEnvVar = "GROUPS_PORTAL_COOKIE_PATH"

// writeMu serializes Write calls, so two concurrent admin saves can't
// interleave their directory-creation and temp-file steps.
var writeMu sync.Mutex

func path() string {
	if p := strings.TrimSpace(os.Getenv(PathEnvVar)); p != "" {
		return p
	}
	return DefaultPath
}

// Read returns the current cookie value, the time it was last written
// (the file's mtime), and whether one is configured at all. A missing or
// empty file is reported as not configured, not an error.
//
// It stats and reads through a single open file descriptor rather than
// os.Stat followed by os.ReadFile, so a concurrent Write's rename can't pair
// this call's reported mtime with a different Write's content.
func Read() (value string, updatedAt time.Time, configured bool, err error) {
	p := path()

	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("open %s: %w", p, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("stat %s: %w", p, err)
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("read %s: %w", p, err)
	}

	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", info.ModTime(), false, nil
	}
	return v, info.ModTime(), true, nil
}

// Write stores value as the cookie, creating its containing directory if
// needed, with owner-only permissions. It writes to a temporary file in the
// same directory and renames it into place, so a concurrent Read (from a
// live request) never observes a partially-written value, and the
// owner-only permissions apply even if a file or directory already existed.
//
// It returns the mtime the write ends up with, observed under the same lock
// that serializes Write calls — so a caller that needs to report "what did I
// just save" doesn't have to make a separate, unsynchronized Read call that
// a second, concurrent Write could race.
func Write(value string) (updatedAt time.Time, err error) {
	writeMu.Lock()
	defer writeMu.Unlock()

	p := path()
	dir := filepath.Dir(p)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return time.Time{}, fmt.Errorf("create %s: %w", dir, err)
	}
	// MkdirAll only sets the mode on directories it creates; re-assert it in
	// case dir already existed with broader permissions.
	if err := os.Chmod(dir, 0o700); err != nil {
		return time.Time{}, fmt.Errorf("chmod %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".groups-portal-cookie-*")
	if err != nil {
		return time.Time{}, fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	if _, err := tmp.WriteString(value); err != nil {
		tmp.Close()
		return time.Time{}, fmt.Errorf("write %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return time.Time{}, fmt.Errorf("write %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, p); err != nil {
		return time.Time{}, fmt.Errorf("rename %s to %s: %w", tmpPath, p, err)
	}

	info, err := os.Stat(p)
	if err != nil {
		return time.Time{}, fmt.Errorf("stat %s: %w", p, err)
	}
	return info.ModTime(), nil
}
