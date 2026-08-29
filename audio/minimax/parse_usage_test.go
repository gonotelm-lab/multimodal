package minimax

import (
	"encoding/hex"
	"testing"
)

func TestParseResponse_UsageCharacters(t *testing.T) {
	audioHex := hex.EncodeToString([]byte("fake-audio"))
	body := []byte(`{
		"data": {"audio": "` + audioHex + `", "status": 2},
		"extra_info": {
			"audio_length": 1000,
			"usage_characters": 163,
			"word_count": 163,
			"audio_format": "mp3",
			"audio_channel": 1
		},
		"trace_id": "tid",
		"base_resp": {"status_code": 0, "status_msg": "success"}
	}`)

	resp, err := parseResponse(body)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Usage == nil || resp.Usage.Characters == nil || *resp.Usage.Characters != 163 {
		t.Fatalf("Characters: got %#v want 163", resp.Usage)
	}
	if resp.Usage.TokenUsage != nil {
		t.Fatalf("TokenUsage: got %#v want nil", resp.Usage.TokenUsage)
	}
	if got, ok := resp.Extras[ExtraUsageCharacters].(int64); !ok || got != 163 {
		t.Fatalf("Extras usage_characters: %#v", resp.Extras[ExtraUsageCharacters])
	}
}

func TestParseResponse_UsageMissingExtraInfo(t *testing.T) {
	audioHex := hex.EncodeToString([]byte("fake-audio"))
	body := []byte(`{
		"data": {"audio": "` + audioHex + `", "status": 2},
		"base_resp": {"status_code": 0, "status_msg": "success"}
	}`)

	resp, err := parseResponse(body)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Usage != nil {
		t.Fatalf("Usage: got %#v want nil", resp.Usage)
	}
}
