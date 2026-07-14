package minimax

import (
	audios "github.com/gonotelm-lab/multimodal/audio"
)

const (
	extraKeySpeed             = "minimax_speed"
	extraKeyVolume            = "minimax_volume"
	extraKeyPitch             = "minimax_pitch"
	extraKeyEmotion           = "minimax_emotion"
	extraKeyAudioFormat       = "minimax_audio_format"
	extraKeySampleRate        = "minimax_sample_rate"
	extraKeyBitrate           = "minimax_bitrate"
	extraKeyChannel           = "minimax_channel"
	extraKeyLanguageBoost     = "minimax_language_boost"
	extraKeyOutputFormat      = "minimax_output_format"
	extraKeyPronunciationDict = "minimax_pronunciation_dict"
	extraKeySubtitleEnable    = "minimax_subtitle_enable"
	extraKeySubtitleType      = "minimax_subtitle_type"
	extraKeyTextNormalization = "minimax_text_normalization"
	extraKeyLatexRead         = "minimax_latex_read"
	extraKeyAigcWatermark     = "minimax_aigc_watermark"
)

const (
	ModelSpeech28HD    = "speech-2.8-hd"
	ModelSpeech28Turbo = "speech-2.8-turbo"
	ModelSpeech26HD    = "speech-2.6-hd"
	ModelSpeech26Turbo = "speech-2.6-turbo"
	ModelSpeech02HD    = "speech-02-hd"
	ModelSpeech02Turbo = "speech-02-turbo"
	ModelSpeech01HD    = "speech-01-hd"
	ModelSpeech01Turbo = "speech-01-turbo"
)

type Emotion string

const (
	EmotionHappy     Emotion = "happy"
	EmotionSad       Emotion = "sad"
	EmotionAngry     Emotion = "angry"
	EmotionFearful   Emotion = "fearful"
	EmotionDisgusted Emotion = "disgusted"
	EmotionSurprised Emotion = "surprised"
	EmotionCalm      Emotion = "calm"
	EmotionFluent    Emotion = "fluent"
	EmotionWhisper   Emotion = "whisper"
)

type AudioFormat string

const (
	AudioFormatMP3     AudioFormat = "mp3"
	AudioFormatPCM     AudioFormat = "pcm"
	AudioFormatFLAC    AudioFormat = "flac"
	AudioFormatWAV     AudioFormat = "wav"
	AudioFormatPCMURaw AudioFormat = "pcmu_raw"
	AudioFormatPCMUWav AudioFormat = "pcmu_wav"
	AudioFormatOpus    AudioFormat = "opus"
)

type SampleRate int

const (
	SampleRate8000  SampleRate = 8000
	SampleRate16000 SampleRate = 16000
	SampleRate22050 SampleRate = 22050
	SampleRate24000 SampleRate = 24000
	SampleRate32000 SampleRate = 32000
	SampleRate44100 SampleRate = 44100
)

type Bitrate int

const (
	Bitrate32000  Bitrate = 32000
	Bitrate64000  Bitrate = 64000
	Bitrate128000 Bitrate = 128000
	Bitrate256000 Bitrate = 256000
)

type Channel int

const (
	ChannelMono   Channel = 1
	ChannelStereo Channel = 2
)

type OutputFormat string

const (
	OutputFormatHex OutputFormat = "hex"
	OutputFormatURL OutputFormat = "url"
)

type SubtitleType string

const (
	SubtitleTypeSentence      SubtitleType = "sentence"
	SubtitleTypeWord          SubtitleType = "word"
	SubtitleTypeWordStreaming SubtitleType = "word_streaming"
)

type LanguageBoost string

const (
	LanguageBoostAuto       LanguageBoost = "auto"
	LanguageBoostChinese    LanguageBoost = "Chinese"
	LanguageBoostChineseYue LanguageBoost = "Chinese,Yue"
	LanguageBoostEnglish    LanguageBoost = "English"
	LanguageBoostArabic     LanguageBoost = "Arabic"
	LanguageBoostRussian    LanguageBoost = "Russian"
	LanguageBoostSpanish    LanguageBoost = "Spanish"
	LanguageBoostFrench     LanguageBoost = "French"
	LanguageBoostPortuguese LanguageBoost = "Portuguese"
	LanguageBoostGerman     LanguageBoost = "German"
	LanguageBoostTurkish    LanguageBoost = "Turkish"
	LanguageBoostDutch      LanguageBoost = "Dutch"
	LanguageBoostUkrainian  LanguageBoost = "Ukrainian"
	LanguageBoostVietnamese LanguageBoost = "Vietnamese"
	LanguageBoostIndonesian LanguageBoost = "Indonesian"
	LanguageBoostJapanese   LanguageBoost = "Japanese"
	LanguageBoostItalian    LanguageBoost = "Italian"
	LanguageBoostKorean     LanguageBoost = "Korean"
	LanguageBoostThai       LanguageBoost = "Thai"
	LanguageBoostPolish     LanguageBoost = "Polish"
	LanguageBoostRomanian   LanguageBoost = "Romanian"
	LanguageBoostGreek      LanguageBoost = "Greek"
	LanguageBoostCzech      LanguageBoost = "Czech"
	LanguageBoostFinnish    LanguageBoost = "Finnish"
	LanguageBoostHindi      LanguageBoost = "Hindi"
	LanguageBoostBulgarian  LanguageBoost = "Bulgarian"
	LanguageBoostDanish     LanguageBoost = "Danish"
	LanguageBoostHebrew     LanguageBoost = "Hebrew"
	LanguageBoostMalay      LanguageBoost = "Malay"
	LanguageBoostPersian    LanguageBoost = "Persian"
	LanguageBoostSlovak     LanguageBoost = "Slovak"
	LanguageBoostSwedish    LanguageBoost = "Swedish"
	LanguageBoostCroatian   LanguageBoost = "Croatian"
	LanguageBoostFilipino   LanguageBoost = "Filipino"
	LanguageBoostHungarian  LanguageBoost = "Hungarian"
	LanguageBoostNorwegian  LanguageBoost = "Norwegian"
	LanguageBoostSlovenian  LanguageBoost = "Slovenian"
	LanguageBoostCatalan    LanguageBoost = "Catalan"
	LanguageBoostNynorsk    LanguageBoost = "Nynorsk"
	LanguageBoostTamil      LanguageBoost = "Tamil"
	LanguageBoostAfrikaans  LanguageBoost = "Afrikaans"
)

func WithSpeed(v float64) audios.Option {
	return audios.WithExtra(extraKeySpeed, v)
}

func WithVolume(v float64) audios.Option {
	return audios.WithExtra(extraKeyVolume, v)
}

func WithPitch(v int) audios.Option {
	return audios.WithExtra(extraKeyPitch, v)
}

func WithEmotion(v Emotion) audios.Option {
	return audios.WithExtra(extraKeyEmotion, v)
}

func WithAudioFormat(v AudioFormat) audios.Option {
	return audios.WithExtra(extraKeyAudioFormat, v)
}

func WithSampleRate(v SampleRate) audios.Option {
	return audios.WithExtra(extraKeySampleRate, v)
}

func WithBitrate(v Bitrate) audios.Option {
	return audios.WithExtra(extraKeyBitrate, v)
}

func WithChannel(v Channel) audios.Option {
	return audios.WithExtra(extraKeyChannel, v)
}

func WithLanguageBoost(v LanguageBoost) audios.Option {
	return audios.WithExtra(extraKeyLanguageBoost, v)
}

func WithOutputFormat(v OutputFormat) audios.Option {
	return audios.WithExtra(extraKeyOutputFormat, v)
}

func WithPronunciationDict(tone []string) audios.Option {
	return audios.WithExtra(extraKeyPronunciationDict, tone)
}

func WithSubtitleEnable(v bool) audios.Option {
	return audios.WithExtra(extraKeySubtitleEnable, v)
}

func WithSubtitleType(v SubtitleType) audios.Option {
	return audios.WithExtra(extraKeySubtitleType, v)
}

func WithTextNormalization(v bool) audios.Option {
	return audios.WithExtra(extraKeyTextNormalization, v)
}

func WithLatexRead(v bool) audios.Option {
	return audios.WithExtra(extraKeyLatexRead, v)
}

func WithAigcWatermark(v bool) audios.Option {
	return audios.WithExtra(extraKeyAigcWatermark, v)
}
