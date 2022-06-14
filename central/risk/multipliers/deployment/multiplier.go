package deployment

import (
	"context"

	"github.com/stackrox/stackrox/generated/storage"
)

// Multiplier is the interface that all deployment risk calculations must implement
type Multiplier interface {
	Score(ctx context.Context, deployment *storage.Deployment, imageRiskResults map[string][]*storage.Risk_Result) *storage.Risk_Result
}
