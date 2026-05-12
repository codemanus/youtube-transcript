package youtube

import (
	"testing"
)

func TestNewTranscript(t *testing.T) {
	client := NewClient()
	tl := []TranslationLanguage{
		{Language: "German", LanguageCode: "de"},
		{Language: "French", LanguageCode: "fr"},
	}

	transcript := newTranscript(
		client,
		"test123",
		"http://example.com/transcript",
		"English",
		"en",
		false,
		tl,
	)

	if transcript.VideoID != "test123" {
		t.Errorf("VideoID = %s, want test123", transcript.VideoID)
	}
	if transcript.BaseURL != "http://example.com/transcript" {
		t.Errorf("BaseURL = %s, want 'http://example.com/transcript'", transcript.BaseURL)
	}
	if transcript.Language != "English" {
		t.Errorf("Language = %s, want 'English'", transcript.Language)
	}
	if transcript.LanguageCode != "en" {
		t.Errorf("LanguageCode = %s, want 'en'", transcript.LanguageCode)
	}
	if transcript.IsGenerated {
		t.Error("IsGenerated = true, want false")
	}
	if len(transcript.translationLanguages) != 2 {
		t.Errorf("translationLanguages length = %d, want 2", len(transcript.translationLanguages))
	}
}

func TestTranscript_IsTranslatable(t *testing.T) {
	tests := []struct {
		name           string
		translationLanguages []TranslationLanguage
		expected       bool
	}{
		{
			name: "With translation languages",
			translationLanguages: []TranslationLanguage{
				{Language: "German", LanguageCode: "de"},
			},
			expected: true,
		},
		{
			name: "Without translation languages",
			translationLanguages: []TranslationLanguage{},
			expected: false,
		},
		{
			name: "Nil translation languages",
			translationLanguages: nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transcript := newTranscript(
				NewClient(),
				"test123",
				"http://example.com",
				"English",
				"en",
				false,
				tt.translationLanguages,
			)
			if got := transcript.IsTranslatable(); got != tt.expected {
				t.Errorf("IsTranslatable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTranscript_Translate(t *testing.T) {
	tl := []TranslationLanguage{
		{Language: "German", LanguageCode: "de"},
		{Language: "French", LanguageCode: "fr"},
	}

	transcript := newTranscript(
		NewClient(),
		"test123",
		"http://example.com/transcript",
		"English",
		"en",
		false,
		tl,
	)

	t.Run("Translate to available language", func(t *testing.T) {
		translated, err := transcript.Translate("de")
		if err != nil {
			t.Fatalf("Translate() error = %v", err)
		}
		if translated.LanguageCode != "de" {
			t.Errorf("LanguageCode = %s, want 'de'", translated.LanguageCode)
		}
		if translated.Language != "German" {
			t.Errorf("Language = %s, want 'German'", translated.Language)
		}
		if !translated.IsGenerated {
			t.Error("Translated transcript should be marked as generated")
		}
	})

	t.Run("Translate with non-translatable transcript", func(t *testing.T) {
		nonTranslatable := newTranscript(
			NewClient(),
			"test123",
			"http://example.com",
			"English",
			"en",
			false,
			[]TranslationLanguage{},
		)
		_, err := nonTranslatable.Translate("de")
		if err == nil {
			t.Error("Translate() should return error for non-translatable transcript")
		}
		if _, ok := err.(*NotTranslatable); !ok {
			t.Errorf("Error should be NotTranslatable, got %T", err)
		}
	})

	t.Run("Translate to unavailable language", func(t *testing.T) {
		_, err := transcript.Translate("es")
		if err == nil {
			t.Error("Translate() should return error for unavailable language")
		}
		if _, ok := err.(*TranslationLanguageNotAvailable); !ok {
			t.Errorf("Error should be TranslationLanguageNotAvailable, got %T", err)
		}
	})
}

func TestTranscript_String(t *testing.T) {
	t.Run("Translatable transcript", func(t *testing.T) {
		tl := []TranslationLanguage{
			{Language: "German", LanguageCode: "de"},
		}
		transcript := newTranscript(
			NewClient(),
			"test123",
			"http://example.com",
			"English",
			"en",
			false,
			tl,
		)
		str := transcript.String()
		expected := "en (\"English\")[TRANSLATABLE]"
		if str != expected {
			t.Errorf("String() = %s, want %s", str, expected)
		}
	})

	t.Run("Non-translatable transcript", func(t *testing.T) {
		transcript := newTranscript(
			NewClient(),
			"test123",
			"http://example.com",
			"English",
			"en",
			false,
			[]TranslationLanguage{},
		)
		str := transcript.String()
		expected := "en (\"English\")"
		if str != expected {
			t.Errorf("String() = %s, want %s", str, expected)
		}
	})
}

func TestNewTranscriptList(t *testing.T) {
	manual := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com", "English", "en", false, nil),
	}
	generated := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com", "English", "en", true, nil),
	}
	tl := []TranslationLanguage{{Language: "German", LanguageCode: "de"}}

	list := newTranscriptList("test123", manual, generated, tl)

	if list.VideoID != "test123" {
		t.Errorf("VideoID = %s, want 'test123'", list.VideoID)
	}
	if len(list.ManuallyCreatedTranscripts) != 1 {
		t.Errorf("ManuallyCreatedTranscripts length = %d, want 1", len(list.ManuallyCreatedTranscripts))
	}
	if len(list.GeneratedTranscripts) != 1 {
		t.Errorf("GeneratedTranscripts length = %d, want 1", len(list.GeneratedTranscripts))
	}
	if len(list.TranslationLanguages) != 1 {
		t.Errorf("TranslationLanguages length = %d, want 1", len(list.TranslationLanguages))
	}
}

func TestTranscriptList_FindTranscript(t *testing.T) {
	manual := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com/en", "English", "en", false, nil),
		"de": newTranscript(NewClient(), "test", "http://example.com/de", "German", "de", false, nil),
	}
	generated := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com/gen-en", "English (auto)", "en", true, nil),
	}

	list := newTranscriptList("test123", manual, generated, nil)

	t.Run("Find manual transcript", func(t *testing.T) {
		transcript, err := list.FindTranscript([]string{"en"})
		if err != nil {
			t.Fatalf("FindTranscript() error = %v", err)
		}
		if transcript.IsGenerated {
			t.Error("Should find manual transcript, not generated")
		}
	})

	t.Run("Find with fallback", func(t *testing.T) {
		transcript, err := list.FindTranscript([]string{"fr", "de"})
		if err != nil {
			t.Fatalf("FindTranscript() error = %v", err)
		}
		if transcript.LanguageCode != "de" {
			t.Errorf("LanguageCode = %s, want 'de'", transcript.LanguageCode)
		}
	})

	t.Run("Not found", func(t *testing.T) {
		_, err := list.FindTranscript([]string{"fr", "es"})
		if err == nil {
			t.Error("FindTranscript() should return error when not found")
		}
		if _, ok := err.(*NoTranscriptFound); !ok {
			t.Errorf("Error should be NoTranscriptFound, got %T", err)
		}
	})
}

func TestTranscriptList_FindGeneratedTranscript(t *testing.T) {
	manual := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com/en", "English", "en", false, nil),
	}
	generated := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com/gen-en", "English (auto)", "en", true, nil),
	}

	list := newTranscriptList("test123", manual, generated, nil)

	transcript, err := list.FindGeneratedTranscript([]string{"en"})
	if err != nil {
		t.Fatalf("FindGeneratedTranscript() error = %v", err)
	}
	if !transcript.IsGenerated {
		t.Error("Should find generated transcript")
	}
}

func TestTranscriptList_FindManuallyCreatedTranscript(t *testing.T) {
	manual := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com/en", "English", "en", false, nil),
	}
	generated := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com/gen-en", "English (auto)", "en", true, nil),
	}

	list := newTranscriptList("test123", manual, generated, nil)

	transcript, err := list.FindManuallyCreatedTranscript([]string{"en"})
	if err != nil {
		t.Fatalf("FindManuallyCreatedTranscript() error = %v", err)
	}
	if transcript.IsGenerated {
		t.Error("Should find manually created transcript")
	}
}

func TestTranscriptList_String(t *testing.T) {
	manual := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com", "English", "en", false, nil),
	}
	generated := map[string]*Transcript{
		"en": newTranscript(NewClient(), "test", "http://example.com", "English", "en", true, nil),
	}
	tl := []TranslationLanguage{{Language: "German", LanguageCode: "de"}}

	list := newTranscriptList("test123", manual, generated, tl)

	str := list.String()
	if !contains(str, "test123") {
		t.Error("String() should contain video ID")
	}
	if !contains(str, "MANUALLY CREATED") {
		t.Error("String() should mention manually created")
	}
	if !contains(str, "GENERATED") {
		t.Error("String() should mention generated")
	}
	if !contains(str, "TRANSLATION LANGUAGES") {
		t.Error("String() should mention translation languages")
	}
}

func TestFetchedTranscript_ToRawData(t *testing.T) {
	snippets := []FetchedTranscriptSnippet{
		{Text: "Hello", Start: 0.0, Duration: 1.0},
		{Text: "World", Start: 1.5, Duration: 2.0},
	}
	transcript := &FetchedTranscript{
		Snippets:     snippets,
		VideoID:      "test123",
		Language:     "English",
		LanguageCode: "en",
		IsGenerated:  false,
	}

	data, err := transcript.ToRawData()
	if err != nil {
		t.Fatalf("ToRawData() error = %v", err)
	}

	if len(data) != 2 {
		t.Errorf("ToRawData() returned %d items, want 2", len(data))
	}

	if data[0]["text"] != "Hello" {
		t.Errorf("First snippet text = %s, want 'Hello'", data[0]["text"])
	}
	if data[1]["text"] != "World" {
		t.Errorf("Second snippet text = %s, want 'World'", data[1]["text"])
	}
}

func TestFetchedTranscript_ToJSON(t *testing.T) {
	snippets := []FetchedTranscriptSnippet{
		{Text: "Hello", Start: 0.0, Duration: 1.0},
	}
	transcript := &FetchedTranscript{
		Snippets:     snippets,
		VideoID:      "test123",
		Language:     "English",
		LanguageCode: "en",
		IsGenerated:  false,
	}

	jsonStr, err := transcript.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if !contains(jsonStr, "Hello") {
		t.Error("ToJSON() should contain snippet text")
	}
	if !contains(jsonStr, "0") {
		t.Error("ToJSON() should contain start time")
	}
}

func TestFetchedTranscript_Len(t *testing.T) {
	snippets := []FetchedTranscriptSnippet{
		{Text: "Hello", Start: 0.0, Duration: 1.0},
		{Text: "World", Start: 1.5, Duration: 2.0},
	}
	transcript := &FetchedTranscript{
		Snippets: snippets,
	}

	if transcript.Len() != 2 {
		t.Errorf("Len() = %d, want 2", transcript.Len())
	}
}

func TestFetchedTranscript_GetSnippet(t *testing.T) {
	snippets := []FetchedTranscriptSnippet{
		{Text: "Hello", Start: 0.0, Duration: 1.0},
		{Text: "World", Start: 1.5, Duration: 2.0},
	}
	transcript := &FetchedTranscript{
		Snippets: snippets,
	}

	t.Run("Valid index", func(t *testing.T) {
		snippet, ok := transcript.GetSnippet(0)
		if !ok {
			t.Error("GetSnippet(0) should return true")
		}
		if snippet.Text != "Hello" {
			t.Errorf("Text = %s, want 'Hello'", snippet.Text)
		}
	})

	t.Run("Invalid index - negative", func(t *testing.T) {
		_, ok := transcript.GetSnippet(-1)
		if ok {
			t.Error("GetSnippet(-1) should return false")
		}
	})

	t.Run("Invalid index - out of bounds", func(t *testing.T) {
		_, ok := transcript.GetSnippet(10)
		if ok {
			t.Error("GetSnippet(10) should return false")
		}
	})
}

func TestNewYouTubeTranscriptApi(t *testing.T) {
	api := NewYouTubeTranscriptApi()
	if api == nil {
		t.Fatal("NewYouTubeTranscriptApi() should not return nil")
	}
	if api.client == nil {
		t.Error("API should have a client")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
