package imagecomponent

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Multiplier is the interface that all image component risk calculations must implement
type Multiplier interface {
	Score(ctx context.Context, imageComponent *storage.EmbeddedImageScanComponent) *storage.Risk_Result
}
