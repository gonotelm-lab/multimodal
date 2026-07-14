package dashscope

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/gonotelm-lab/multimodal/audio/schema"
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
		Language: "English",
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
