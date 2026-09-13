// Package transcriptmcp serves the get_youtube_transcript tool over MCP.
//
// The server is built from a transcript service base URL and an HTTP
// client, then served on whatever transport the caller chooses (stdio in
// production, an in-memory transport in tests).
package transcriptmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codychambers/youtube-transcript/backend/internal/transcriptapi"
)

const toolName = "get_youtube_transcript"

// NewServer builds an MCP server exposing get_youtube_transcript, which
// fetches transcripts from the transcript service at baseURL using client.
func NewServer(baseURL string, client *http.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "youtube-transcript", Version: "0.1.0"}, nil)

	svc := &transcriptTool{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        toolName,
		Description: "Fetch a YouTube video's transcript via the local transcript service. Accepts a watch/short/embed URL or a bare video ID.",
		InputSchema: inputSchema,
	}, svc.call)

	return server
}

type transcriptArgs struct {
	URL               string `json:"url"`
	Lang              string `json:"lang,omitempty"`
	IncludeTimestamps *bool  `json:"includeTimestamps,omitempty"`
}

var inputSchema = &jsonschema.Schema{
	Type: "object",
	Properties: map[string]*jsonschema.Schema{
		"url": {
			Type:        "string",
			Description: "YouTube watch/short/embed URL, or a bare 11-character video ID.",
		},
		"lang": {
			Type:        "string",
			Description: "Language code, or comma-separated fallback codes. Defaults to the service's default.",
		},
		"includeTimestamps": {
			Type:        "boolean",
			Description: "Include [mm:ss] timestamps in the transcript. Defaults to true.",
		},
	},
	Required: []string{"url"},
}

type transcriptTool struct {
	baseURL string
	client  *http.Client
}

func (t *transcriptTool) call(ctx context.Context, _ *mcp.CallToolRequest, in transcriptArgs) (*mcp.CallToolResult, any, error) {
	includeTimestamps := true
	if in.IncludeTimestamps != nil {
		includeTimestamps = *in.IncludeTimestamps
	}

	body, err := json.Marshal(transcriptapi.Request{
		URL:               in.URL,
		Lang:              in.Lang,
		IncludeTimestamps: includeTimestamps,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("encode transcript request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/api/transcript", bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("build transcript request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return errorResult(fmt.Sprintf("could not reach the transcript service at %s: %v (is the Mac on the LAN/VPN?)", t.baseURL, err)), nil, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read transcript response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return errorResult(serviceErrorMessage(resp.StatusCode, respBody)), nil, nil
	}

	var out transcriptapi.Response
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, nil, fmt.Errorf("decode transcript response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: formatResult(&out, includeTimestamps)}},
	}, nil, nil
}

// serviceErrorMessage builds the isError text for a non-2xx response,
// including the status and, when present, the service's own {error} message
// (or its raw body, if that doesn't parse as the expected shape).
func serviceErrorMessage(status int, body []byte) string {
	var errResp transcriptapi.ErrorResponse
	msg := ""
	if json.Unmarshal(body, &errResp) == nil {
		msg = errResp.Error
	}
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if msg == "" {
		return fmt.Sprintf("transcript service returned %d", status)
	}
	return fmt.Sprintf("transcript service returned %d: %s", status, msg)
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

func formatResult(r *transcriptapi.Response, includeTimestamps bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Video ID: %s\n", r.VideoID)
	if r.VideoTitle != "" {
		fmt.Fprintf(&b, "Title: %s\n", r.VideoTitle)
	}
	if r.ChannelTitle != "" {
		fmt.Fprintf(&b, "Channel: %s\n", r.ChannelTitle)
	}
	fmt.Fprintf(&b, "Language: %s (%s)\n", r.Language, r.Lang)
	fmt.Fprintf(&b, "Auto-generated: %s\n", yesNo(r.IsGenerated))
	fmt.Fprintf(&b, "URL: https://youtu.be/%s\n", r.VideoID)
	b.WriteByte('\n')

	text := r.Text
	if includeTimestamps && r.TextTimestamped != nil {
		text = *r.TextTimestamped
	}
	b.WriteString(text)

	return b.String()
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
