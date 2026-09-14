package portalcookie

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadNotConfiguredWhenFileAbsent(t *testing.T) {
	t.Setenv(PathEnvVar, filepath.Join(t.TempDir(), "groups-portal-cookie"))

	value, updatedAt, configured, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if configured {
		t.Error("configured = true, want false when the file does not exist")
	}
	if value != "" {
		t.Errorf("value = %q, want empty", value)
	}
	if !updatedAt.IsZero() {
		t.Errorf("updatedAt = %v, want zero time", updatedAt)
	}
}

func TestWriteThenReadRoundTrip(t *testing.T) {
	t.Setenv(PathEnvVar, filepath.Join(t.TempDir(), "groups-portal-cookie"))

	if err := Write("connect.sid=abc123"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	value, _, configured, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !configured {
		t.Error("configured = false, want true after Write")
	}
	if value != "connect.sid=abc123" {
		t.Errorf("value = %q, want %q", value, "connect.sid=abc123")
	}
}

func TestReadReportsFileModTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "groups-portal-cookie")
	t.Setenv(PathEnvVar, path)

	if err := Write("connect.sid=abc123"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	_, updatedAt, _, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !updatedAt.Equal(info.ModTime()) {
		t.Errorf("updatedAt = %v, want %v (file mtime)", updatedAt, info.ModTime())
	}
}

func TestWriteOverwritesPreviousValue(t *testing.T) {
	t.Setenv(PathEnvVar, filepath.Join(t.TempDir(), "groups-portal-cookie"))

	if err := Write("connect.sid=old"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	time.Sleep(2 * time.Millisecond) // ensure a distinguishable mtime, in case the filesystem's clock is coarse
	if err := Write("connect.sid=new"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	value, _, _, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if value != "connect.sid=new" {
		t.Errorf("value = %q, want %q", value, "connect.sid=new")
	}
}
