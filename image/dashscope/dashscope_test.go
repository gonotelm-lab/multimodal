package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/gonotelm-lab/multimodal/callbacks"
	images "github.com/gonotelm-lab/multimodal/image"
	"github.com/gonotelm-lab/multimodal/image/schema"
)

func getAPIKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("GONOTELM_DASHSCOPE_APIKEY")
	if key == "" {
		t.Skip("GONOTELM_DASHSCOPE_APIKEY not set, skipping integration test")
	}
	return key
}

func TestGenerate_Basic(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Model:  "qwen-image-2.0",
		Prompt: "一只可爱的橘猫坐在窗台上，阳光照射进来，温暖舒适的氛围",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("Response: %s\n", string(output))
}

func TestGenerate_WithSize(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Prompt: "一座现代化的城市夜景，高楼大厦灯火通明",
		Size:   "1024*1024",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.ImageURL == "" {
		t.Fatal("expected non-empty image url")
	}

	// 检查 extras 中是否包含尺寸信息
	if _, ok := resp.Extras["width"]; !ok {
		t.Log("warning: expected width in extras")
	}
	if _, ok := resp.Extras["height"]; !ok {
		t.Log("warning: expected height in extras")
	}

	fmt.Printf("Image URL: %s\n", resp.ImageURL)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_WithOptions(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Prompt: "一片宁静的湖泊，周围是雪山和森林，倒影清晰可见",
		Size:   "1024*1024",
	},
		WithNegativePrompt("模糊, 低质量, 变形"),
		WithPromptExtend(false),
		WithWatermark(false),
		WithSeed(42),
	)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.ImageURL == "" {
		t.Fatal("expected non-empty image url")
	}

	fmt.Printf("Image URL: %s\n", resp.ImageURL)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_InvalidAPIKey(t *testing.T) {
	gen, err := New(Config{APIKey: "invalid-key"})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	_, err = gen.Generate(t.Context(), &schema.Request{
		Prompt: "test prompt",
	})
	if err == nil {
		t.Fatal("expected error with invalid api key")
	}

	fmt.Printf("Expected error: %v\n", err)
}

func TestGenerate_EmptyPrompt(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	_, err = gen.Generate(t.Context(), &schema.Request{
		Prompt: "",
	})
	if err == nil {
		t.Fatal("expected error with empty prompt")
	}

	fmt.Printf("Expected error: %v\n", err)
}

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(Config{APIKey: ""})
	if err == nil {
		t.Fatal("expected error with empty api key")
	}

	fmt.Printf("Expected error: %v\n", err)
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
				"output": {"choices": [{"finish_reason": "stop", "message": {"role": "assistant", "content": [{"text": "", "image": "https://example.com/img.png"}]}}]},
				"request_id": "req-123"
			}`)),
			Header: make(http.Header),
		}, nil
	})}))

	var starts, ends, errs []string
	h := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			starts = append(starts, info.Type)
			ci := images.ConvCallbackInput(input)
			if ci == nil || ci.Request == nil || ci.CallOptions == nil {
				t.Errorf("expected image.CallbackInput with Request and CallOptions")
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
	if len(starts) != 1 || starts[0] != "dashscope" {
		t.Fatalf("expected 1 start with type dashscope, got %v", starts)
	}
	if len(ends) != 1 || ends[0] != "dashscope" {
		t.Fatalf("expected 1 end with type dashscope, got %v", ends)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no error callback, got %v", errs)
	}
}

func TestGenerate_CallbacksOnError(t *testing.T) {
	gen, err := New(Config{APIKey: "test-key"}, images.WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body: io.NopCloser(bytes.NewBufferString(`{"code":"InvalidParameter","message":"bad"}`)),
			Header: make(http.Header),
		}, nil
	})}))

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
	_, err = gen.Generate(ctx, &schema.Request{Prompt: "test prompt"})
	if err == nil {
		t.Fatal("expected error from http 400")
	}
	if len(starts) != 1 {
		t.Fatalf("expected 1 start, got %v", starts)
	}
	if len(ends) != 0 {
		t.Fatalf("expected no end callback on error, got %v", ends)
	}
	if len(errs) != 1 || errs[0] != "dashscope" {
		t.Fatalf("expected 1 error callback with type dashscope, got %v", errs)
	}
}

func TestGenerate_CallbacksOnValidationError(t *testing.T) {
	gen, err := New(Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	var starts, errs []string
	h := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			starts = append(starts, info.Type)
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
	if len(starts) != 1 || len(errs) != 1 {
		t.Fatalf("expected 1 start and 1 error, got starts=%v errs=%v", starts, errs)
	}
}
