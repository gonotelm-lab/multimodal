package minimax

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
	"github.com/gonotelm-lab/multimodal/audio/util"
	"github.com/gonotelm-lab/multimodal/callbacks"

	audios "github.com/gonotelm-lab/multimodal/audio"
)

func getAPIKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("GONOTELM_MINIMAX_APIKEY")
	if key == "" {
		t.Skip("GONOTELM_MINIMAX_APIKEY not set, skipping integration test")
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
		Voice: "male-qn-qingse",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.ResponseFormat != schema.ResponseFormatBytes {
		t.Fatalf("expected bytes format, got %s", resp.ResponseFormat)
	}
	if resp.Reader == nil {
		t.Fatal("expected non-nil reader")
	}
	defer resp.Reader.Close()

	n, err := io.Copy(io.Discard, resp.Reader)
	if err != nil {
		t.Fatalf("read audio failed: %v", err)
	}
	if n == 0 {
		t.Fatal("expected non-empty audio bytes")
	}

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
		Voice:    "English_radiant_girl",
		Language: schema.LanguageEnglish,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.Reader == nil {
		t.Fatal("expected non-nil reader")
	}
	defer resp.Reader.Close()

	fmt.Printf("AudioFormat: %s\n", resp.AudioFormat)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_WithEmotion(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Text:  "今天是不是很开心呀，当然了！",
		Voice: "male-qn-qingse",
	}, WithEmotion(EmotionHappy), WithSpeed(1.0))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	defer resp.Reader.Close()

	fmt.Printf("AudioFormat: %s\n", resp.AudioFormat)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_WithOutputFormatURL(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Text:  "测试 url 输出模式。",
		Voice: "male-qn-qingse",
	}, WithOutputFormat(OutputFormatURL))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.ResponseFormat != schema.ResponseFormatURL {
		t.Fatalf("expected url format, got %s", resp.ResponseFormat)
	}
	if resp.URL == "" {
		t.Fatal("expected non-empty audio url")
	}

	reader, err := util.ResolveResponse(resp)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	defer reader.Close()

	n, err := io.Copy(io.Discard, reader)
	if err != nil {
		t.Fatalf("download audio failed: %v", err)
	}
	if n == 0 {
		t.Fatal("expected non-empty audio bytes")
	}

	fmt.Printf("Audio URL: %s\n", resp.URL)
	fmt.Printf("AudioFormat: %s\n", resp.AudioFormat)
}

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(Config{APIKey: ""})
	if err == nil {
		t.Fatal("expected error with empty api key")
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
		Voice: "male-qn-qingse",
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

func TestGenerate_InvalidAPIKey(t *testing.T) {
	gen, err := New(Config{APIKey: "invalid-key"})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	_, err = gen.Generate(t.Context(), &schema.Request{
		Text:  "test",
		Voice: "male-qn-qingse",
	})
	if err == nil {
		t.Fatal("expected error with invalid api key")
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
				"data": {"audio": "6865782d74657374"},
				"base_resp": {"status_code": 0, "status_msg": ""},
				"extra_info": {"audio_format": "mp3"}
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
	resp, err := gen.Generate(ctx, &schema.Request{Text: "hello", Voice: "male-qn-qingse"})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.Reader == nil {
		t.Fatal("expected non-nil audio reader")
	}
	if len(starts) != 1 || starts[0] != "minimax" {
		t.Fatalf("expected 1 start with type minimax, got %v", starts)
	}
	if len(ends) != 1 || ends[0] != "minimax" {
		t.Fatalf("expected 1 end with type minimax, got %v", ends)
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
	_, err = gen.Generate(ctx, &schema.Request{Text: "", Voice: "male-qn-qingse"})
	if err == nil {
		t.Fatal("expected error with empty text")
	}
	if len(starts) != 1 || len(errs) != 1 || len(ends) != 0 {
		t.Fatalf("expected 1 start and 1 error, got starts=%v ends=%v errs=%v", starts, ends, errs)
	}
}
