# multimodal

`multimodal` 是 `gonotelm-lab` 的多模态能力模块。

[![GoReportCard](https://goreportcard.com/badge/gonotelm-lab/multimodal)](https://goreportcard.com/report/github.com/gonotelm-lab/multimodal)

## 安装

```bash
go get github.com/gonotelm-lab/multimodal
```

## 能力

| 能力 | 目录 | Provider |
|------|------|----------|
| 文生图 | `image` | `dashscope`、`agnes` |
| 文生音（TTS） | `audio` | `dashscope`、`minimax`、`mimo` |

## 文生图

以 DashScope 为例：

```go
gen, err := dashscope.New(dashscope.Config{APIKey: "your-api-key"})
if err != nil {
	log.Fatal(err)
}

resp, err := gen.Generate(context.Background(), &schema.Request{
	Prompt: "一只在书桌上看书的橘猫，暖光，写实风格",
	Size:   "1024*1024",
})
if err != nil {
	log.Fatal(err)
}

log.Printf("image url: %s", resp.ImageURL) // 或 resp.ImageBase64
```

| Provider | 包 | 默认模型 | 说明 |
|----------|-----|----------|------|
| DashScope | `image/dashscope` | qwen-image-2.0-pro | `WithNegativePrompt` / `WithPromptExtend` / `WithWatermark` / `WithN` / `WithSeed` |
| Agnes | `image/agnes` | agnes-image-2.1-flash | `ResponseFormat` 支持 Base64 |

## 文生音（TTS）

以 DashScope 为例：

```go
gen, err := dashscope.New(dashscope.Config{APIKey: "your-api-key"})
if err != nil {
	log.Fatal(err)
}

resp, err := gen.Generate(context.Background(), &schema.Request{
	Text:  "今天天气真好，适合出去散步。",
	Voice: "Cherry",
})
if err != nil {
	log.Fatal(err)
}

reader, err := util.ResolveResponse(resp)
if err != nil {
	log.Fatal(err)
}
defer reader.Close()

f, _ := os.Create("output.wav")
io.Copy(f, reader)
```

| Provider | 包 | 默认模型 | 说明 |
|----------|-----|----------|------|
| DashScope | `audio/dashscope` | qwen3-tts-flash | 返回音频 URL |
| MiniMax | `audio/minimax` | speech-2.8-hd | 默认返回音频字节；`WithAudioFormat` / `WithSpeed` / `WithVolume` / `WithPitch` / `WithEmotion` / `WithSubtitleEnable` 等 |
| MiMo | `audio/mimo` | tts | `WithFormat` 控制格式；支持音色克隆、`Instruction` 风格控制 |

## 回调（callbacks）

`image` 与 `audio` 的 `Generate` 支持 eino 风格的 callback（hook）：通过 ctx 注入 handler，
调用开始时触发 `OnStart`，成功时触发 `OnEnd`，失败时触发 `OnError`。

```go
handler := callbacks.NewHandlerBuilder().
	OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		ci := image.ConvCallbackInput(input)
		log.Printf("[%s] start: %q", info.Type, ci.Request.Prompt)
		return ctx
	}).
	OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		co := image.ConvCallbackOutput(output)
		log.Printf("[%s] end: %s", info.Type, co.Response.ImageURL)
		return ctx
	}).
	OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
		log.Printf("[%s] error: %v", info.Type, err)
		return ctx
	}).
	Build()

ctx := callbacks.WithCallbacks(context.Background(), handler)
resp, err := gen.Generate(ctx, &schema.Request{Prompt: "..."})
```

`audio` 同理，换用 `audio.ConvCallbackInput`/`audio.ConvCallbackOutput`。

说明：

- 仅支持 `OnStart`/`OnEnd`/`OnError`（同步调用，无流式）；ctx 沿 handler 链传递（OnStart 倒序、OnEnd/OnError 正序），OnStart 中 `context.WithValue` 的状态默认可达 OnEnd/OnError
- 未注册 handler 的调用零开销
