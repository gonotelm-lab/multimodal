package mimo

import (
	"encoding/base64"
	"testing"

	"github.com/gonotelm-lab/multimodal/audio/schema"
)

func TestParseResponse_UsageWithCache(t *testing.T) {
	audioB64 := base64.StdEncoding.EncodeToString([]byte("fake-audio"))
	body := []byte(`{
		"id": "cmpl-1",
		"choices": [{
			"finish_reason": "stop",
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "",
				"audio": {"id": "a1", "data": "` + audioB64 + `"}
			}
		}],
		"usage": {
			"prompt_tokens": 213,
			"completion_tokens": 97,
			"total_tokens": 310,
			"completion_tokens_details": {"reasoning_tokens": 0},
			"prompt_tokens_details": {"cached_tokens": 109}
		}
	}`)

	resp, err := parseResponse(body, "wav")
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Usage == nil || resp.Usage.Characters != nil {
		t.Fatalf("Usage.Characters: got %#v want nil Characters with Usage set", resp.Usage)
	}
	tu := resp.Usage.TokenUsage
	if tu == nil {
		t.Fatal("expected TokenUsage")
	}
	want := schema.TokenUsage{
		InputTokens:       213,
		CachedInputTokens: 109,
		OutputTokens:      97,
		TotalTokens:       310,
	}
	if *tu != want {
		t.Fatalf("TokenUsage: got %+v want %+v", *tu, want)
	}
	if got, ok := resp.Extras[ExtraPromptTokens].(int); !ok || got != 213 {
		t.Fatalf("Extras prompt_tokens: %#v", resp.Extras[ExtraPromptTokens])
	}
	if got, ok := resp.Extras[ExtraCachedTokens].(int); !ok || got != 109 {
		t.Fatalf("Extras cached_tokens: %#v", resp.Extras[ExtraCachedTokens])
	}
}

func TestParseResponse_UsageWithoutDetails(t *testing.T) {
	audioB64 := base64.StdEncoding.EncodeToString([]byte("fake-audio"))
	body := []byte(`{
		"choices": [{
			"message": {"role": "assistant", "audio": {"data": "` + audioB64 + `"}}
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 20,
			"total_tokens": 30
		}
	}`)

	resp, err := parseResponse(body, "mp3")
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	tu := resp.Usage.TokenUsage
	if tu == nil {
		t.Fatal("expected TokenUsage")
	}
	want := schema.TokenUsage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30}
	if *tu != want {
		t.Fatalf("TokenUsage: got %+v want %+v", *tu, want)
	}
	if _, ok := resp.Extras[ExtraCachedTokens]; ok {
		t.Fatalf("cached_tokens should be absent, got %#v", resp.Extras[ExtraCachedTokens])
	}
}

func TestParseResponse_NoUsage(t *testing.T) {
	audioB64 := base64.StdEncoding.EncodeToString([]byte("fake-audio"))
	body := []byte(`{
		"choices": [{
			"message": {"role": "assistant", "audio": {"data": "` + audioB64 + `"}}
		}]
	}`)

	resp, err := parseResponse(body, "wav")
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Usage != nil {
		t.Fatalf("Usage: got %#v want nil", resp.Usage)
	}
}
