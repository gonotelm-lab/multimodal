package image

import (
	"testing"

	"github.com/gonotelm-lab/multimodal/callbacks"
	"github.com/gonotelm-lab/multimodal/image/schema"
)

func TestConvCallbackInput(t *testing.T) {
	in := &CallbackInput{Request: &schema.Request{Prompt: "x"}}
	if got := ConvCallbackInput(in); got != in {
		t.Fatalf("expected same pointer, got %v", got)
	}
	if got := ConvCallbackInput(callbacks.CallbackInput("not-typed")); got != nil {
		t.Fatalf("expected nil for wrong type, got %v", got)
	}
}

func TestConvCallbackOutput(t *testing.T) {
	out := &CallbackOutput{Response: &schema.Response{ImageURL: "http://x"}}
	if got := ConvCallbackOutput(out); got != out {
		t.Fatalf("expected same pointer, got %v", got)
	}
	if got := ConvCallbackOutput(callbacks.CallbackOutput("not-typed")); got != nil {
		t.Fatalf("expected nil for wrong type, got %v", got)
	}
}
