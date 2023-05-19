package delegatedregistryconfig

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type Delegator interface {
	// DelegateEnrichImage determines if a scan should be delegated, if so delegates it, and returns any errors
	// returns true if scanning of the image should be delegated, false shouldn't be delegated or couldn't be determined
	DelegateEnrichImage(ctx context.Context, image *storage.Image) (bool, error)
}
