package youtube

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	// WatchURL is the URL pattern for watching a YouTube video.
	WatchURL = "https://www.youtube.com/watch?v=%s"

	// InnertubeAPIURL is the URL pattern for the InnerTube API.
	InnertubeAPIURL = "https://www.youtube.com/youtubei/v1/player?key=%s"

	// ClientName is the client name used for InnerTube API.
	ClientName = "ANDROID"

	// ClientVersion is the client version used for InnerTube API.
	ClientVersion = "20.10.38"
)

// InnertubeContext represents the context for InnerTube API requests.
type InnertubeContext struct {
	Client struct {
		ClientName    string `json:"clientName"`
		ClientVersion string `json:"clientVersion"`
	} `json:"client"`
}

// NewInnertubeContext creates a new InnertubeContext.
func NewInnertubeContext() InnertubeContext {
	ctx := InnertubeContext{}
	ctx.Client.ClientName = ClientName
	ctx.Client.ClientVersion = ClientVersion
	return ctx
}

// InnertubeRequest represents a request to the InnerTube API.
type InnertubeRequest struct {
	Context InnertubeContext `json:"context"`
	VideoID string           `json:"videoId"`
}

// NewInnertubeRequest creates a new InnertubeRequest.
func NewInnertubeRequest(videoID string) InnertubeRequest {
	return InnertubeRequest{
		Context: NewInnertubeContext(),
		VideoID: videoID,
	}
}

// PlayabilityStatus represents the playability status from InnerTube response.
type PlayabilityStatus struct {
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	ErrorScreen *struct {
		PlayerErrorMessageRenderer *struct {
			Subreason *struct {
				Runs []struct {
					Text string `json:"text"`
				} `json:"runs"`
			} `json:"subreason"`
		} `json:"playerErrorMessageRenderer"`
	} `json:"errorScreen,omitempty"`
}

// CaptionTrack represents a caption track from InnerTube response.
type CaptionTrack struct {
	BaseURL         string `json:"baseUrl"`
	Name            *struct {
		Runs []struct {
			Text string `json:"text"`
		} `json:"runs"`
	} `json:"name"`
	LanguageCode string `json:"languageCode"`
	Kind         string `json:"kind,omitempty"`
	IsTranslatable bool  `json:"isTranslatable,omitempty"`
}

// TranslationLanguageInfo represents a translation language from InnerTube response.
type TranslationLanguageInfo struct {
	LanguageName *struct {
		Runs []struct {
			Text string `json:"text"`
		} `json:"runs"`
	} `json:"languageName"`
	LanguageCode string `json:"languageCode"`
}

// CaptionsRenderer represents the captions renderer from InnerTube response.
type CaptionsRenderer struct {
	CaptionTracks          []CaptionTrack           `json:"captionTracks"`
	TranslationLanguages   []TranslationLanguageInfo `json:"translationLanguages,omitempty"`
}

// InnertubeResponse represents a response from the InnerTube API.
type InnertubeResponse struct {
	PlayabilityStatus *PlayabilityStatus `json:"playabilityStatus"`
	Captions          *struct {
		PlayerCaptionsTracklistRenderer *CaptionsRenderer `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
}

var (
	// innertubeAPIKeyPattern extracts the INNERTUBE_API_KEY from HTML
	innertubeAPIKeyPattern = regexp.MustCompile(`"INNERTUBE_API_KEY":\s*"([a-zA-Z0-9_-]+)"`)

	// consentValuePattern extracts the consent value from HTML
	consentValuePattern = regexp.MustCompile(`name="v" value="(.*?)"`)
)

// TranscriptListFetcher fetches transcript lists from YouTube.
type TranscriptListFetcher struct {
	client *Client
}

// newTranscriptListFetcher creates a new TranscriptListFetcher.
func newTranscriptListFetcher(client *Client) *TranscriptListFetcher {
	return &TranscriptListFetcher{
		client: client,
	}
}

// Fetch retrieves the transcript list for a video.
func (f *TranscriptListFetcher) Fetch(videoID string) (*TranscriptList, error) {
	captionsJSON, err := f.fetchCaptionsJSON(videoID)
	if err != nil {
		return nil, err
	}

	return buildTranscriptList(f.client, videoID, captionsJSON)
}

// fetchCaptionsJSON fetches and parses the captions JSON from YouTube.
func (f *TranscriptListFetcher) fetchCaptionsJSON(videoID string) (*CaptionsRenderer, error) {
	html, err := f.fetchVideoHTML(videoID)
	if err != nil {
		return nil, err
	}

	apiKey, err := f.extractInnertubeAPIKey(html, videoID)
	if err != nil {
		return nil, err
	}

	innertubeData, err := f.fetchInnertubeData(videoID, apiKey)
	if err != nil {
		return nil, err
	}

	return f.extractCaptionsJSON(innertubeData, videoID)
}

// fetchVideoHTML fetches the HTML for a video page.
func (f *TranscriptListFetcher) fetchVideoHTML(videoID string) (string, error) {
	url := fmt.Sprintf(WatchURL, videoID)
	html, err := f.client.Get(url)
	if err != nil {
		if e, ok := err.(*YouTubeRequestFailed); ok {
			e.VideoID = videoID
		}
		if e, ok := err.(*IpBlocked); ok {
			e.VideoID = videoID
		}
		return "", err
	}

	// Check for consent page
	if strings.Contains(html, `action="https://consent.youtube.com/s"`) {
		err = f.createConsentCookie(html, videoID)
		if err != nil {
			return "", err
		}

		// Retry after setting consent cookie
		html, err = f.client.Get(url)
		if err != nil {
			if e, ok := err.(*YouTubeRequestFailed); ok {
				e.VideoID = videoID
			}
			if e, ok := err.(*IpBlocked); ok {
				e.VideoID = videoID
			}
			return "", err
		}

		if strings.Contains(html, `action="https://consent.youtube.com/s"`) {
			return "", NewFailedToCreateConsentCookie(videoID)
		}
	}

	return html, nil
}

// createConsentCookie creates and sets a consent cookie.
func (f *TranscriptListFetcher) createConsentCookie(html string, videoID string) error {
	matches := consentValuePattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return NewFailedToCreateConsentCookie(videoID)
	}

	// Note: Go's http.Client uses a CookieJar interface
	// For now, this is a placeholder for future cookie support
	// The consent cookie would need to be set on the client's cookie jar
	return nil
}

// extractInnertubeAPIKey extracts the InnerTube API key from HTML.
func (f *TranscriptListFetcher) extractInnertubeAPIKey(html string, videoID string) (string, error) {
	matches := innertubeAPIKeyPattern.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return matches[1], nil
	}

	if strings.Contains(html, `class="g-recaptcha"`) {
		return "", NewIpBlocked(videoID)
	}

	return "", NewYouTubeDataUnparsable(videoID)
}

// fetchInnertubeData fetches data from the InnerTube API.
func (f *TranscriptListFetcher) fetchInnertubeData(videoID string, apiKey string) (*InnertubeResponse, error) {
	request := NewInnertubeRequest(videoID)
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(InnertubeAPIURL, apiKey)
	response, err := f.client.Post(url, string(jsonData))
	if err != nil {
		if e, ok := err.(*YouTubeRequestFailed); ok {
			e.VideoID = videoID
		}
		if e, ok := err.(*IpBlocked); ok {
			e.VideoID = videoID
		}
		return nil, err
	}

	var innertubeResponse InnertubeResponse
	err = json.Unmarshal([]byte(response), &innertubeResponse)
	if err != nil {
		return nil, NewYouTubeDataUnparsable(videoID)
	}

	return &innertubeResponse, nil
}

// extractCaptionsJSON extracts captions from InnerTube response.
func (f *TranscriptListFetcher) extractCaptionsJSON(innertubeData *InnertubeResponse, videoID string) (*CaptionsRenderer, error) {
	f.assertPlayability(innertubeData.PlayabilityStatus, videoID)

	if innertubeData.Captions == nil || innertubeData.Captions.PlayerCaptionsTracklistRenderer == nil {
		return nil, NewTranscriptsDisabled(videoID)
	}

	captions := innertubeData.Captions.PlayerCaptionsTracklistRenderer
	if len(captions.CaptionTracks) == 0 {
		return nil, NewTranscriptsDisabled(videoID)
	}

	return captions, nil
}

// assertPlayability checks if the video is playable.
func (f *TranscriptListFetcher) assertPlayability(playabilityStatus *PlayabilityStatus, videoID string) error {
	if playabilityStatus == nil {
		return nil
	}

	switch playabilityStatus.Status {
	case "OK":
		return nil
	case "LOGIN_REQUIRED":
		if strings.Contains(playabilityStatus.Reason, "Sign in to confirm you're not a bot") {
			return NewRequestBlocked(videoID)
		}
		if strings.Contains(playabilityStatus.Reason, "This video may be inappropriate for some users") {
			return NewAgeRestricted(videoID)
		}
	case "ERROR":
		if strings.Contains(playabilityStatus.Reason, "This video is unavailable") {
			if strings.HasPrefix(videoID, "http://") || strings.HasPrefix(videoID, "https://") {
				return NewInvalidVideoID(videoID)
			}
			return NewVideoUnavailable(videoID)
		}
	}

	// Extract subreasons if available
	subreasons := []string{}
	if playabilityStatus.ErrorScreen != nil &&
		playabilityStatus.ErrorScreen.PlayerErrorMessageRenderer != nil &&
		playabilityStatus.ErrorScreen.PlayerErrorMessageRenderer.Subreason != nil {
		for _, run := range playabilityStatus.ErrorScreen.PlayerErrorMessageRenderer.Subreason.Runs {
			subreasons = append(subreasons, run.Text)
		}
	}

	return NewVideoUnplayable(videoID, playabilityStatus.Reason, subreasons)
}

// buildTranscriptList builds a TranscriptList from captions JSON.
func buildTranscriptList(client *Client, videoID string, captions *CaptionsRenderer) (*TranscriptList, error) {
	// Extract translation languages
	translationLanguages := make([]TranslationLanguage, 0, len(captions.TranslationLanguages))
	for _, tl := range captions.TranslationLanguages {
		if tl.LanguageName != nil && len(tl.LanguageName.Runs) > 0 {
			translationLanguages = append(translationLanguages, TranslationLanguage{
				Language:     tl.LanguageName.Runs[0].Text,
				LanguageCode: tl.LanguageCode,
			})
		}
	}

	manuallyCreatedTranscripts := make(map[string]*Transcript)
	generatedTranscripts := make(map[string]*Transcript)

	for _, caption := range captions.CaptionTracks {
		var transcriptDict map[string]*Transcript
		if caption.Kind == "asr" {
			transcriptDict = generatedTranscripts
		} else {
			transcriptDict = manuallyCreatedTranscripts
		}

		language := ""
		if caption.Name != nil && len(caption.Name.Runs) > 0 {
			language = caption.Name.Runs[0].Text
		}

		// Remove &fmt=srv3 from base URL
		baseURL := strings.ReplaceAll(caption.BaseURL, "&fmt=srv3", "")

		var tl []TranslationLanguage
		if caption.IsTranslatable {
			tl = translationLanguages
		}

		transcriptDict[caption.LanguageCode] = newTranscript(
			client,
			videoID,
			baseURL,
			language,
			caption.LanguageCode,
			caption.Kind == "asr",
			tl,
		)
	}

	return newTranscriptList(
		videoID,
		manuallyCreatedTranscripts,
		generatedTranscripts,
		translationLanguages,
	), nil
}
