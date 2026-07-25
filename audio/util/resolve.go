package util

import (
	"context"
	"io"
	"net/http"

	"github.com/gonotelm-lab/multimodal/error"
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
		return nil, errx.New(errx.KindInvalidArgument, "empty audio response")
	}

	switch r.ResponseFormat {
	case schema.ResponseFormatBytes:
		if r.Reader == nil {
			return nil, errx.New(errx.KindInvalidArgument, "audio response has no reader")
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
			return nil, errx.Wrap(err, errx.KindInvalidArgument, "build audio download request failed")
		}
		httpResp, err := opt.httpClient.Do(req)
		if err != nil {
			return nil, errx.Wrap(err, errx.KindNetwork, "download audio failed")
		}

		if httpResp.StatusCode != http.StatusOK {
			io.Copy(io.Discard, httpResp.Body)
			httpResp.Body.Close()
			return nil, errx.Newf(errx.FromHTTPStatus(httpResp.StatusCode),
				"download audio failed: status=%d", httpResp.StatusCode)
		}

		return httpResp.Body, nil
	}

	return nil, errx.Newf(errx.KindInvalidArgument,
		"audio response format not supported: %s", r.ResponseFormat)
}
