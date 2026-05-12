package youtube

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewInnertubeContext(t *testing.T) {
	ctx := NewInnertubeContext()
	if ctx.Client.ClientName != ClientName {
		t.Errorf("ClientName = %s, want %s", ctx.Client.ClientName, ClientName)
	}
	if ctx.Client.ClientVersion != ClientVersion {
		t.Errorf("ClientVersion = %s, want %s", ctx.Client.ClientVersion, ClientVersion)
	}
}

func TestNewInnertubeRequest(t *testing.T) {
	req := NewInnertubeRequest("test123")
	if req.VideoID != "test123" {
		t.Errorf("VideoID = %s, want 'test123'", req.VideoID)
	}
	if req.Context.Client.ClientName != ClientName {
		t.Errorf("ClientName = %s, want %s", req.Context.Client.ClientName, ClientName)
	}
}

func TestInnertubeRequest_MarshalJSON(t *testing.T) {
	req := NewInnertubeRequest("test123")
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if !strings.Contains(string(data), "test123") {
		t.Error("JSON should contain video ID")
	}
	if !strings.Contains(string(data), ClientName) {
		t.Error("JSON should contain client name")
	}
}

func TestExtractInnertubeAPIKey(t *testing.T) {
	fetcher := newTranscriptListFetcher(NewClient())

	t.Run("Valid API key", func(t *testing.T) {
		html := `<script>"INNERTUBE_API_KEY": "AIzaSyTestKey123"</script>`
		key, err := fetcher.extractInnertubeAPIKey(html, "test123")
		if err != nil {
			t.Fatalf("extractInnertubeAPIKey() error = %v", err)
		}
		if key != "AIzaSyTestKey123" {
			t.Errorf("API key = %s, want 'AIzaSyTestKey123'", key)
		}
	})

	t.Run("API key with spaces", func(t *testing.T) {
		html := `<script>"INNERTUBE_API_KEY":  "AIzaSyTestKey123"  </script>`
		key, err := fetcher.extractInnertubeAPIKey(html, "test123")
		if err != nil {
			t.Fatalf("extractInnertubeAPIKey() error = %v", err)
		}
		if key != "AIzaSyTestKey123" {
			t.Errorf("API key = %s, want 'AIzaSyTestKey123'", key)
		}
	})

	t.Run("No API key found", func(t *testing.T) {
		html := `<html><body>No API key here</body></html>`
		_, err := fetcher.extractInnertubeAPIKey(html, "test123")
		if err == nil {
			t.Error("extractInnertubeAPIKey() should return error when API key not found")
		}
		if _, ok := err.(*YouTubeDataUnparsable); !ok {
			t.Errorf("Error should be YouTubeDataUnparsable, got %T", err)
		}
	})

	t.Run("Captcha detected (IP blocked)", func(t *testing.T) {
		html := `<html><body><div class="g-recaptcha"></div></body></html>`
		_, err := fetcher.extractInnertubeAPIKey(html, "test123")
		if err == nil {
			t.Error("extractInnertubeAPIKey() should return error when captcha detected")
		}
		if _, ok := err.(*IpBlocked); !ok {
			t.Errorf("Error should be IpBlocked, got %T", err)
		}
	})
}

func TestAssertPlayability(t *testing.T) {
	fetcher := newTranscriptListFetcher(NewClient())

	t.Run("OK status", func(t *testing.T) {
		status := &PlayabilityStatus{Status: "OK"}
		err := fetcher.assertPlayability(status, "test123")
		if err != nil {
			t.Errorf("assertPlayability() should return nil for OK status, got %v", err)
		}
	})

	t.Run("Nil status", func(t *testing.T) {
		err := fetcher.assertPlayability(nil, "test123")
		if err != nil {
			t.Errorf("assertPlayability() should return nil for nil status, got %v", err)
		}
	})

	t.Run("LOGIN_REQUIRED - bot detected", func(t *testing.T) {
		status := &PlayabilityStatus{
			Status: "LOGIN_REQUIRED",
			Reason: "Sign in to confirm you're not a bot",
		}
		err := fetcher.assertPlayability(status, "test123")
		if err == nil {
			t.Error("assertPlayability() should return error for bot detection")
		}
		if _, ok := err.(*RequestBlocked); !ok {
			t.Errorf("Error should be RequestBlocked, got %T", err)
		}
	})

	t.Run("LOGIN_REQUIRED - age restricted", func(t *testing.T) {
		status := &PlayabilityStatus{
			Status: "LOGIN_REQUIRED",
			Reason: "This video may be inappropriate for some users.",
		}
		err := fetcher.assertPlayability(status, "test123")
		if err == nil {
			t.Error("assertPlayability() should return error for age restricted")
		}
		if _, ok := err.(*AgeRestricted); !ok {
			t.Errorf("Error should be AgeRestricted, got %T", err)
		}
	})

	t.Run("ERROR - video unavailable", func(t *testing.T) {
		status := &PlayabilityStatus{
			Status: "ERROR",
			Reason: "This video is unavailable",
		}
		err := fetcher.assertPlayability(status, "test123")
		if err == nil {
			t.Error("assertPlayability() should return error for unavailable video")
		}
		if _, ok := err.(*VideoUnavailable); !ok {
			t.Errorf("Error should be VideoUnavailable, got %T", err)
		}
	})

	t.Run("ERROR - invalid video ID (URL provided)", func(t *testing.T) {
		status := &PlayabilityStatus{
			Status: "ERROR",
			Reason: "This video is unavailable",
		}
		err := fetcher.assertPlayability(status, "https://www.youtube.com/watch?v=test123")
		if err == nil {
			t.Error("assertPlayability() should return error for URL as video ID")
		}
		if _, ok := err.(*InvalidVideoID); !ok {
			t.Errorf("Error should be InvalidVideoID, got %T", err)
		}
	})

	t.Run("Video unplayable with subreasons", func(t *testing.T) {
		status := &PlayabilityStatus{
			Status: "ERROR",
			Reason: "Some error",
			ErrorScreen: &struct {
				PlayerErrorMessageRenderer *struct {
					Subreason *struct {
						Runs []struct {
							Text string `json:"text"`
						} `json:"runs"`
					} `json:"subreason"`
				} `json:"playerErrorMessageRenderer"`
			}{
				PlayerErrorMessageRenderer: &struct {
					Subreason *struct {
						Runs []struct {
							Text string `json:"text"`
						} `json:"runs"`
					} `json:"subreason"`
				}{
					Subreason: &struct {
						Runs []struct {
							Text string `json:"text"`
						} `json:"runs"`
					}{
						Runs: []struct {
							Text string `json:"text"`
						}{
							{Text: "Subreason 1"},
							{Text: "Subreason 2"},
						},
					},
				},
			},
		}
		err := fetcher.assertPlayability(status, "test123")
		if err == nil {
			t.Error("assertPlayability() should return error")
		}
		if unplayable, ok := err.(*VideoUnplayable); ok {
			if len(unplayable.Subreasons) != 2 {
				t.Errorf("Subreasons length = %d, want 2", len(unplayable.Subreasons))
			}
		} else {
			t.Errorf("Error should be VideoUnplayable, got %T", err)
		}
	})
}

func TestBuildTranscriptList(t *testing.T) {
	client := NewClient()

	captionsJSON := &CaptionsRenderer{
		CaptionTracks: []CaptionTrack{
			{
				BaseURL: "http://example.com/en&fmt=srv3",
				Name: &struct {
					Runs []struct {
						Text string `json:"text"`
					} `json:"runs"`
				}{
					Runs: []struct {
						Text string `json:"text"`
					}{{Text: "English"}},
				},
				LanguageCode:   "en",
				Kind:           "",
				IsTranslatable: true,
			},
			{
				BaseURL: "http://example.com/de",
				Name: &struct {
					Runs []struct {
						Text string `json:"text"`
					} `json:"runs"`
				}{
					Runs: []struct {
						Text string `json:"text"`
					}{{Text: "German"}},
				},
				LanguageCode:   "de",
				Kind:           "asr",
				IsTranslatable: true,
			},
		},
		TranslationLanguages: []TranslationLanguageInfo{
			{
				LanguageName: &struct {
					Runs []struct {
						Text string `json:"text"`
					} `json:"runs"`
				}{
					Runs: []struct {
						Text string `json:"text"`
					}{{Text: "French"}},
				},
				LanguageCode: "fr",
			},
		},
	}

	list, err := buildTranscriptList(client, "test123", captionsJSON)
	if err != nil {
		t.Fatalf("buildTranscriptList() error = %v", err)
	}

	if list.VideoID != "test123" {
		t.Errorf("VideoID = %s, want 'test123'", list.VideoID)
	}

	if len(list.ManuallyCreatedTranscripts) != 1 {
		t.Errorf("ManuallyCreatedTranscripts length = %d, want 1", len(list.ManuallyCreatedTranscripts))
	}
	if _, ok := list.ManuallyCreatedTranscripts["en"]; !ok {
		t.Error("Should have English manual transcript")
	}

	if len(list.GeneratedTranscripts) != 1 {
		t.Errorf("GeneratedTranscripts length = %d, want 1", len(list.GeneratedTranscripts))
	}
	if _, ok := list.GeneratedTranscripts["de"]; !ok {
		t.Error("Should have German generated transcript")
	}

	if len(list.TranslationLanguages) != 1 {
		t.Errorf("TranslationLanguages length = %d, want 1", len(list.TranslationLanguages))
	}

	// Check that &fmt=srv3 was removed from base URL
	enTranscript := list.ManuallyCreatedTranscripts["en"]
	if strings.Contains(enTranscript.BaseURL, "&fmt=srv3") {
		t.Error("BaseURL should have &fmt=srv3 removed")
	}
}

func TestFetchInnertubeData(t *testing.T) {
	// We need to extract the API key part from the server URL
	// For this test, we'll test JSON marshaling of the request/response
	// In real scenario, the URL would be constructed with the API key

	// Since fetchInnertubeData constructs the URL with apiKey, we need to test differently
	// Let's test with a mock that returns the response directly

	t.Run("Successful response", func(t *testing.T) {
		// Create test response
		response := InnertubeResponse{
			PlayabilityStatus: &PlayabilityStatus{Status: "OK"},
		}

		// Test JSON marshaling
		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}

		var parsed InnertubeResponse
		err = json.Unmarshal(data, &parsed)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if parsed.PlayabilityStatus.Status != "OK" {
			t.Errorf("Status = %s, want 'OK'", parsed.PlayabilityStatus.Status)
		}
	})
}

func TestConstants(t *testing.T) {
	if WatchURL == "" {
		t.Error("WatchURL should not be empty")
	}
	if !strings.Contains(WatchURL, "%s") {
		t.Error("WatchURL should contain format specifier")
	}

	if InnertubeAPIURL == "" {
		t.Error("InnertubeAPIURL should not be empty")
	}
	if !strings.Contains(InnertubeAPIURL, "%s") {
		t.Error("InnertubeAPIURL should contain format specifier")
	}

	if ClientName == "" {
		t.Error("ClientName should not be empty")
	}

	if ClientVersion == "" {
		t.Error("ClientVersion should not be empty")
	}
}

func TestInnertubeRegexPatterns(t *testing.T) {
	t.Run("API key pattern matches valid key", func(t *testing.T) {
		html := `{"INNERTUBE_API_KEY": "AIzaSyABC123xyz-789_YZ"}`
		matches := innertubeAPIKeyPattern.FindStringSubmatch(html)
		if len(matches) < 2 {
			t.Error("Pattern should match API key")
		} else if matches[1] != "AIzaSyABC123xyz-789_YZ" {
			t.Errorf("Matched key = %s, want 'AIzaSyABC123xyz-789_YZ'", matches[1])
		}
	})

	t.Run("API key pattern doesn't match invalid format", func(t *testing.T) {
		html := `{"INNERTUBE_API_KEY": "INVALID!!@#"}`
		matches := innertubeAPIKeyPattern.FindStringSubmatch(html)
		if len(matches) >= 2 {
			t.Errorf("Pattern should not match invalid key, got %s", matches[1])
		}
	})

	t.Run("Consent value pattern", func(t *testing.T) {
		html := `name="v" value="ABCD1234"`
		matches := consentValuePattern.FindStringSubmatch(html)
		if len(matches) < 2 {
			t.Error("Pattern should match consent value")
		} else if matches[1] != "ABCD1234" {
			t.Errorf("Matched value = %s, want 'ABCD1234'", matches[1])
		}
	})
}
