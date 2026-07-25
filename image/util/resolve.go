package util

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/gonotelm-lab/multimodal/error"
	"github.com/gonotelm-lab/multimodal/image/schema"
)

type resolveOption struct {
	ctx        context.Context
	httpClient *http.Client
}

type ResolveResponseOption func(*resolveOption)

func WithResolveContext(ctx context.Context) ResolveResponseOption {
	return func(o *resolveOption) {
		if ctx != nil {
			o.ctx = ctx
		}
	}
}

func WithResolveHttpClient(client *http.Client) ResolveResponseOption {
	return func(o *resolveOption) {
		if client != nil {
			o.httpClient = client
		}
	}
}

func ResolveResponse(r *schema.Response, opts ...ResolveResponseOption) (io.ReadCloser, error) {
	if r == nil {
		return nil, errx.New(errx.KindInvalidArgument, "empty images response")
	}

	switch r.ResponseFormat {
	case schema.ResponseFormatBase64:
		data, err := base64.StdEncoding.DecodeString(r.ImageBase64)
		if err != nil {
			return nil, errx.Wrap(err, errx.KindInvalidArgument, "decode image base64 failed")
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	case schema.ResponseFormatURL:
		opt := &resolveOption{
			ctx:        context.Background(),
			httpClient: http.DefaultClient,
		}
		for _, o := range opts {
			o(opt)
		}

		req, err := http.NewRequestWithContext(opt.ctx, http.MethodGet, r.ImageURL, nil)
		if err != nil {
			return nil, errx.Wrap(err, errx.KindInvalidArgument, "build image download request failed")
		}
		httpResp, err := opt.httpClient.Do(req)
		if err != nil {
			return nil, errx.Wrap(err, errx.KindNetwork, "download image failed")
		}

		if httpResp.StatusCode != http.StatusOK {
			io.Copy(io.Discard, httpResp.Body)
			httpResp.Body.Close()
			return nil, errx.Newf(errx.FromHTTPStatus(httpResp.StatusCode),
				"download image failed: status=%d", httpResp.StatusCode)
		}

		return httpResp.Body, nil
	}

	return nil, errx.Newf(errx.KindInvalidArgument,
		"images response format not supported: %s", r.ResponseFormat)
}
