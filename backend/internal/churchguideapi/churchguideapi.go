// Package churchguideapi defines the wire types for GET /api/church-guide,
// shared by the HTTP handler and any client that calls it (e.g. the MCP
// server). Errors use transcriptapi.ErrorResponse's {error} shape.
package churchguideapi

// Response is the JSON body returned on success.
type Response struct {
	Title         string `json:"title"`
	WeekStartDate string `json:"weekStartDate"`
	Text          string `json:"text"`
}
