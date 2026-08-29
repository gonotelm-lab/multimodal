package dashscope

import (
	"bytes"
	"context"
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
	runType        = "dashscope"
	defaultBaseUrl = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	defaultModel   = "qwen3-tts-flash"
)

type Generator struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config, opts ...audios.ClientOption) (*Generator, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "dashscope api key is required")
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
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "marshal dashscope tts request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "build dashscope tts request failed")
	}

	httpReq.Header.Set("Authorization", "Bearer "+g.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "call dashscope tts failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "read dashscope tts response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, dashScopeHTTPErrorToErr(httpResp.StatusCode, respBody, "tts")
	}

	resp, err = parseResponse(respBody)
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnEnd(ctx, &audios.CallbackOutput{Response: resp})
	return resp, nil
}

func dashScopeLanguage(l schema.Language) string {
	switch l {
	case schema.LanguageChinese:
		return "Chinese"
	case schema.LanguageEnglish:
		return "English"
	default:
		return ""
	}
}

func (g *Generator) buildPayload(model string, req *schema.Request, callOpts *audios.CallOptions) apiRequest {
	inputFields := make(map[string]any)
	inputFields["text"] = req.Text
	inputFields["voice"] = req.Voice

	if lt := dashScopeLanguage(req.Language); lt != "" {
		inputFields["language_type"] = lt
	}
	if strings.TrimSpace(req.Instruction) != "" {
		inputFields[paramInstruction] = req.Instruction
	}

	if callOpts.Extra != nil {
		if v, ok := callOpts.Extra[optKeyFormat].(string); ok && v != "" {
			inputFields[paramFormat] = v
		}
		if v, ok := callOpts.Extra[optKeySampleRate].(int); ok && v > 0 {
			inputFields[paramSampleRate] = v
		}
	}

	return apiRequest{
		Model: model,
		Input: inputFields,
	}
}

type apiRequest struct {
	Model string         `json:"model"`
	Input map[string]any `json:"input"`
}

type apiResponse struct {
	Output    apiOutput `json:"output"`
	Usage     apiUsage  `json:"usage"`
	RequestID string    `json:"request_id"`
	Code      string    `json:"code,omitempty"`
	Message   string    `json:"message,omitempty"`
}

type apiOutput struct {
	Audio apiAudio `json:"audio"`
}

type apiAudio struct {
	URL       string `json:"url"`
	Data      string `json:"data"`
	ID        string `json:"id"`
	ExpiresAt int64  `json:"expires_at"`
}

type apiUsage struct {
	InputTokens         int               `json:"input_tokens"`
	OutputTokens        int               `json:"output_tokens"`
	TotalTokens         int               `json:"total_tokens"`
	Characters          int               `json:"characters"`
	InputTokensDetails  *usageTokenDetail `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *usageTokenDetail `json:"output_tokens_details,omitempty"`
}

type usageTokenDetail struct {
	TextTokens  int `json:"text_tokens"`
	AudioTokens int `json:"audio_tokens"`
}

func parseResponse(respBody []byte) (*schema.Response, error) {
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, errx.Wrap(err, errx.KindInternal, "decode dashscope tts response failed")
	}

	if apiResp.Code != "" {
		e := errx.Newf(errx.DashScopeCodeToKind(apiResp.Code),
			"dashscope tts error: code=%s message=%s", apiResp.Code, apiResp.Message)
		e.Raw = &apiResp
		return nil, e
	}

	if apiResp.Output.Audio.URL == "" {
		return nil, errx.New(errx.KindInternal, "dashscope tts response has no audio url")
	}

	extras := make(map[string]any)
	if apiResp.RequestID != "" {
		extras[ExtraRequestID] = apiResp.RequestID
	}
	if apiResp.Output.Audio.ExpiresAt > 0 {
		extras[ExtraExpiresAt] = apiResp.Output.Audio.ExpiresAt
	}
	if apiResp.Usage.InputTokens > 0 {
		extras[ExtraInputTokens] = apiResp.Usage.InputTokens
	}
	if apiResp.Usage.OutputTokens > 0 {
		extras[ExtraOutputTokens] = apiResp.Usage.OutputTokens
	}
	if apiResp.Usage.TotalTokens > 0 {
		extras[ExtraTotalTokens] = apiResp.Usage.TotalTokens
	}
	if apiResp.Usage.Characters > 0 {
		extras[ExtraCharacters] = apiResp.Usage.Characters
	}
	if apiResp.Usage.InputTokensDetails != nil {
		if apiResp.Usage.InputTokensDetails.TextTokens > 0 {
			extras[ExtraInputTextTokens] = apiResp.Usage.InputTokensDetails.TextTokens
		}
		if apiResp.Usage.InputTokensDetails.AudioTokens > 0 {
			extras[ExtraInputAudioTokens] = apiResp.Usage.InputTokensDetails.AudioTokens
		}
	}
	if apiResp.Usage.OutputTokensDetails != nil {
		if apiResp.Usage.OutputTokensDetails.AudioTokens > 0 {
			extras[ExtraOutputAudioTokens] = apiResp.Usage.OutputTokensDetails.AudioTokens
		}
		if apiResp.Usage.OutputTokensDetails.TextTokens > 0 {
			extras[ExtraOutputTextTokens] = apiResp.Usage.OutputTokensDetails.TextTokens
		}
	}

	var usage *schema.Usage
	if apiResp.Usage.Characters > 0 {
		chars := int64(apiResp.Usage.Characters)
		usage = &schema.Usage{Characters: &chars}
	}
	if apiResp.Usage.InputTokens > 0 || apiResp.Usage.OutputTokens > 0 || apiResp.Usage.TotalTokens > 0 {
		if usage == nil {
			usage = &schema.Usage{}
		}
		usage.TokenUsage = &schema.TokenUsage{
			InputTokens:  int64(apiResp.Usage.InputTokens),
			OutputTokens: int64(apiResp.Usage.OutputTokens),
			TotalTokens:  int64(apiResp.Usage.TotalTokens),
		}
	}

	return &schema.Response{
		ResponseFormat: schema.ResponseFormatURL,
		URL:            apiResp.Output.Audio.URL,
		AudioFormat:    "wav",
		Usage:          usage,
		Extras:         extras,
	}, nil
}

func dashScopeHTTPErrorToErr(status int, body []byte, api string) *errx.Error {
	var codeResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if uerr := json.Unmarshal(body, &codeResp); uerr == nil && codeResp.Code != "" {
		e := errx.Newf(errx.DashScopeCodeToKind(codeResp.Code),
			"dashscope %s error: code=%s message=%s", api, codeResp.Code, codeResp.Message)
		e.Raw = &codeResp
		return e
	}
	return errx.Newf(errx.FromHTTPStatus(status),
		"dashscope %s request failed: status=%d", api, status)
}
