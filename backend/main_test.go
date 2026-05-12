package main

import "testing"

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
