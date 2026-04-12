//go:build !gcpcloud

package gcp

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// GetMetadata returns nil when built without the gcpcloud tag.
func GetMetadata(_ context.Context) (*storage.ProviderMetadata, error) {
	return nil, nil
}
