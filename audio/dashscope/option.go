package dashscope

import (
	audios "github.com/gonotelm-lab/multimodal/audio"
)

const (
	optKeyFormat     = "dashscope_format"
	optKeySampleRate = "dashscope_sample_rate"

	paramInstruction = "instruction"
	paramFormat      = "format"
	paramSampleRate  = "sample_rate"
)

func WithFormat(format string) audios.Option {
	return audios.WithExtra(optKeyFormat, format)
}

func WithSampleRate(sampleRate int) audios.Option {
	return audios.WithExtra(optKeySampleRate, sampleRate)
}
