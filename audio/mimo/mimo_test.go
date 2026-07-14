package mimo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/gonotelm-lab/multimodal/audio/schema"
	"github.com/gonotelm-lab/multimodal/audio/util"
)

func getAPIKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("GONOTELM_MIMO_APIKEY")
	if key == "" {
		t.Skip("GONOTELM_MIMO_APIKEY not set, skipping integration test")
	}
	return key
}

func TestGenerate_Basic(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Text:  "Hey boss — guess what, I actually passed the exam with distinction!",
		Voice: string(VoiceChloe),
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.Reader == nil {
		t.Fatal("expected non-nil audio reader")
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("Response: %s\n", string(output))
	reader, err := util.ResolveResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	f, err := os.OpenFile("/tmp/mimo.wav", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Fatal(err)
	}

	_, err = io.Copy(f, reader)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	reader.Close()
}

func TestGenerate_WithInstruction(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Text:        "今天天气真好，适合出去散步。",
		Voice:       string(VoiceBingTang),
		Instruction: "用温柔慵懒的语调，语速稍慢。",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.Reader == nil {
		t.Fatal("expected non-nil audio reader")
	}

	fmt.Printf("AudioFormat: %s\n", resp.AudioFormat)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_VoiceDesign(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t), Model: string(ModelVoiceDesign)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(
		t.Context(),
		&schema.Request{
			Instruction: "Give me a young male tone.",
			Text:        "Yes, I had a sandwich.",
		},
		WithOptimizeTextPreview(true),
	)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.Reader == nil {
		t.Fatal("expected non-nil audio reader")
	}

	fmt.Printf("AudioFormat: %s\n", resp.AudioFormat)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_VoiceClone(t *testing.T) {
	apiKey := getAPIKey(t)
	voicePath := os.Getenv("GONOTELM_MIMO_VOICE_SAMPLE")
	if voicePath == "" {
		t.Skip("GONOTELM_MIMO_VOICE_SAMPLE not set, skipping voiceclone test")
	}

	voiceBytes, err := os.ReadFile(voicePath)
	if err != nil {
		t.Fatalf("read voice sample failed: %v", err)
	}
	voiceBase64 := "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(voiceBytes)

	gen, err := New(Config{APIKey: apiKey, Model: string(ModelVoiceClone)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(t.Context(), &schema.Request{
		Text:  "Yes, I had a sandwich.",
		Voice: voiceBase64,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.Reader == nil {
		t.Fatal("expected non-nil audio reader")
	}

	fmt.Printf("AudioFormat: %s\n", resp.AudioFormat)
	fmt.Printf("Extras: %+v\n", resp.Extras)
}

func TestGenerate_WithFormat(t *testing.T) {
	gen, err := New(Config{APIKey: getAPIKey(t)})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	resp, err := gen.Generate(
		t.Context(),
		&schema.Request{
			Text:  "ping",
			Voice: string(VoiceMia),
		},
		WithFormat(FormatPCM16),
	)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.AudioFormat != "pcm16" {
		t.Fatalf("expected AudioFormat=pcm16, got %s", resp.AudioFormat)
	}
}

func TestGenerate_InvalidAPIKey(t *testing.T) {
	gen, err := New(Config{APIKey: "invalid-key"})
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	_, err = gen.Generate(t.Context(), &schema.Request{
		Text:  "test",
		Voice: string(VoiceChloe),
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
		Voice: string(VoiceChloe),
	})
	if err == nil {
		t.Fatal("expected error with empty text")
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
