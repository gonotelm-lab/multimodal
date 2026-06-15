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
