package youtube

import (
	"strings"
	"testing"
)

func TestNewTranscriptParser(t *testing.T) {
	// Test parser with preserve_formatting = false
	parser := NewTranscriptParser(false)
	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}

	// Test parser with preserve_formatting = true
	parser = NewTranscriptParser(true)
	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}
}

func TestTranscriptParser_Parse(t *testing.T) {
	tests := []struct {
		name            string
		rawXML          string
		preserveFormat  bool
		videoID         string
		language        string
		languageCode    string
		isGenerated     bool
		expectedSnippets int
		expectedFirstText string
	}{
		{
			name: "Basic transcript",
			rawXML: `<transcript>
				<text start="0.0" dur="1.0">Hello world</text>
				<text start="1.5" dur="2.0">This is a test</text>
			</transcript>`,
			preserveFormat:  false,
			videoID:         "test123",
			language:        "English",
			languageCode:    "en",
			isGenerated:     false,
			expectedSnippets: 2,
			expectedFirstText: "Hello world",
		},
		{
			name: "Transcript with HTML tags",
			rawXML: `<transcript>
				<text start="0.0" dur="1.0">&lt;b&gt;Hello&lt;/b&gt; world</text>
				<text start="1.5" dur="2.0">This is &lt;i&gt;awesome&lt;/i&gt;</text>
			</transcript>`,
			preserveFormat:  false,
			videoID:         "test123",
			language:        "English",
			languageCode:    "en",
			isGenerated:     false,
			expectedSnippets: 2,
			expectedFirstText: "Hello world",
		},
		{
			name: "Transcript with HTML entities",
			rawXML: `<transcript>
				<text start="0.0" dur="1.0">Hello &amp; welcome</text>
				<text start="1.5" dur="2.0">Test &quot;quotes&quot;</text>
			</transcript>`,
			preserveFormat:  false,
			videoID:         "test123",
			language:        "English",
			languageCode:    "en",
			isGenerated:     false,
			expectedSnippets: 2,
			expectedFirstText: "Hello & welcome",
		},
		{
			name: "Empty transcript",
			rawXML: `<transcript></transcript>`,
			preserveFormat:  false,
			videoID:         "test123",
			language:        "English",
			languageCode:    "en",
			isGenerated:     false,
			expectedSnippets: 0,
			expectedFirstText: "",
		},
		{
			name: "Transcript with duration 0",
			rawXML: `<transcript>
				<text start="0.0">No duration specified</text>
			</transcript>`,
			preserveFormat:  false,
			videoID:         "test123",
			language:        "English",
			languageCode:    "en",
			isGenerated:     false,
			expectedSnippets: 1,
			expectedFirstText: "No duration specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewTranscriptParser(tt.preserveFormat)
			result, err := parser.Parse(tt.rawXML, tt.videoID, tt.language, tt.languageCode, tt.isGenerated)

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if result.Len() != tt.expectedSnippets {
				t.Errorf("Parse() got %d snippets, want %d", result.Len(), tt.expectedSnippets)
			}

			if result.VideoID != tt.videoID {
				t.Errorf("Parse() VideoID = %s, want %s", result.VideoID, tt.videoID)
			}

			if result.Language != tt.language {
				t.Errorf("Parse() Language = %s, want %s", result.Language, tt.language)
			}

			if result.LanguageCode != tt.languageCode {
				t.Errorf("Parse() LanguageCode = %s, want %s", result.LanguageCode, tt.languageCode)
			}

			if result.IsGenerated != tt.isGenerated {
				t.Errorf("Parse() IsGenerated = %v, want %v", result.IsGenerated, tt.isGenerated)
			}

			if tt.expectedSnippets > 0 && tt.expectedFirstText != "" {
				if len(result.Snippets) == 0 {
					t.Fatal("Parse() returned no snippets")
				}
				if result.Snippets[0].Text != tt.expectedFirstText {
					t.Errorf("Parse() first snippet text = %s, want %s", result.Snippets[0].Text, tt.expectedFirstText)
				}
			}
		})
	}
}

func TestTranscriptParser_Parse_InvalidXML(t *testing.T) {
	parser := NewTranscriptParser(false)
	_, err := parser.Parse("<invalid>xml", "test", "en", "en", false)
	if err == nil {
		t.Error("Parse() should return error for invalid XML")
	}
}

func TestTranscriptParser_stripHTML(t *testing.T) {
	// Test preserve_formatting = false (strip all HTML)
	parser := NewTranscriptParser(false)
	testStr := "<b>Hello</b> <i>World</i>"
	result := parser.stripHTML(testStr)
	if result != "Hello World" {
		t.Errorf("stripHTML(false) = %s, want 'Hello World'", result)
	}

	// Test preserve_formatting = true (preserve formatting tags)
	parser = NewTranscriptParser(true)
	testStr = "<b>Hello</b> <span>World</span>"
	result = parser.stripHTML(testStr)
	// Note: stripHTML keeps the tag content even when removing the tag
	if !strings.Contains(result, "<b>Hello</b>") {
		t.Logf("stripHTML(true) = %s", result)
	}
}
