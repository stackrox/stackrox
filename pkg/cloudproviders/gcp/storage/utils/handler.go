package utils

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/auth"
	gcpStorage "github.com/stackrox/rox/pkg/cloudproviders/gcp/storage"
	"github.com/stackrox/rox/pkg/features"
	"golang.org/x/oauth2/google"
	googleStoragev1 "google.golang.org/api/storage/v1"
)

// CreateHandlerFromConfig creates a handler based on the GCS integration configuration.
func CreateHandlerFromConfig(ctx context.Context,
	manager auth.STSClientManager, conf *storage.GCSConfig,
) (gcpStorage.ClientHandler, error) {
	if !conf.GetUseWorkloadId() {
		creds, err := google.CredentialsFromJSON(
			ctx,
			[]byte(conf.GetServiceAccount()),
			googleStoragev1.CloudPlatformScope,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create credentials")
		}
		return gcpStorage.NewClientHandler(ctx, creds)
	}

	if features.CloudCredentials.Enabled() {
		return manager.StorageClientHandler(), nil
	}

	creds, err := google.FindDefaultCredentials(ctx, googleStoragev1.CloudPlatformScope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credentials")
	}
	return gcpStorage.NewClientHandler(ctx, creds)
}
