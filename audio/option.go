package audio

import (
	"net/http"
	"time"
)

type Option func(*CallOptions)

type CallOptions struct {
	Extra map[string]any
}

func BuildCallOptions(opts ...Option) *CallOptions {
	co := &CallOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(co)
		}
	}
	return co
}

func WithExtra(key string, value any) Option {
	return func(o *CallOptions) {
		if o.Extra == nil {
			o.Extra = make(map[string]any)
		}
		o.Extra[key] = value
	}
}

type ClientOption func(*ClientOptions)

type ClientOptions struct {
	HTTPClient *http.Client
}

func BuildClientOptions(timeout time.Duration, opts ...ClientOption) *ClientOptions {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	co := &ClientOptions{
		HTTPClient: &http.Client{Timeout: timeout},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(co)
		}
	}
	return co
}

func WithHTTPClient(client *http.Client) ClientOption {
	return func(o *ClientOptions) {
		if client != nil {
			o.HTTPClient = client
		}
	}
}
