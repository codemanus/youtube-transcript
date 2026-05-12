package youtube

import (
	"strings"
	"testing"
)

func TestYouTubeTranscriptApiException_Error(t *testing.T) {
	err := &YouTubeTranscriptApiException{
		VideoID: "test123",
	}
	msg := err.Error()
	expected := "Could not retrieve a transcript for the video https://www.youtube.com/watch?v=test123!"
	if msg != expected {
		t.Errorf("Error() = %s, want %s", msg, expected)
	}
}

func TestCouldNotRetrieveTranscript_Error(t *testing.T) {
	err := &CouldNotRetrieveTranscript{
		YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: "test123"},
		Cause: "Test cause",
	}
	msg := err.Error()
	if !strings.Contains(msg, "test123") {
		t.Errorf("Error() should contain video ID")
	}
	if !strings.Contains(msg, "Test cause") {
		t.Errorf("Error() should contain cause")
	}
}

func TestNewYouTubeDataUnparsable(t *testing.T) {
	err := NewYouTubeDataUnparsable("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if err.Cause == "" {
		t.Error("Cause should not be empty")
	}
}

func TestNewYouTubeRequestFailed(t *testing.T) {
	err := NewYouTubeRequestFailed("test123", "connection timeout")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if err.Reason != "connection timeout" {
		t.Errorf("Reason = %s, want 'connection timeout'", err.Reason)
	}
	if !strings.Contains(err.Error(), "connection timeout") {
		t.Error("Error() should contain reason")
	}
}

func TestNewVideoUnavailable(t *testing.T) {
	err := NewVideoUnavailable("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "no longer available") {
		t.Error("Error() should mention video unavailable")
	}
}

func TestNewInvalidVideoID(t *testing.T) {
	err := NewInvalidVideoID("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "invalid video id") {
		t.Error("Error() should mention invalid video ID")
	}
	if !strings.Contains(err.Error(), "NOT the url") {
		t.Error("Error() should warn about using URL instead of ID")
	}
}

func TestNewRequestBlocked(t *testing.T) {
	err := NewRequestBlocked("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "blocking requests") {
		t.Error("Error() should mention blocking")
	}
}

func TestNewIpBlocked(t *testing.T) {
	err := NewIpBlocked("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	// IpBlocked embeds RequestBlocked
	if !strings.Contains(err.Error(), "blocking") {
		t.Error("Error() should mention blocking")
	}
}

func TestNewTranscriptsDisabled(t *testing.T) {
	err := NewTranscriptsDisabled("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Error("Error() should mention transcripts disabled")
	}
}

func TestNewAgeRestricted(t *testing.T) {
	err := NewAgeRestricted("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "age-restricted") {
		t.Error("Error() should mention age-restricted")
	}
}

func TestNewNotTranslatable(t *testing.T) {
	err := NewNotTranslatable("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "not translatable") {
		t.Error("Error() should mention not translatable")
	}
}

func TestNewTranslationLanguageNotAvailable(t *testing.T) {
	err := NewTranslationLanguageNotAvailable("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "not available") {
		t.Error("Error() should mention not available")
	}
}

func TestNewFailedToCreateConsentCookie(t *testing.T) {
	err := NewFailedToCreateConsentCookie("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "consent") {
		t.Error("Error() should mention consent")
	}
}

func TestNewPoTokenRequired(t *testing.T) {
	err := NewPoTokenRequired("test123")
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if !strings.Contains(err.Error(), "PO Token") {
		t.Error("Error() should mention PO Token")
	}
}

func TestNewNoTranscriptFound(t *testing.T) {
	transcriptList := newTranscriptList(
		"test123",
		map[string]*Transcript{},
		map[string]*Transcript{},
		[]TranslationLanguage{},
	)
	err := NewNoTranscriptFound("test123", []string{"en", "de"}, transcriptList)
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if len(err.RequestedLanguageCodes) != 2 {
		t.Errorf("RequestedLanguageCodes length = %d, want 2", len(err.RequestedLanguageCodes))
	}
	if err.TranscriptData == nil {
		t.Error("TranscriptData should not be nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "en") || !strings.Contains(msg, "de") {
		t.Error("Error() should contain requested language codes")
	}
}

func TestNewVideoUnplayable(t *testing.T) {
	subreasons := []string{"Subreason 1", "Subreason 2"}
	err := NewVideoUnplayable("test123", "Content not available", subreasons)
	if err.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", err.VideoID)
	}
	if err.Reason != "Content not available" {
		t.Errorf("Reason = %s, want 'Content not available'", err.Reason)
	}
	if len(err.Subreasons) != 2 {
		t.Errorf("Subreasons length = %d, want 2", len(err.Subreasons))
	}
	msg := err.Error()
	if !strings.Contains(msg, "Content not available") {
		t.Error("Error() should contain reason")
	}
}

// Test error interface implementation
func TestErrorsImplementError(t *testing.T) {
	errors := []error{
		NewYouTubeDataUnparsable("test"),
		NewYouTubeRequestFailed("test", "reason"),
		NewVideoUnavailable("test"),
		NewInvalidVideoID("test"),
		NewRequestBlocked("test"),
		NewIpBlocked("test"),
		NewTranscriptsDisabled("test"),
		NewAgeRestricted("test"),
		NewNotTranslatable("test"),
		NewTranslationLanguageNotAvailable("test"),
		NewFailedToCreateConsentCookie("test"),
		NewPoTokenRequired("test"),
		NewVideoUnplayable("test", "reason", []string{"subreason"}),
	}

	for i, err := range errors {
		if err == nil {
			t.Errorf("Error %d should not be nil", i)
		}
		if err.Error() == "" {
			t.Errorf("Error %d should have non-empty Error() message", i)
		}
	}
}
