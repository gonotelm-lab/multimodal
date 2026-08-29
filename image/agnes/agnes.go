package agnes

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

const (
	runType        = "agnes"
	defaultBaseUrl = "https://apihub.agnes-ai.com/v1/images/generations"
	defaultModel   = "agnes-image-2.1-flash"
)

type Generator struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config, opts ...images.ClientOption) (*Generator, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errx.New(errx.KindInvalidArgument, "agnes api key is required")
	}
	if strings.TrimSpace(cfg.BaseUrl) == "" {
		cfg.BaseUrl = defaultBaseUrl
	} else {
		// Ensure it points to the correct endpoint if they just provided the base url
		if !strings.HasSuffix(cfg.BaseUrl, "/images/generations") {
			cfg.BaseUrl = strings.TrimRight(cfg.BaseUrl, "/") + "/images/generations"
		}
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

	payload := apiRequest{
		Model:        model,
		Prompt:       req.Prompt,
		ReturnBase64: req.ResponseFormat == schema.ResponseFormatBase64,
	}
	if !payload.ReturnBase64 {
		payload.ExtraBody = &apiRequestExtraBody{
			ResponseFormat: "url",
		}
	}

	if strings.TrimSpace(req.Size) != "" {
		payload.Size = req.Size
		payload.Size = schema.ConvSizeX(req.Size)
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "marshal agnes text2image request failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errx.Wrap(err, errx.KindInvalidArgument, "build agnes text2image request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "call agnes text2image failed")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errx.Wrap(err, errx.KindNetwork, "read agnes text2image response failed")
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return nil, openAIHTTPErrorToErr(httpResp.StatusCode, respBody, "agnes", "text2image")
	}

	resp, err = parseResponse(respBody)
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnEnd(ctx, &images.CallbackOutput{Response: resp})
	return resp, nil
}

// --- request payload ---

type apiRequest struct {
	Model        string               `json:"model"`
	Prompt       string               `json:"prompt"`
	Size         string               `json:"size,omitempty"`
	ReturnBase64 bool                 `json:"return_base64,omitempty"`
	ExtraBody    *apiRequestExtraBody `json:"extra_body,omitempty"`
}

type apiRequestExtraBody struct {
	ResponseFormat string `json:"response_format,omitempty"`
}

// --- response parsing ---

type apiResponse struct {
	Created int64      `json:"created"`
	Data    []dataItem `json:"data"`
	Error   *apiError  `json:"error,omitempty"`
}

type dataItem struct {
	URL           string `json:"url"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func parseResponse(respBody []byte) (*schema.Response, error) {
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, errx.Wrap(err, errx.KindInternal, "decode agnes text2image response failed")
	}

	if apiResp.Error != nil {
		e := errx.Newf(errx.OpenAIErrorTypeToKind(apiResp.Error.Type),
			"agnes text2image error: type=%s code=%s message=%s",
			apiResp.Error.Type, apiResp.Error.Code, apiResp.Error.Message)
		e.Raw = apiResp.Error
		return nil, e
	}

	if len(apiResp.Data) == 0 {
		return nil, errx.New(errx.KindInternal, "agnes text2image response has no data")
	}

	item := apiResp.Data[0]

	extras := make(map[string]any)
	if item.RevisedPrompt != "" {
		extras["revised_prompt"] = item.RevisedPrompt
	}

	// 优先返回 Base64 (如果存在)，否则返回 URL
	if item.B64JSON != "" {
		return &schema.Response{
			ResponseFormat: schema.ResponseFormatBase64,
			ImageBase64:    item.B64JSON,
			Extras:         extras,
		}, nil
	}

	if item.URL == "" {
		return nil, errx.New(errx.KindInternal, "agnes text2image response has no url or b64_json")
	}
	return &schema.Response{
		ResponseFormat: schema.ResponseFormatURL,
		ImageURL:       item.URL,
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
