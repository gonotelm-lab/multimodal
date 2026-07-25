package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gonotelm-lab/multimodal/error"

	audios "github.com/gonotelm-lab/multimodal/audio"
	"github.com/gonotelm-lab/multimodal/audio/schema"
)

const (
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

func (g *Generator) Generate(ctx context.Context, req *schema.Request, opts ...audios.Option) (*schema.Response, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "text is required")
	}
	if strings.TrimSpace(req.Voice) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "voice is required")
	}

	callOpts := audios.BuildCallOptions(opts...)

	model := g.cfg.Model
	if req.Model != "" {
		model = req.Model
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

	return parseResponse(respBody)
}

func (g *Generator) buildPayload(model string, req *schema.Request, callOpts *audios.CallOptions) apiRequest {
	inputFields := make(map[string]any)
	inputFields["text"] = req.Text
	inputFields["voice"] = req.Voice

	if strings.TrimSpace(req.Language) != "" {
		inputFields["language_type"] = req.Language
	}
	if strings.TrimSpace(req.Instruction) != "" {
		inputFields[paramInstruction] = req.Instruction
	}

	if callOpts.Extra != nil {
		if v, ok := callOpts.Extra[extraKeyFormat].(string); ok && v != "" {
			inputFields[paramFormat] = v
		}
		if v, ok := callOpts.Extra[extraKeySampleRate].(int); ok && v > 0 {
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
		extras["request_id"] = apiResp.RequestID
	}
	if apiResp.Output.Audio.ExpiresAt > 0 {
		extras["expires_at"] = apiResp.Output.Audio.ExpiresAt
	}
	if apiResp.Usage.InputTokens > 0 {
		extras["input_tokens"] = apiResp.Usage.InputTokens
	}
	if apiResp.Usage.OutputTokens > 0 {
		extras["output_tokens"] = apiResp.Usage.OutputTokens
	}
	if apiResp.Usage.TotalTokens > 0 {
		extras["total_tokens"] = apiResp.Usage.TotalTokens
	}
	if apiResp.Usage.Characters > 0 {
		extras["characters"] = apiResp.Usage.Characters
	}

	return &schema.Response{
		ResponseFormat: schema.ResponseFormatURL,
		URL:            apiResp.Output.Audio.URL,
		AudioFormat:    "wav",
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
