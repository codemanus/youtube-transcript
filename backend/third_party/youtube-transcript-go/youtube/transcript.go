package youtube

import (
	"encoding/json"
)

// FetchedTranscriptSnippet represents a single transcript segment.
type FetchedTranscriptSnippet struct {
	Text     string  `json:"text"`
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
}

// FetchedTranscript represents a fully fetched transcript.
type FetchedTranscript struct {
	Snippets     []FetchedTranscriptSnippet `json:"snippets"`
	VideoID      string                     `json:"video_id"`
	Language     string                     `json:"language"`
	LanguageCode string                     `json:"language_code"`
	IsGenerated  bool                       `json:"is_generated"`
}

// ToRawData converts the transcript to a JSON-serializable format.
func (ft *FetchedTranscript) ToRawData() ([]map[string]interface{}, error) {
	data := make([]map[string]interface{}, len(ft.Snippets))
	for i, snippet := range ft.Snippets {
		data[i] = map[string]interface{}{
			"text":     snippet.Text,
			"start":    snippet.Start,
			"duration": snippet.Duration,
		}
	}
	return data, nil
}

// ToJSON converts the transcript to a JSON string.
func (ft *FetchedTranscript) ToJSON() (string, error) {
	data, err := ft.ToRawData()
	if err != nil {
		return "", err
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// Len returns the number of snippets in the transcript.
func (ft *FetchedTranscript) Len() int {
	return len(ft.Snippets)
}

// GetSnippet returns the snippet at the given index.
func (ft *FetchedTranscript) GetSnippet(index int) (FetchedTranscriptSnippet, bool) {
	if index < 0 || index >= len(ft.Snippets) {
		return FetchedTranscriptSnippet{}, false
	}
	return ft.Snippets[index], true
}

// TranslationLanguage represents a translatable language.
type TranslationLanguage struct {
	Language     string `json:"language"`
	LanguageCode string `json:"language_code"`
}

// Transcript represents a fetchable transcript.
type Transcript struct {
	client                *Client
	VideoID               string
	BaseURL               string
	Language              string
	LanguageCode          string
	IsGenerated           bool
	translationLanguages  []TranslationLanguage
	translationLanguagesMap map[string]string
}

// newTranscript creates a new Transcript instance.
func newTranscript(
	client *Client,
	videoID string,
	baseURL string,
	language string,
	languageCode string,
	isGenerated bool,
	translationLanguages []TranslationLanguage,
) *Transcript {
	t := &Transcript{
		client:              client,
		VideoID:             videoID,
		BaseURL:             baseURL,
		Language:            language,
		LanguageCode:        languageCode,
		IsGenerated:         isGenerated,
		translationLanguages: translationLanguages,
		translationLanguagesMap: make(map[string]string),
	}
	for _, tl := range translationLanguages {
		t.translationLanguagesMap[tl.LanguageCode] = tl.Language
	}
	return t
}

// IsTranslatable returns true if the transcript can be translated.
func (t *Transcript) IsTranslatable() bool {
	return len(t.translationLanguages) > 0
}

// Translate returns a translated version of this transcript.
func (t *Transcript) Translate(languageCode string) (*Transcript, error) {
	if !t.IsTranslatable() {
		return nil, NewNotTranslatable(t.VideoID)
	}

	_, ok := t.translationLanguagesMap[languageCode]
	if !ok {
		return nil, NewTranslationLanguageNotAvailable(t.VideoID)
	}

	return newTranscript(
		t.client,
		t.VideoID,
		t.BaseURL+"&tlang="+languageCode,
		t.translationLanguagesMap[languageCode],
		languageCode,
		true,
		nil,
	), nil
}

// String returns a string representation of the transcript.
func (t *Transcript) String() string {
	translatableDesc := ""
	if t.IsTranslatable() {
		translatableDesc = "[TRANSLATABLE]"
	}
	return t.LanguageCode + " (\"" + t.Language + "\")" + translatableDesc
}

// TranscriptList represents available transcripts for a video.
type TranscriptList struct {
	VideoID                   string
	ManuallyCreatedTranscripts map[string]*Transcript
	GeneratedTranscripts       map[string]*Transcript
	TranslationLanguages       []TranslationLanguage
}

// newTranscriptList creates a new TranscriptList.
func newTranscriptList(
	videoID string,
	manuallyCreatedTranscripts map[string]*Transcript,
	generatedTranscripts map[string]*Transcript,
	translationLanguages []TranslationLanguage,
) *TranscriptList {
	return &TranscriptList{
		VideoID:                   videoID,
		ManuallyCreatedTranscripts: manuallyCreatedTranscripts,
		GeneratedTranscripts:       generatedTranscripts,
		TranslationLanguages:       translationLanguages,
	}
}

// FindTranscript finds a transcript by language codes, preferring manually created transcripts.
func (tl *TranscriptList) FindTranscript(languageCodes []string) (*Transcript, error) {
	return tl.findTranscript(languageCodes, []map[string]*Transcript{
		tl.ManuallyCreatedTranscripts,
		tl.GeneratedTranscripts,
	})
}

// FindGeneratedTranscript finds an automatically generated transcript by language codes.
func (tl *TranscriptList) FindGeneratedTranscript(languageCodes []string) (*Transcript, error) {
	return tl.findTranscript(languageCodes, []map[string]*Transcript{
		tl.GeneratedTranscripts,
	})
}

// FindManuallyCreatedTranscript finds a manually created transcript by language codes.
func (tl *TranscriptList) FindManuallyCreatedTranscript(languageCodes []string) (*Transcript, error) {
	return tl.findTranscript(languageCodes, []map[string]*Transcript{
		tl.ManuallyCreatedTranscripts,
	})
}

// findTranscript is the internal method to find a transcript.
func (tl *TranscriptList) findTranscript(languageCodes []string, transcriptDicts []map[string]*Transcript) (*Transcript, error) {
	for _, languageCode := range languageCodes {
		for _, transcriptDict := range transcriptDicts {
			if transcript, ok := transcriptDict[languageCode]; ok {
				return transcript, nil
			}
		}
	}
	return nil, NewNoTranscriptFound(tl.VideoID, languageCodes, tl)
}

// String returns a formatted string describing available transcripts.
func (tl *TranscriptList) String() string {
	result := "For this video (" + tl.VideoID + ") transcripts are available in the following languages:\n\n"
	result += "(MANUALLY CREATED)\n"
	result += tl.getLanguageDescription(tl.manuallyCreatedTranscriptStrings()) + "\n\n"
	result += "(GENERATED)\n"
	result += tl.getLanguageDescription(tl.generatedTranscriptStrings()) + "\n\n"
	result += "(TRANSLATION LANGUAGES)\n"
	result += tl.getLanguageDescription(tl.translationLanguageStrings())
	return result
}

func (tl *TranscriptList) manuallyCreatedTranscriptStrings() []string {
	strings := make([]string, 0, len(tl.ManuallyCreatedTranscripts))
	for _, transcript := range tl.ManuallyCreatedTranscripts {
		strings = append(strings, transcript.String())
	}
	return strings
}

func (tl *TranscriptList) generatedTranscriptStrings() []string {
	strings := make([]string, 0, len(tl.GeneratedTranscripts))
	for _, transcript := range tl.GeneratedTranscripts {
		strings = append(strings, transcript.String())
	}
	return strings
}

func (tl *TranscriptList) translationLanguageStrings() []string {
	strings := make([]string, 0, len(tl.TranslationLanguages))
	for _, tl := range tl.TranslationLanguages {
		strings = append(strings, tl.LanguageCode+" (\""+tl.Language+"\")")
	}
	return strings
}

func (tl *TranscriptList) getLanguageDescription(strings []string) string {
	if len(strings) == 0 {
		return "None"
	}
	result := ""
	for _, s := range strings {
		result += " - " + s + "\n"
	}
	return result
}

// YouTubeTranscriptApi is the main API client.
type YouTubeTranscriptApi struct {
	client *Client
}

// NewYouTubeTranscriptApi creates a new API client.
func NewYouTubeTranscriptApi() *YouTubeTranscriptApi {
	return &YouTubeTranscriptApi{
		client: NewClient(),
	}
}

// List retrieves available transcripts for a video.
func (y *YouTubeTranscriptApi) List(videoID string) (*TranscriptList, error) {
	fetcher := newTranscriptListFetcher(y.client)
	return fetcher.Fetch(videoID)
}

// Fetch retrieves transcript for a video.
func (y *YouTubeTranscriptApi) Fetch(videoID string, languages []string, preserveFormatting bool) (*FetchedTranscript, error) {
	if len(languages) == 0 {
		languages = []string{"en"}
	}

	transcriptList, err := y.List(videoID)
	if err != nil {
		return nil, err
	}

	transcript, err := transcriptList.FindTranscript(languages)
	if err != nil {
		return nil, err
	}

	return transcript.Fetch(preserveFormatting)
}
