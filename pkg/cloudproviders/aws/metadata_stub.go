//go:build !awscloud

package aws

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// GetMetadata returns nil when built without the awscloud tag.
// Build with -tags awscloud to enable AWS cloud provider detection.
func GetMetadata(_ context.Context) (*storage.ProviderMetadata, error) {
	return nil, nil
}
