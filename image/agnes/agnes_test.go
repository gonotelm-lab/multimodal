package agnes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gonotelm-lab/multimodal/callbacks"
	images "github.com/gonotelm-lab/multimodal/image"
	"github.com/gonotelm-lab/multimodal/image/schema"
	"github.com/gonotelm-lab/multimodal/image/util"
)

func getAPIKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("GONOTELM_AGNES_API_KEY")
	if key == "" {
		t.Skip("GONOTELM_AGNES_APIKEY not set, skipping integration test")
	}
	return key
}

func TestGenerate_Basic(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Prompt: "一只可爱的橘猫坐在窗台上，阳光照射进来，温暖舒适的氛围",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("Response: %s\n", string(output))
}

func TestGenerate_WithBase64Output(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Prompt:         "一只可爱的柯基狗狗在草地上玩耍，阳光照射进来，温暖舒适的氛围",
		ResponseFormat: schema.ResponseFormatBase64,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.ImageBase64 == "" {
		t.Fatalf("expected non-empty image base64")
	}

	reader, err := util.ResolveResponse(resp)
	if err != nil {
		t.Fatalf("resolve response failed: %v", err)
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read response failed: %v", err)
	}
	if err := os.WriteFile("/tmp/image.png", b, 0644); err != nil {
		t.Fatalf("write image to file failed: %v", err)
	}
}

func TestGenerate_WithURLOutput(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Prompt: "a beautiful landscape with a river and a mountain",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	reader, err := util.ResolveResponse(resp)
	if err != nil {
		t.Fatalf("resolve response failed: %v", err)
	}
	defer reader.Close()
	mimeType, err := mimetype.DetectReader(reader)
	if err != nil {
		t.Fatalf("detect mime type failed: %v", err)
	}
	fmt.Printf("mime type: %s\n", mimeType.String())

	b, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read response failed: %v", err)
	}

	if err := os.WriteFile("/tmp/image.png", b, 0644); err != nil {
		t.Fatalf("write image to file failed: %v", err)
	}

}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGenerate_Callbacks(t *testing.T) {
	gen, err := New(Config{APIKey: "test-key"}, images.WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewBufferString(`{
				"created": 1750000000,
				"data": [{"url": "https://example.com/img.png"}]
			}`)),
			Header: make(http.Header),
		}, nil
	})}))

	var starts, ends, errs []string
	h := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			starts = append(starts, info.Type)
			ci := images.ConvCallbackInput(input)
			if ci == nil || ci.Request == nil {
				t.Errorf("expected image.CallbackInput with Request")
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			ends = append(ends, info.Type)
			co := images.ConvCallbackOutput(output)
			if co == nil || co.Response == nil {
				t.Errorf("expected image.CallbackOutput with Response")
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			errs = append(errs, info.Type)
			return ctx
		}).
		Build()

	ctx := callbacks.WithCallbacks(t.Context(), h)
	resp, err := gen.Generate(ctx, &schema.Request{Prompt: "test prompt"})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.ImageURL == "" {
		t.Fatal("expected non-empty image url")
	}
	if len(starts) != 1 || starts[0] != "agnes" {
		t.Fatalf("expected 1 start with type agnes, got %v", starts)
	}
	if len(ends) != 1 || ends[0] != "agnes" {
		t.Fatalf("expected 1 end with type agnes, got %v", ends)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no error callback, got %v", errs)
	}
}

func TestGenerate_CallbacksOnValidationError(t *testing.T) {
	gen, err := New(Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	var starts, ends, errs []string
	h := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			starts = append(starts, info.Type)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			ends = append(ends, info.Type)
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			errs = append(errs, info.Type)
			return ctx
		}).
		Build()

	ctx := callbacks.WithCallbacks(t.Context(), h)
	_, err = gen.Generate(ctx, &schema.Request{Prompt: ""})
	if err == nil {
		t.Fatal("expected error with empty prompt")
	}
	if len(starts) != 1 || len(errs) != 1 || len(ends) != 0 {
		t.Fatalf("expected 1 start and 1 error, got starts=%v ends=%v errs=%v", starts, ends, errs)
	}
}
