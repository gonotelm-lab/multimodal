package schema

import "io"

type ResponseFormat string

const (
	ResponseFormatURL   ResponseFormat = "url"
	ResponseFormatBytes ResponseFormat = "bytes"
)

type Response struct {
	ResponseFormat ResponseFormat

	// either URL or Reader must be set
	URL         string
	Reader      io.ReadCloser
	AudioFormat string
	Usage       *Usage
	Extras      map[string]any
}

// Usage is a cross-provider billing summary.
// Characters and TokenUsage are independently optional.
type Usage struct {
	Characters *int64
	TokenUsage *TokenUsage
}

// TokenUsage holds token billing. CachedInputTokens is 0 when the provider
// does not report cache hits.
type TokenUsage struct {
	InputTokens       int64
	CachedInputTokens int64
	OutputTokens      int64
	TotalTokens       int64
}
