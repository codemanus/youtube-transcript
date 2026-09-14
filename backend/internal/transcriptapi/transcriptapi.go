// Package transcriptapi defines the wire types for POST /api/transcript,
// shared by the HTTP handler and any client that calls it (e.g. the MCP server).
package transcriptapi

// Request is the JSON body of POST /api/transcript.
type Request struct {
	URL               string `json:"url"`
	Lang              string `json:"lang"`
	IncludeTimestamps bool   `json:"includeTimestamps"`
}

// Response is the JSON body returned on success.
type Response struct {
	VideoID         string  `json:"videoId"`
	VideoTitle      string  `json:"videoTitle,omitempty"`
	ChannelTitle    string  `json:"channelTitle,omitempty"`
	Lang            string  `json:"lang"`
	Language        string  `json:"language"`
	IsGenerated     bool    `json:"isGenerated"`
	Text            string  `json:"text"`
	TextTimestamped *string `json:"textTimestamped,omitempty"`
	SnippetCount    int     `json:"snippetCount"`
}

// ErrorResponse is the JSON body returned on failure.
type ErrorResponse struct {
	Error string `json:"error"`
}
