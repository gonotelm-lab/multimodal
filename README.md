# multimodal

`multimodal` 是 `gonotelm-lab` 的多模态能力模块。

[![GoReportCard](https://goreportcard.com/badge/gonotelm-lab/multimodal)](https://goreportcard.com/report/github.com/gonotelm-lab/multimodal)

## 安装

```bash
go get github.com/gonotelm-lab/multimodal
```

## 文生图

### DashScope

```go
package main

import (
	"context"
	"log"

	"github.com/gonotelm-lab/multimodal/image/dashscope"
	"github.com/gonotelm-lab/multimodal/image/schema"
)

func main() {
	gen, err := dashscope.New(dashscope.Config{
		APIKey: "your-api-key",
		// BaseUrl / Model / Timeout 为空时会使用默认值
	})
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

	log.Printf("image url: %s", resp.ImageURL)
}
```

### Agnes

```go
package main

import (
	"context"
	"log"

	"github.com/gonotelm-lab/multimodal/image/agnes"
	"github.com/gonotelm-lab/multimodal/image/schema"
)

func main() {
	gen, err := agnes.New(agnes.Config{
		APIKey: "your-api-key",
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := gen.Generate(context.Background(), &schema.Request{
		Prompt:         "一条在森林里发光的小路，电影感",
		ResponseFormat: schema.ResponseFormatBase64,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("base64 length: %d", len(resp.ImageBase64))
}
```

## 文生音（TTS）

### DashScope

```go
package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/gonotelm-lab/multimodal/audio/dashscope"
	"github.com/gonotelm-lab/multimodal/audio/schema"
	"github.com/gonotelm-lab/multimodal/audio/util"
)

func main() {
	gen, err := dashscope.New(dashscope.Config{
		APIKey: "your-api-key",
		// BaseUrl / Model / Timeout 为空时会使用默认值
	})
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

	// 保存到文件
	f, err := os.Create("output.wav")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	io.Copy(f, reader)
}
```

### MiniMax

```go
package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/gonotelm-lab/multimodal/audio/minimax"
	"github.com/gonotelm-lab/multimodal/audio/schema"
)

func main() {
	gen, err := minimax.New(minimax.Config{
		APIKey: "your-api-key",
		// BaseUrl / Model / Timeout 为空时会使用默认值
	})
	if err != nil {
		log.Fatal(err)
	}

	// 默认 output_format=hex，直接返回音频字节
	resp, err := gen.Generate(context.Background(), &schema.Request{
		Text:  "今天天气真好，适合出去散步。",
		Voice: "male-qn-qingse",
	}, minimax.WithAudioFormat(minimax.AudioFormatMP3))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Reader.Close()

	// 保存到文件
	f, err := os.Create("output.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	io.Copy(f, resp.Reader)
}
```

### MiMo

```go
package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/gonotelm-lab/multimodal/audio/mimo"
	"github.com/gonotelm-lab/multimodal/audio/schema"
	"github.com/gonotelm-lab/multimodal/audio/util"
)

func main() {
	gen, err := mimo.New(mimo.Config{
		APIKey: "your-api-key",
		// BaseUrl / Model / Timeout 为空时会使用默认值
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := gen.Generate(context.Background(), &schema.Request{
		Text:  "Hey boss — guess what, I actually passed the exam with distinction!",
		Voice: string(mimo.VoiceChloe),
		// Instruction 可选，用于自然语言风格控制（放在 user 消息中）
		// Instruction: "Bright, bouncy tone, fast pace.",
	}, mimo.WithFormat(mimo.FormatWAV))
	if err != nil {
		log.Fatal(err)
	}

	reader, err := util.ResolveResponse(resp)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	f, err := os.Create("output.wav")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	io.Copy(f, reader)
}
```
