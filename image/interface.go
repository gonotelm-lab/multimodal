package image

import (
	"context"

	"github.com/gonotelm-lab/multimodal/image/schema"
)

type Generator interface {
	Generate(ctx context.Context, req *schema.Request, opts ...Option) (*schema.Response, error)
}
