package image

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Multiplier is the interface that all image risk calculations must implement
type Multiplier interface {
	Score(ctx context.Context, image *storage.Image) *storage.Risk_Result
}
