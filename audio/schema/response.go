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
	Extras      map[string]any
}
