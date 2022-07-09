package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for logs.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.LogImbue, error)
	Upsert(ctx context.Context, log *storage.LogImbue) error
}
