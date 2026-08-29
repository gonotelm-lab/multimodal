package minimax

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gonotelm-lab/multimodal/callbacks"
	"github.com/gonotelm-lab/multimodal/error"

	audios "github.com/gonotelm-lab/multimodal/audio"
	"github.com/gonotelm-lab/multimodal/audio/schema"
)

const (
	runType        = "minimax"
	defaultBaseUrl = "https://api.minimaxi.com/v1/t2a_v2"
	defaultModel   = ModelSpeech28HD
)

type Generator struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config, opts ...audios.ClientOption) (*Generator, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "minimax api key is required")
	}
	if strings.TrimSpace(cfg.BaseUrl) == "" {
		cfg.BaseUrl = defaultBaseUrl
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultModel
	}

	co := audios.BuildClientOptions(cfg.Timeout, opts...)
	return &Generator{
		cfg:        cfg,
		httpClient: co.HTTPClient,
	}, nil
}

func (g *Generator) Generate(ctx context.Context, req *schema.Request, opts ...audios.Option) (resp *schema.Response, err error) {
	callOpts := audios.BuildCallOptions(opts...)

	// 模型优先级：Request > Config；回写 req 供 callback / recorder 使用
	model := g.cfg.Model
	if req != nil && req.Model != "" {
		model = req.Model
	}
	if req != nil {
		req.Model = model
	}

	ctx = callbacks.EnsureRunInfo(ctx, runType, callbacks.ComponentAudio)
	ctx = callbacks.OnStart(ctx, &audios.CallbackInput{
		Request:     req,
		CallOptions: callOpts,
	})
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()

	if req == nil || strings.TrimSpace(req.Text) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "text is required")
	}
	if strings.TrimSpace(req.Voice) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "voice is required")
	}

	payload := g.buildPayload(model, req, callOpts)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "marshal minimax tts request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "build minimax tts request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "call minimax tts failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "read minimax tts response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, miniMaxHTTPErrorToErr(httpResp.StatusCode, respBody)
	}

	resp, err = parseResponse(respBody)
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnEnd(ctx, &audios.CallbackOutput{Response: resp})
	return resp, nil
}

func minimaxLanguage(l schema.Language) LanguageBoost {
	switch l {
	case schema.LanguageChinese:
		return LanguageBoostChinese
	case schema.LanguageEnglish:
		return LanguageBoostEnglish
	default:
		return ""
	}
}

func (g *Generator) buildPayload(model string, req *schema.Request, callOpts *audios.CallOptions) apiRequest {
	payload := apiRequest{
		Model:        model,
		Text:         req.Text,
		Stream:       false,
		VoiceSetting: voiceSetting{VoiceID: req.Voice},
	}

	if lb := minimaxLanguage(req.Language); lb != "" {
		payload.LanguageBoost = lb
	}

	if callOpts.Extra != nil {
		if v, ok := callOpts.Extra[extraKeySpeed].(float64); ok {
			payload.VoiceSetting.Speed = v
		}
		if v, ok := callOpts.Extra[extraKeyVolume].(float64); ok {
			payload.VoiceSetting.Vol = v
		}
		if v, ok := callOpts.Extra[extraKeyPitch].(int); ok {
			payload.VoiceSetting.Pitch = v
		}
		if v, ok := callOpts.Extra[extraKeyEmotion].(Emotion); ok && v != "" {
			payload.VoiceSetting.Emotion = v
		}
		if v, ok := callOpts.Extra[extraKeyLanguageBoost].(LanguageBoost); ok && v != "" {
			payload.LanguageBoost = v
		}
		if v, ok := callOpts.Extra[extraKeyOutputFormat].(OutputFormat); ok && v != "" {
			payload.OutputFormat = v
		}
		if v, ok := callOpts.Extra[extraKeyPronunciationDict].([]string); ok && len(v) > 0 {
			payload.PronunciationDict = &pronunciationDict{Tone: v}
		}
		if v, ok := callOpts.Extra[extraKeySubtitleEnable].(bool); ok {
			payload.SubtitleEnable = v
		}
		if v, ok := callOpts.Extra[extraKeySubtitleType].(SubtitleType); ok && v != "" {
			payload.SubtitleType = v
		}
		if v, ok := callOpts.Extra[extraKeyTextNormalization].(bool); ok {
			payload.VoiceSetting.TextNormalization = v
		}
		if v, ok := callOpts.Extra[extraKeyLatexRead].(bool); ok {
			payload.VoiceSetting.LatexRead = v
		}
		if v, ok := callOpts.Extra[extraKeyAigcWatermark].(bool); ok {
			payload.AigcWatermark = v
		}

		audioSet := audioSetting{}
		hasAudio := false
		if v, ok := callOpts.Extra[extraKeySampleRate].(SampleRate); ok && v > 0 {
			audioSet.SampleRate = v
			hasAudio = true
		}
		if v, ok := callOpts.Extra[extraKeyBitrate].(Bitrate); ok && v > 0 {
			audioSet.Bitrate = v
			hasAudio = true
		}
		if v, ok := callOpts.Extra[extraKeyChannel].(Channel); ok && v > 0 {
			audioSet.Channel = v
			hasAudio = true
		}
		if v, ok := callOpts.Extra[extraKeyAudioFormat].(AudioFormat); ok && v != "" {
			audioSet.Format = v
			hasAudio = true
		}
		if hasAudio {
			payload.AudioSetting = &audioSet
		}
	}

	return payload
}

type apiRequest struct {
	Model             string             `json:"model"`
	Text              string             `json:"text"`
	Stream            bool               `json:"stream"`
	VoiceSetting      voiceSetting       `json:"voice_setting"`
	AudioSetting      *audioSetting      `json:"audio_setting,omitempty"`
	LanguageBoost     LanguageBoost      `json:"language_boost,omitempty"`
	PronunciationDict *pronunciationDict `json:"pronunciation_dict,omitempty"`
	OutputFormat      OutputFormat       `json:"output_format,omitempty"`
	SubtitleEnable    bool               `json:"subtitle_enable,omitempty"`
	SubtitleType      SubtitleType       `json:"subtitle_type,omitempty"`
	AigcWatermark     bool               `json:"aigc_watermark,omitempty"`
}

type voiceSetting struct {
	VoiceID           string  `json:"voice_id"`
	Speed             float64 `json:"speed,omitempty"`
	Vol               float64 `json:"vol,omitempty"`
	Pitch             int     `json:"pitch,omitempty"`
	Emotion           Emotion `json:"emotion,omitempty"`
	TextNormalization bool    `json:"text_normalization,omitempty"`
	LatexRead         bool    `json:"latex_read,omitempty"`
}

type audioSetting struct {
	SampleRate SampleRate  `json:"sample_rate,omitempty"`
	Bitrate    Bitrate     `json:"bitrate,omitempty"`
	Format     AudioFormat `json:"format,omitempty"`
	Channel    Channel     `json:"channel,omitempty"`
}

type pronunciationDict struct {
	Tone []string `json:"tone,omitempty"`
}

type apiResponse struct {
	Data      *responseData `json:"data"`
	TraceID   string        `json:"trace_id"`
	ExtraInfo *extraInfo    `json:"extra_info"`
	BaseResp  baseResp      `json:"base_resp"`
}

type responseData struct {
	Audio        string `json:"audio"`
	SubtitleFile string `json:"subtitle_file"`
	Status       int    `json:"status"`
}

type extraInfo struct {
	AudioLength             int64   `json:"audio_length"`
	AudioSampleRate         int64   `json:"audio_sample_rate"`
	AudioSize               int64   `json:"audio_size"`
	Bitrate                 int64   `json:"bitrate"`
	AudioFormat             string  `json:"audio_format"`
	AudioChannel            int64   `json:"audio_channel"`
	UsageCharacters         int64   `json:"usage_characters"`
	WordCount               int64   `json:"word_count"`
	InvisibleCharacterRatio float64 `json:"invisible_character_ratio"`
}

type baseResp struct {
	StatusCode int64  `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func parseResponse(respBody []byte) (*schema.Response, error) {
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, errx.Wrap(err, errx.KindInternal, "decode minimax tts response failed")
	}

	if apiResp.BaseResp.StatusCode != 0 {
		e := errx.Newf(errx.MiniMaxCodeToKind(apiResp.BaseResp.StatusCode),
			"minimax tts error: status_code=%d status_msg=%s",
			apiResp.BaseResp.StatusCode, apiResp.BaseResp.StatusMsg)
		e.Raw = &apiResp.BaseResp
		return nil, e
	}

	if apiResp.Data == nil || apiResp.Data.Audio == "" {
		return nil, errx.New(errx.KindInternal, "minimax tts response has no audio data")
	}

	extras := make(map[string]any)
	if apiResp.TraceID != "" {
		extras["trace_id"] = apiResp.TraceID
	}
	if apiResp.ExtraInfo != nil {
		if apiResp.ExtraInfo.AudioLength > 0 {
			extras["audio_length"] = apiResp.ExtraInfo.AudioLength
		}
		if apiResp.ExtraInfo.AudioSize > 0 {
			extras["audio_size"] = apiResp.ExtraInfo.AudioSize
		}
		if apiResp.ExtraInfo.UsageCharacters > 0 {
			extras["usage_characters"] = apiResp.ExtraInfo.UsageCharacters
		}
		if apiResp.ExtraInfo.AudioSampleRate > 0 {
			extras["audio_sample_rate"] = apiResp.ExtraInfo.AudioSampleRate
		}
		if apiResp.ExtraInfo.Bitrate > 0 {
			extras["bitrate"] = apiResp.ExtraInfo.Bitrate
		}
		if apiResp.ExtraInfo.AudioChannel > 0 {
			extras["audio_channel"] = apiResp.ExtraInfo.AudioChannel
		}
		if apiResp.ExtraInfo.WordCount > 0 {
			extras["word_count"] = apiResp.ExtraInfo.WordCount
		}
		if apiResp.ExtraInfo.InvisibleCharacterRatio > 0 {
			extras["invisible_character_ratio"] = apiResp.ExtraInfo.InvisibleCharacterRatio
		}
	}
	if apiResp.Data.SubtitleFile != "" {
		extras["subtitle_file"] = apiResp.Data.SubtitleFile
	}

	audioFormat := ""
	if apiResp.ExtraInfo != nil {
		audioFormat = apiResp.ExtraInfo.AudioFormat
	}

	// output_format=hex → data.audio 是 hex 字符串；output_format=url → data.audio 是 URL
	if strings.HasPrefix(apiResp.Data.Audio, "http://") || strings.HasPrefix(apiResp.Data.Audio, "https://") {
		return &schema.Response{
			ResponseFormat: schema.ResponseFormatURL,
			URL:            apiResp.Data.Audio,
			AudioFormat:    audioFormat,
			Extras:         extras,
		}, nil
	}

	decoded, err := hex.DecodeString(apiResp.Data.Audio)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInternal, "decode minimax audio hex failed")
	}
	return &schema.Response{
		ResponseFormat: schema.ResponseFormatBytes,
		Reader:         io.NopCloser(bytes.NewReader(decoded)),
		AudioFormat:    audioFormat,
		Extras:         extras,
	}, nil
}

func miniMaxHTTPErrorToErr(status int, body []byte) *errx.Error {
	var errResp struct {
		BaseResp struct {
			StatusCode int64  `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if uerr := json.Unmarshal(body, &errResp); uerr == nil && errResp.BaseResp.StatusCode != 0 {
		e := errx.Newf(errx.MiniMaxCodeToKind(errResp.BaseResp.StatusCode),
			"minimax tts error: status_code=%d status_msg=%s",
			errResp.BaseResp.StatusCode, errResp.BaseResp.StatusMsg)
		e.Raw = &errResp.BaseResp
		return e
	}
	return errx.Newf(errx.FromHTTPStatus(status),
		"minimax tts request failed: status=%d", status)
}
