package image

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Multiplier is the interface that all image risk calculations must implement
type Multiplier interface {
	// TODO(ROX-30117): Remove Score after ImageV2 model is fully rolled out
	Score(ctx context.Context, image *storage.Image) *storage.Risk_Result
	ScoreV2(ctx context.Context, image *storage.ImageV2) *storage.Risk_Result
}
