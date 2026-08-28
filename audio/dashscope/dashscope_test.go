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

	"github.com/gonotelm-lab/multimodal/audio/schema"
	"github.com/gonotelm-lab/multimodal/callbacks"

	audios "github.com/gonotelm-lab/multimodal/audio"
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
		Text:  "今天天气真好，适合出去散步。",
		Voice: "Cherry",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.URL == "" {
		t.Fatal("expected non-empty audio url")
	}
	_ = resp.AudioFormat // dashscope 不返回音频格式，留空

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("Response: %s\n", string(output))
}

func TestGenerate_WithLanguage(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Text:     "Today is a beautiful day for a walk.",
		Voice:    "Stella",
		Language: schema.LanguageEnglish,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.URL == "" {
		t.Fatal("expected non-empty audio url")
	}

	fmt.Printf("Audio URL: %s\n", resp.URL)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_WithInstruction(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t), Model: "cosyvoice-v3-flash"})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Text:        "各位观众朋友们大家好，欢迎收看今天的节目。",
		Voice:       "longanyang",
		Instruction: "你正在进行闲聊互动，你说话的情感是neutral。",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.URL == "" {
		t.Fatal("expected non-empty audio url")
	}

	fmt.Printf("Audio URL: %s\n", resp.URL)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_InvalidAPIKey(t *testing.T) {
	gen, err := New(Config{APIKey: "invalid-key"})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	_, err = gen.Generate(t.Context(), &schema.Request{
		Text:  "test",
		Voice: "Cherry",
	})
	if err == nil {
		t.Fatal("expected error with invalid api key")
	}

	fmt.Printf("Expected error: %v\n", err)
}

func TestGenerate_EmptyText(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	_, err = gen.Generate(t.Context(), &schema.Request{
		Text:  "",
		Voice: "Cherry",
	})
	if err == nil {
		t.Fatal("expected error with empty text")
	}

	fmt.Printf("Expected error: %v\n", err)
}

func TestGenerate_EmptyVoice(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	_, err = gen.Generate(t.Context(), &schema.Request{
		Text:  "test",
		Voice: "",
	})
	if err == nil {
		t.Fatal("expected error with empty voice")
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
	gen, err := New(Config{APIKey: "test-key"}, audios.WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewBufferString(`{
				"output": {"audio": {"url": "https://example.com/audio.wav", "expires_at": 1750000000}},
				"request_id": "req-123"
			}`)),
			Header: make(http.Header),
		}, nil
	})}))

	var starts, ends, errs []string
	h := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			starts = append(starts, info.Type)
			ci := audios.ConvCallbackInput(input)
			if ci == nil || ci.Request == nil {
				t.Errorf("expected audio.CallbackInput with Request")
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			ends = append(ends, info.Type)
			co := audios.ConvCallbackOutput(output)
			if co == nil || co.Response == nil {
				t.Errorf("expected audio.CallbackOutput with Response")
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			errs = append(errs, info.Type)
			return ctx
		}).
		Build()

	ctx := callbacks.WithCallbacks(t.Context(), h)
	resp, err := gen.Generate(ctx, &schema.Request{Text: "hello", Voice: "Cherry"})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.URL == "" {
		t.Fatal("expected non-empty audio url")
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
	_, err = gen.Generate(ctx, &schema.Request{Text: "", Voice: "Cherry"})
	if err == nil {
		t.Fatal("expected error with empty text")
	}
	if len(starts) != 1 || len(errs) != 1 || len(ends) != 0 {
		t.Fatalf("expected 1 start and 1 error, got starts=%v ends=%v errs=%v", starts, ends, errs)
	}
}
