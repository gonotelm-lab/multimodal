package util

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gonotelm-lab/multimodal/audio/schema"
)

type resolveOption struct {
	ctx        context.Context
	httpClient *http.Client
}

type ResolveOption func(*resolveOption)

func WithResolveContext(ctx context.Context) ResolveOption {
	return func(o *resolveOption) {
		if ctx != nil {
			o.ctx = ctx
		}
	}
}

func WithResolveHttpClient(client *http.Client) ResolveOption {
	return func(o *resolveOption) {
		if client != nil {
			o.httpClient = client
		}
	}
}

func ResolveResponse(r *schema.Response, opts ...ResolveOption) (io.ReadCloser, error) {
	if r == nil {
		return nil, fmt.Errorf("empty audio response")
	}

	switch r.ResponseFormat {
	case schema.ResponseFormatBytes:
		if r.Reader == nil {
			return nil, fmt.Errorf("audio response has no reader")
		}
		return r.Reader, nil
	case schema.ResponseFormatURL:
		opt := &resolveOption{
			ctx:        context.Background(),
			httpClient: http.DefaultClient,
		}
		for _, o := range opts {
			o(opt)
		}

		req, err := http.NewRequestWithContext(opt.ctx, http.MethodGet, r.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("build audio download request failed: %w", err)
		}
		httpResp, err := opt.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download audio failed: %w", err)
		}

		if httpResp.StatusCode != http.StatusOK {
			io.Copy(io.Discard, httpResp.Body)
			httpResp.Body.Close()
			return nil, fmt.Errorf("download audio failed: status=%d", httpResp.StatusCode)
		}

		return httpResp.Body, nil
	}

	return nil, fmt.Errorf("audio response format not supported: %s", r.ResponseFormat)
}
