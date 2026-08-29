package mimo

import (
	audios "github.com/gonotelm-lab/multimodal/audio"
)

const (
	optKeyFormat              = "mimo_format"
	optKeyOptimizeTextPreview = "mimo_optimize_text_preview"

	paramFormat              = "format"
	paramVoice               = "voice"
	paramOptimizeTextPreview = "optimize_text_preview"
)

// Format 表示 MiMo TTS 合成音频的编码格式。
type Format string

const (
	// FormatWAV 为 WAV 封装格式，适用于非流式调用。
	FormatWAV Format = "wav"
	// FormatPCM16 为 24kHz PCM16LE 单声道格式，适用于流式调用，便于拼接成完整音频。
	FormatPCM16 Format = "pcm16"
)

// Voice 表示 MiMo TTS 的预置音色。
//
// 仅适用于 mimo-v2.5-tts 模型；voicedesign 模型由文本描述生成音色，
// voiceclone 模型则传入形如 "data:audio/mpeg;base64,..." 的音频样本。
type Voice string

const (
	VoiceMiMoDefault Voice = "mimo_default"
	VoiceBingTang    Voice = "冰糖"
	VoiceMoLi        Voice = "茉莉"
	VoiceSuDa        Voice = "苏打"
	VoiceBaiHua      Voice = "白桦"
	VoiceMia         Voice = "Mia"
	VoiceChloe       Voice = "Chloe"
	VoiceMilo        Voice = "Milo"
	VoiceDean        Voice = "Dean"
)

// Model 表示 MiMo TTS 系列的模型 ID。
type Model string

const (
	// ModelTTS 使用预置精品音色进行语音合成，支持唱歌模式，不支持音色设计与复刻。
	ModelTTS Model = "mimo-v2.5-tts"
	// ModelVoiceDesign 通过文本描述定制音色，不支持唱歌模式、预置音色与音色复刻。
	ModelVoiceDesign Model = "mimo-v2.5-tts-voicedesign"
	// ModelVoiceClone 基于音频样本复刻任意音色，不支持唱歌模式、预置音色与音色设计。
	ModelVoiceClone Model = "mimo-v2.5-tts-voiceclone"
)

// WithFormat 设置合成音频的格式，不传时默认为 FormatWAV。
func WithFormat(format Format) audios.Option {
	return audios.WithExtra(optKeyFormat, format)
}

// WithOptimizeTextPreview 用于 ModelVoiceDesign 模型，
// 设为 true 时由模型对目标播报文本进行智能润色，可省略 assistant 消息。
func WithOptimizeTextPreview(enable bool) audios.Option {
	return audios.WithExtra(optKeyOptimizeTextPreview, enable)
}
