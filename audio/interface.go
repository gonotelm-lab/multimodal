package audio

import (
	"context"

	"github.com/gonotelm-lab/multimodal/audio/schema"
)

type Generator interface {
	Generate(ctx context.Context, req *schema.Request, opts ...Option) (*schema.Response, error)
}
