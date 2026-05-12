package youtube

import (
	"fmt"
)

// YouTubeTranscriptApiException is the base error type for all YouTube transcript API errors.
type YouTubeTranscriptApiException struct {
	VideoID string
	Message string
}

func (e *YouTubeTranscriptApiException) Error() string {
	return fmt.Sprintf("Could not retrieve a transcript for the video https://www.youtube.com/watch?v=%s!", e.VideoID)
}

// CouldNotRetrieveTranscript is raised if a transcript could not be retrieved.
type CouldNotRetrieveTranscript struct {
	YouTubeTranscriptApiException
	Cause string
}

func (e *CouldNotRetrieveTranscript) Error() string {
	msg := fmt.Sprintf("Could not retrieve a transcript for the video https://www.youtube.com/watch?v=%s!", e.VideoID)
	if e.Cause != "" {
		msg += "\n This is most likely caused by:\n\n" + e.Cause
	}
	msg += "\n\nIf you are sure that the described cause is not responsible for this error " +
		"and that a transcript should be retrievable, please create an issue at " +
		"https://github.com/rahadiangg/youtube-transcript-go/issues."
	return msg
}

// YouTubeDataUnparsable is raised when the data required to fetch the transcript is not parsable.
type YouTubeDataUnparsable struct {
	CouldNotRetrieveTranscript
}

func NewYouTubeDataUnparsable(videoID string) *YouTubeDataUnparsable {
	return &YouTubeDataUnparsable{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause: "The data required to fetch the transcript is not parsable. " +
				"This should not happen, please open an issue (make sure to include the video ID)!",
		},
	}
}

// YouTubeRequestFailed is raised when a request to YouTube fails.
type YouTubeRequestFailed struct {
	CouldNotRetrieveTranscript
	Reason string
}

func NewYouTubeRequestFailed(videoID string, reason string) *YouTubeRequestFailed {
	return &YouTubeRequestFailed{
		CouldNotRetrieveTranscript: CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause:                          fmt.Sprintf("Request to YouTube failed: %s", reason),
		},
		Reason: reason,
	}
}

// VideoUnplayable is raised when the video is unplayable.
type VideoUnplayable struct {
	CouldNotRetrieveTranscript
	Reason     string
	Subreasons []string
}

func NewVideoUnplayable(videoID string, reason string, subreasons []string) *VideoUnplayable {
	return &VideoUnplayable{
		CouldNotRetrieveTranscript: CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause: fmt.Sprintf("The video is unplayable for the following reason: %s", reason),
		},
		Reason:     reason,
		Subreasons: subreasons,
	}
}

// VideoUnavailable is raised when the video is no longer available.
type VideoUnavailable struct {
	CouldNotRetrieveTranscript
}

func NewVideoUnavailable(videoID string) *VideoUnavailable {
	return &VideoUnavailable{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause:                          "The video is no longer available",
		},
	}
}

// InvalidVideoID is raised when an invalid video ID is provided.
type InvalidVideoID struct {
	CouldNotRetrieveTranscript
}

func NewInvalidVideoID(videoID string) *InvalidVideoID {
	return &InvalidVideoID{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause: "You provided an invalid video id. Make sure you are using the video id and NOT the url!\n\n" +
				"Do NOT run: `api.Fetch(\"https://www.youtube.com/watch?v=1234\")`\n" +
				"Instead run: `api.Fetch(\"1234\")`",
		},
	}
}

// RequestBlocked is raised when YouTube is blocking requests from your IP.
type RequestBlocked struct {
	CouldNotRetrieveTranscript
}

func NewRequestBlocked(videoID string) *RequestBlocked {
	return &RequestBlocked{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause: "YouTube is blocking requests from your IP. This usually is due to one of the following reasons:\n" +
				"- You have done too many requests and your IP has been blocked by YouTube\n" +
				"- You are doing requests from an IP belonging to a cloud provider (like AWS, " +
				"Google Cloud Platform, Azure, etc.). Unfortunately, most IPs from cloud " +
				"providers are blocked by YouTube.\n\n" +
				"To work around this, you can use proxies to hide your IP address.",
		},
	}
}

// IpBlocked is raised when your IP is blocked by YouTube (HTTP 429).
type IpBlocked struct {
	RequestBlocked
}

func NewIpBlocked(videoID string) *IpBlocked {
	return &IpBlocked{
		RequestBlocked: *NewRequestBlocked(videoID),
	}
}

// TranscriptsDisabled is raised when subtitles are disabled for a video.
type TranscriptsDisabled struct {
	CouldNotRetrieveTranscript
}

func NewTranscriptsDisabled(videoID string) *TranscriptsDisabled {
	return &TranscriptsDisabled{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause:                          "Subtitles are disabled for this video",
		},
	}
}

// AgeRestricted is raised when a video is age-restricted.
type AgeRestricted struct {
	CouldNotRetrieveTranscript
}

func NewAgeRestricted(videoID string) *AgeRestricted {
	return &AgeRestricted{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause: "This video is age-restricted. Therefore, you are unable to retrieve " +
				"transcripts for it without authenticating yourself.\n\n" +
				"Unfortunately, Cookie Authentication is temporarily unsupported.",
		},
	}
}

// NotTranslatable is raised when the requested language is not translatable.
type NotTranslatable struct {
	CouldNotRetrieveTranscript
}

func NewNotTranslatable(videoID string) *NotTranslatable {
	return &NotTranslatable{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause:                          "The requested language is not translatable",
		},
	}
}

// TranslationLanguageNotAvailable is raised when the requested translation language is not available.
type TranslationLanguageNotAvailable struct {
	CouldNotRetrieveTranscript
}

func NewTranslationLanguageNotAvailable(videoID string) *TranslationLanguageNotAvailable {
	return &TranslationLanguageNotAvailable{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause:                          "The requested translation language is not available",
		},
	}
}

// FailedToCreateConsentCookie is raised when consent cookie creation fails.
type FailedToCreateConsentCookie struct {
	CouldNotRetrieveTranscript
}

func NewFailedToCreateConsentCookie(videoID string) *FailedToCreateConsentCookie {
	return &FailedToCreateConsentCookie{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause:                          "Failed to automatically give consent to saving cookies",
		},
	}
}

// NoTranscriptFound is raised when no transcripts are found for the requested language codes.
type NoTranscriptFound struct {
	CouldNotRetrieveTranscript
	RequestedLanguageCodes []string
	TranscriptData         *TranscriptList
}

func NewNoTranscriptFound(videoID string, languageCodes []string, transcriptList *TranscriptList) *NoTranscriptFound {
	return &NoTranscriptFound{
		CouldNotRetrieveTranscript: CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
		},
		RequestedLanguageCodes: languageCodes,
		TranscriptData:         transcriptList,
	}
}

func (e *NoTranscriptFound) Error() string {
	return fmt.Sprintf("Could not retrieve a transcript for the video https://www.youtube.com/watch?v=%s!\n"+
		" No transcripts were found for any of the requested language codes: %v\n\n%s",
		e.VideoID, e.RequestedLanguageCodes, e.TranscriptData)
}

// PoTokenRequired is raised when a PO Token is required.
type PoTokenRequired struct {
	CouldNotRetrieveTranscript
}

func NewPoTokenRequired(videoID string) *PoTokenRequired {
	return &PoTokenRequired{
		CouldNotRetrieveTranscript{
			YouTubeTranscriptApiException: YouTubeTranscriptApiException{VideoID: videoID},
			Cause: "The requested video cannot be retrieved without a PO Token. " +
				"If this happens, please open a GitHub issue!",
		},
	}
}
