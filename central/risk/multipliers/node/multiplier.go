package node

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Multiplier is the interface that all node risk calculations must implement
type Multiplier interface {
	Score(ctx context.Context, node *storage.Node) *storage.Risk_Result
}
