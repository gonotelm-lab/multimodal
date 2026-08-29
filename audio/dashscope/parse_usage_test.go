package dashscope

import (
	"testing"

	"github.com/gonotelm-lab/multimodal/audio/schema"
)

func TestParseResponse_UsageCharactersOnly(t *testing.T) {
	body := []byte(`{
		"request_id": "rid-flash",
		"output": {"audio": {"url": "https://example.com/a.wav", "expires_at": 1}},
		"usage": {"input_tokens": 0, "output_tokens": 0, "characters": 195}
	}`)

	resp, err := parseResponse(body)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Usage == nil {
		t.Fatal("expected Usage")
	}
	if resp.Usage.Characters == nil || *resp.Usage.Characters != 195 {
		t.Fatalf("Characters: got %#v want 195", resp.Usage.Characters)
	}
	if resp.Usage.TokenUsage != nil {
		t.Fatalf("TokenUsage: got %#v want nil", resp.Usage.TokenUsage)
	}
	if got, ok := resp.Extras[ExtraCharacters].(int); !ok || got != 195 {
		t.Fatalf("Extras characters: got %#v", resp.Extras[ExtraCharacters])
	}
}

func TestParseResponse_UsageTokensOnly(t *testing.T) {
	body := []byte(`{
		"request_id": "rid-tts",
		"output": {"audio": {"url": "https://example.com/a.wav", "expires_at": 1}},
		"usage": {
			"input_tokens": 76,
			"output_tokens": 1045,
			"total_tokens": 1121,
			"characters": 0,
			"input_tokens_details": {"text_tokens": 76},
			"output_tokens_details": {"audio_tokens": 1045, "text_tokens": 0}
		}
	}`)

	resp, err := parseResponse(body)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Usage == nil {
		t.Fatal("expected Usage")
	}
	if resp.Usage.Characters != nil {
		t.Fatalf("Characters: got %#v want nil", resp.Usage.Characters)
	}
	tu := resp.Usage.TokenUsage
	if tu == nil {
		t.Fatal("expected TokenUsage")
	}
	want := schema.TokenUsage{InputTokens: 76, OutputTokens: 1045, TotalTokens: 1121}
	if *tu != want {
		t.Fatalf("TokenUsage: got %+v want %+v", *tu, want)
	}
	if got, ok := resp.Extras[ExtraInputTokens].(int); !ok || got != 76 {
		t.Fatalf("Extras input_tokens: %#v", resp.Extras[ExtraInputTokens])
	}
	if got, ok := resp.Extras[ExtraInputTextTokens].(int); !ok || got != 76 {
		t.Fatalf("Extras input_text_tokens: %#v", resp.Extras[ExtraInputTextTokens])
	}
	if got, ok := resp.Extras[ExtraOutputAudioTokens].(int); !ok || got != 1045 {
		t.Fatalf("Extras output_audio_tokens: %#v", resp.Extras[ExtraOutputAudioTokens])
	}
}

func TestParseResponse_UsageEmpty(t *testing.T) {
	body := []byte(`{
		"output": {"audio": {"url": "https://example.com/a.wav"}},
		"usage": {}
	}`)

	resp, err := parseResponse(body)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Usage != nil {
		t.Fatalf("Usage: got %#v want nil", resp.Usage)
	}
}
