package transcriptmcp

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// EnvVar is the environment variable naming the transcript service address.
const EnvVar = "TRANSCRIPT_API_URL"

// BaseURLFromEnv reads and validates TRANSCRIPT_API_URL. It returns an error
// naming the variable when it is unset or not a usable absolute URL.
func BaseURLFromEnv() (string, error) {
	raw := strings.TrimSpace(os.Getenv(EnvVar))
	if raw == "" {
		return "", fmt.Errorf("%s is not set; point it at the transcript service, e.g. http://192.168.1.10:8080", EnvVar)
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%s is not a valid URL: %q", EnvVar, raw)
	}

	return strings.TrimRight(raw, "/"), nil
}
