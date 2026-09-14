package transcriptmcp

import (
	"strings"
	"testing"
)

func TestBaseURLFromEnv(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		t.Setenv(EnvVar, "")
		_, err := BaseURLFromEnv()
		if err == nil {
			t.Fatal("want error when TRANSCRIPT_API_URL is unset")
		}
		if !strings.Contains(err.Error(), EnvVar) {
			t.Errorf("error %q does not name %s", err.Error(), EnvVar)
		}
	})

	t.Run("unparseable", func(t *testing.T) {
		t.Setenv(EnvVar, "://not-a-url")
		_, err := BaseURLFromEnv()
		if err == nil {
			t.Fatal("want error for unparseable TRANSCRIPT_API_URL")
		}
		if !strings.Contains(err.Error(), EnvVar) {
			t.Errorf("error %q does not name %s", err.Error(), EnvVar)
		}
	})

	t.Run("valid trims trailing slash", func(t *testing.T) {
		t.Setenv(EnvVar, "http://192.168.30.2:8080/")
		got, err := BaseURLFromEnv()
		if err != nil {
			t.Fatalf("BaseURLFromEnv: %v", err)
		}
		if got != "http://192.168.30.2:8080" {
			t.Errorf("BaseURLFromEnv = %q, want trimmed trailing slash", got)
		}
	})
}
