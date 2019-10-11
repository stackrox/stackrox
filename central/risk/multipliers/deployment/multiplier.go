package deployment

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Multiplier is the interface that all deployment risk calculations must implement
type Multiplier interface {
	Score(ctx context.Context, deployment *storage.Deployment, image []*storage.Image) *storage.Risk_Result
}
