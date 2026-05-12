package youtube

import (
	"encoding/xml"
	"html"
	"regexp"
	"strings"
)

// formattingTags are HTML tags that should be preserved when preserve_formatting is true.
var formattingTags = []string{
	"strong", // important
	"em",     // emphasized
	"b",      // bold
	"i",      // italic
	"mark",   // marked
	"small",  // smaller
	"del",    // deleted
	"ins",    // inserted
	"sub",    // subscript
	"sup",    // superscript
}

// TranscriptXML represents the XML structure of a YouTube transcript.
type TranscriptXML struct {
	XMLName xml.Name       `xml:"transcript"`
	Texts   []TextElement `xml:"text"`
}

// TextElement represents a single text element in the transcript XML.
type TextElement struct {
	Text     string  `xml:",chardata"`
	Start    float64 `xml:"start,attr"`
	Duration float64 `xml:"dur,attr"`
}

// TranscriptParser parses transcript XML data.
type TranscriptParser struct {
	preserveFormatting bool
}

// NewTranscriptParser creates a new TranscriptParser.
func NewTranscriptParser(preserveFormatting bool) *TranscriptParser {
	return &TranscriptParser{
		preserveFormatting: preserveFormatting,
	}
}

// stripHTML removes HTML tags from text based on preserveFormatting setting.
func (p *TranscriptParser) stripHTML(text string) string {
	if !p.preserveFormatting {
		// Remove all HTML tags
		re := regexp.MustCompile(`<[^>]*>`)
		return re.ReplaceAllString(text, "")
	}

	// Preserve formatting tags, remove others
	// This is a simplified approach since Go's regexp doesn't support negative lookahead
	result := text
	for {
		// Find opening tag that's not a formatting tag
		re := regexp.MustCompile(`<([/!])?([a-z]+)[^>]*>`)
		match := re.FindStringSubmatchIndex(result)
		if match == nil {
			break
		}

		tagName := result[match[4]:match[5]]
		// Check if this is a formatting tag
		isFormattingTag := false
		for _, fmtTag := range formattingTags {
			if tagName == fmtTag {
				isFormattingTag = true
				break
			}
		}

		if !isFormattingTag {
			// Remove this tag
			result = result[:match[0]] + result[match[1]:]
		} else {
			// Skip past this tag
			result = result[match[1]:]
		}
	}

	return result
}

// Parse parses raw XML data into a FetchedTranscript.
func (p *TranscriptParser) Parse(rawXML string, videoID, language, languageCode string, isGenerated bool) (*FetchedTranscript, error) {
	var transcriptXML TranscriptXML
	err := xml.Unmarshal([]byte(rawXML), &transcriptXML)
	if err != nil {
		return nil, err
	}

	snippets := make([]FetchedTranscriptSnippet, 0, len(transcriptXML.Texts))
	for _, element := range transcriptXML.Texts {
		if element.Text != "" {
			// Unescape HTML entities
			text := html.UnescapeString(element.Text)

			// Strip HTML tags based on preserve_formatting setting
			text = p.stripHTML(text)

			snippets = append(snippets, FetchedTranscriptSnippet{
				Text:     text,
				Start:    element.Start,
				Duration: element.Duration,
			})
		}
	}

	return &FetchedTranscript{
		Snippets:     snippets,
		VideoID:      videoID,
		Language:     language,
		LanguageCode: languageCode,
		IsGenerated:  isGenerated,
	}, nil
}

// Fetch retrieves and parses the transcript data.
func (t *Transcript) Fetch(preserveFormatting bool) (*FetchedTranscript, error) {
	// Check if PoToken is required
	if strings.Contains(t.BaseURL, "&exp=xpe") {
		return nil, NewPoTokenRequired(t.VideoID)
	}

	response, err := t.client.Get(t.BaseURL)
	if err != nil {
		if e, ok := err.(*YouTubeRequestFailed); ok {
			e.VideoID = t.VideoID
		}
		if e, ok := err.(*IpBlocked); ok {
			e.VideoID = t.VideoID
		}
		return nil, err
	}

	parser := NewTranscriptParser(preserveFormatting)
	return parser.Parse(response, t.VideoID, t.Language, t.LanguageCode, t.IsGenerated)
}
