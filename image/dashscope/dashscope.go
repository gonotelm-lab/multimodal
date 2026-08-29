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

	images "github.com/gonotelm-lab/multimodal/image"
	"github.com/gonotelm-lab/multimodal/image/schema"
)

// https://help.aliyun.com/zh/model-studio/qwen-image-api

const (
	runType        = "dashscope"
	defaultBaseUrl = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	defaultModel   = "qwen-image-2.0-pro"
)

type Generator struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config, opts ...images.ClientOption) (*Generator, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "dashscope api key is required")
	}
	if strings.TrimSpace(cfg.BaseUrl) == "" {
		cfg.BaseUrl = defaultBaseUrl
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultModel
	}

	co := images.BuildClientOptions(cfg.Timeout, opts...)
	return &Generator{
		cfg:        cfg,
		httpClient: co.HTTPClient,
	}, nil
}

func (g *Generator) Generate(ctx context.Context, req *schema.Request, opts ...images.Option) (resp *schema.Response, err error) {
	callOpts := images.BuildCallOptions(opts...)

	// 模型优先级：Request > Config；回写 req 供 callback / recorder 使用
	model := g.cfg.Model
	if req != nil && req.Model != "" {
		model = req.Model
	}
	if req != nil {
		req.Model = model
	}

	ctx = callbacks.EnsureRunInfo(ctx, runType, callbacks.ComponentImage)
	ctx = callbacks.OnStart(ctx, &images.CallbackInput{
		Request:     req,
		CallOptions: callOpts,
	})
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()

	if req == nil || strings.TrimSpace(req.Prompt) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "prompt is required")
	}

	// 构建 parameters
	parameters := buildParameters(req, callOpts)

	// 构建请求体
	payload := apiRequest{
		Model: model,
		Input: input{
			Messages: []message{
				{
					Role: "user",
					Content: []contentItem{
						{Text: req.Prompt},
					},
				},
			},
		},
		Parameters: parameters,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "marshal dashscope text2image request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "build dashscope text2image request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "call dashscope text2image failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "read dashscope text2image response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, dashScopeHTTPErrorToErr(httpResp.StatusCode, respBody, "text2image")
	}

	resp, err = parseResponse(respBody)
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnEnd(ctx, &images.CallbackOutput{Response: resp})
	return resp, nil
}

func buildParameters(req *schema.Request, callOpts *images.CallOptions) map[string]any {
	parameters := make(map[string]any)

	// 默认参数
	parameters[paramPromptExtend] = true
	parameters[paramWatermark] = false
	parameters[paramN] = 1

	// 从请求中设置 size
	if strings.TrimSpace(req.Size) != "" {
		parameters[paramSize] = schema.ConvSizeMul(req.Size)
	}

	// 从 callOpts 中解析额外参数
	if callOpts.Extra != nil {
		if v, ok := callOpts.Extra[extraKeyNegativePrompt].(string); ok && v != "" {
			parameters[paramNegativePrompt] = v
		}
		if v, ok := callOpts.Extra[extraKeyPromptExtend].(bool); ok {
			parameters[paramPromptExtend] = v
		}
		if v, ok := callOpts.Extra[extraKeyWatermark].(bool); ok {
			parameters[paramWatermark] = v
		}
		if v, ok := callOpts.Extra[extraKeyN].(int); ok && v > 0 {
			parameters[paramN] = v
		}
		if v, ok := callOpts.Extra[extraKeySeed].(int); ok {
			parameters[paramSeed] = v
		}
	}

	return parameters
}

// --- request payload ---

type apiRequest struct {
	Model      string         `json:"model"`
	Input      input          `json:"input"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type input struct {
	Messages []message `json:"messages"`
}

type message struct {
	Role    string        `json:"role"`
	Content []contentItem `json:"content"`
}

type contentItem struct {
	Text  string `json:"text,omitempty"`  // for input
	Image string `json:"image,omitempty"` // for output
}

// --- response parsing ---

type apiResponse struct {
	Output    output   `json:"output"`
	Usage     apiUsage `json:"usage"`
	RequestID string   `json:"request_id"`
	Code      string   `json:"code,omitempty"`
	Message   string   `json:"message,omitempty"`
}

type output struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	FinishReason string  `json:"finish_reason"`
	Message      message `json:"message"`
}

type apiUsage struct {
	Height     int `json:"height"`
	Width      int `json:"width"`
	ImageCount int `json:"image_count"`
}

func parseResponse(respBody []byte) (*schema.Response, error) {
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, errx.Wrap(err, errx.KindInternal, "decode dashscope text2image response failed")
	}

	if apiResp.Code != "" {
		e := errx.Newf(errx.DashScopeCodeToKind(apiResp.Code),
			"dashscope text2image error: code=%s message=%s", apiResp.Code, apiResp.Message)
		e.Raw = &apiResp
		return nil, e
	}

	if len(apiResp.Output.Choices) == 0 {
		return nil, errx.New(errx.KindInternal, "dashscope text2image response has no choices")
	}

	// 提取图片 URL
	choice := apiResp.Output.Choices[0]
	if len(choice.Message.Content) == 0 {
		return nil, errx.New(errx.KindInternal, "dashscope text2image response has no content")
	}

	// 构建 extras
	extras := make(map[string]any)
	if apiResp.RequestID != "" {
		extras["request_id"] = apiResp.RequestID
	}
	if apiResp.Usage.Height > 0 {
		extras["height"] = apiResp.Usage.Height
	}
	if apiResp.Usage.Width > 0 {
		extras["width"] = apiResp.Usage.Width
	}
	if apiResp.Usage.ImageCount > 0 {
		extras["image_count"] = apiResp.Usage.ImageCount
	}

	// 默认返回 URL 格式的第一张图片
	imageURL := choice.Message.Content[0].Image

	return &schema.Response{
		ResponseFormat: schema.ResponseFormatURL,
		ImageURL:       imageURL,
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
