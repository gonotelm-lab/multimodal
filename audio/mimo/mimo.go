package mimo

import (
	"bytes"
	"context"
	"encoding/base64"
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
	runType        = "mimo"
	defaultBaseUrl = "https://api.xiaomimimo.com/v1/chat/completions"
	defaultModel   = string(ModelTTS)

	defaultFormat = FormatWAV
)

type Generator struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config, opts ...audios.ClientOption) (*Generator, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "mimo api key is required")
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

	if strings.TrimSpace(req.Text) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "text is required")
	}

	model := g.cfg.Model
	if req.Model != "" {
		model = req.Model
	}

	format := defaultFormat
	if callOpts.Extra != nil {
		if v, ok := callOpts.Extra[extraKeyFormat].(Format); ok && v != "" {
			format = v
		}
	}

	payload := g.buildPayload(model, string(format), req, callOpts)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "marshal mimo tts request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "build mimo tts request failed")
	}
	httpReq.Header.Set("api-key", g.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "call mimo tts failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "read mimo tts response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, openAIHTTPErrorToErr(httpResp.StatusCode, respBody, "mimo", "tts")
	}

	resp, err = parseResponse(respBody, string(format))
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnEnd(ctx, &audios.CallbackOutput{Response: resp})
	return resp, nil
}

func (g *Generator) buildPayload(model, format string, req *schema.Request, callOpts *audios.CallOptions) apiRequest {
	audioField := map[string]any{
		paramFormat: format,
	}
	if strings.TrimSpace(req.Voice) != "" {
		audioField[paramVoice] = req.Voice
	}
	if callOpts.Extra != nil {
		if v, ok := callOpts.Extra[extraKeyOptimizeTextPreview].(bool); ok && v {
			audioField[paramOptimizeTextPreview] = true
		}
	}

	messages := make([]apiMessage, 0, 2)
	if strings.TrimSpace(req.Instruction) != "" {
		messages = append(messages, apiMessage{
			Role:    "user",
			Content: req.Instruction,
		})
	}
	messages = append(messages, apiMessage{
		Role:    "assistant",
		Content: req.Text,
	})

	return apiRequest{
		Model:    model,
		Messages: messages,
		Audio:    audioField,
	}
}

type apiRequest struct {
	Model    string         `json:"model"`
	Messages []apiMessage   `json:"messages"`
	Audio    map[string]any `json:"audio"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	ID      string      `json:"id,omitempty"`
	Choices []apiChoice `json:"choices"`
	Error   *apiError   `json:"error,omitempty"`
	Usage   *apiUsage   `json:"usage,omitempty"`
}

type apiChoice struct {
	Index        int           `json:"index"`
	Message      apiMessageOut `json:"message"`
	FinishReason string        `json:"finish_reason,omitempty"`
}

type apiMessageOut struct {
	Role    string       `json:"role"`
	Content string       `json:"content,omitempty"`
	Audio   *apiAudioOut `json:"audio,omitempty"`
}

type apiAudioOut struct {
	Data       string `json:"data"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
	ID         string `json:"id,omitempty"`
	Transcript string `json:"transcript,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type apiUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

func parseResponse(respBody []byte, format string) (*schema.Response, error) {
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, errx.Wrap(err, errx.KindInternal, "decode mimo tts response failed")
	}

	if apiResp.Error != nil {
		e := errx.Newf(errx.OpenAIErrorTypeToKind(apiResp.Error.Type),
			"mimo tts error: type=%s code=%s message=%s",
			apiResp.Error.Type, apiResp.Error.Code, apiResp.Error.Message)
		e.Raw = apiResp.Error
		return nil, e
	}

	if len(apiResp.Choices) == 0 {
		return nil, errx.New(errx.KindInternal, "mimo tts response has no choices")
	}

	choice := apiResp.Choices[0]
	if choice.Message.Audio == nil || strings.TrimSpace(choice.Message.Audio.Data) == "" {
		return nil, errx.New(errx.KindInternal, "mimo tts response has no audio data")
	}

	audioBytes, err := base64.StdEncoding.DecodeString(choice.Message.Audio.Data)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInternal, "decode mimo tts audio base64 failed")
	}

	extras := make(map[string]any)
	if apiResp.ID != "" {
		extras["id"] = apiResp.ID
	}
	if choice.FinishReason != "" {
		extras["finish_reason"] = choice.FinishReason
	}
	if choice.Message.Audio.ID != "" {
		extras["audio_id"] = choice.Message.Audio.ID
	}
	if choice.Message.Audio.ExpiresAt > 0 {
		extras["expires_at"] = choice.Message.Audio.ExpiresAt
	}
	if choice.Message.Audio.Transcript != "" {
		extras["transcript"] = choice.Message.Audio.Transcript
	}
	if apiResp.Usage != nil {
		if apiResp.Usage.PromptTokens > 0 {
			extras["prompt_tokens"] = apiResp.Usage.PromptTokens
		}
		if apiResp.Usage.CompletionTokens > 0 {
			extras["completion_tokens"] = apiResp.Usage.CompletionTokens
		}
		if apiResp.Usage.TotalTokens > 0 {
			extras["total_tokens"] = apiResp.Usage.TotalTokens
		}
	}

	return &schema.Response{
		ResponseFormat: schema.ResponseFormatBytes,
		Reader:         io.NopCloser(bytes.NewReader(audioBytes)),
		AudioFormat:    format,
		Extras:         extras,
	}, nil
}

func openAIHTTPErrorToErr(status int, body []byte, provider, api string) *errx.Error {
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if uerr := json.Unmarshal(body, &errResp); uerr == nil && errResp.Error.Type != "" {
		e := errx.Newf(errx.OpenAIErrorTypeToKind(errResp.Error.Type),
			"%s %s error: type=%s code=%s message=%s",
			provider, api, errResp.Error.Type, errResp.Error.Code, errResp.Error.Message)
		e.Raw = &errResp.Error
		return e
	}
	return errx.Newf(errx.FromHTTPStatus(status),
		"%s %s request failed: status=%d", provider, api, status)
}
