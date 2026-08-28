package audio

import (
	"github.com/gonotelm-lab/multimodal/audio/schema"
	"github.com/gonotelm-lab/multimodal/callbacks"
)

type CallbackInput struct {
	Request     *schema.Request
	CallOptions *CallOptions
}

type CallbackOutput struct {
	Response *schema.Response
}

func ConvCallbackInput(in callbacks.CallbackInput) *CallbackInput {
	ci, _ := in.(*CallbackInput)
	return ci
}

func ConvCallbackOutput(out callbacks.CallbackOutput) *CallbackOutput {
	co, _ := out.(*CallbackOutput)
	return co
}
