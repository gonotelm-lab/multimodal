package audio

import (
	"testing"

	"github.com/gonotelm-lab/multimodal/audio/schema"
	"github.com/gonotelm-lab/multimodal/callbacks"
)

func TestConvCallbackInput(t *testing.T) {
	in := &CallbackInput{Request: &schema.Request{Text: "x"}}
	if got := ConvCallbackInput(in); got != in {
		t.Fatalf("expected same pointer, got %v", got)
	}
	if got := ConvCallbackInput(callbacks.CallbackInput("not-typed")); got != nil {
		t.Fatalf("expected nil for wrong type, got %v", got)
	}
}

func TestConvCallbackOutput(t *testing.T) {
	out := &CallbackOutput{Response: &schema.Response{URL: "http://x"}}
	if got := ConvCallbackOutput(out); got != out {
		t.Fatalf("expected same pointer, got %v", got)
	}
	if got := ConvCallbackOutput(callbacks.CallbackOutput("not-typed")); got != nil {
		t.Fatalf("expected nil for wrong type, got %v", got)
	}
}
