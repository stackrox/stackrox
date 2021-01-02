package component

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
)

// Multiplier is the interface that all component risk calculations must implement
type Multiplier interface {
	Score(ctx context.Context, component scancomponent.ScanComponent) *storage.Risk_Result
}
