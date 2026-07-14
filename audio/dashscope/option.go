package dashscope

import (
	audios "github.com/gonotelm-lab/multimodal/audio"
)

const (
	extraKeyFormat     = "dashscope_format"
	extraKeySampleRate = "dashscope_sample_rate"

	paramInstruction = "instruction"
	paramFormat      = "format"
	paramSampleRate  = "sample_rate"
)

func WithFormat(format string) audios.Option {
	return audios.WithExtra(extraKeyFormat, format)
}

func WithSampleRate(sampleRate int) audios.Option {
	return audios.WithExtra(extraKeySampleRate, sampleRate)
}
