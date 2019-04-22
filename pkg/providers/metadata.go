package providers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/aws"
	"github.com/stackrox/rox/pkg/cloudproviders/azure"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// GetMetadata returns the metadata for specific cloud providers
func GetMetadata(ctx context.Context) *storage.ProviderMetadata {
	errors := errorhelpers.NewErrorList("getting cloud provider metadata")
	metadata, err := gcp.GetMetadata(ctx)
	if metadata != nil {
		return metadata
	}
	errors.AddWrap(err, "GCP")

	metadata, err = aws.GetMetadata(ctx)
	if metadata != nil {
		return metadata
	}
	errors.AddWrap(err, "AWS")

	metadata, err = azure.GetMetadata(ctx)
	if metadata != nil {
		return metadata
	}
	errors.AddWrap(err, "Azure")

	if err := errors.ToError(); err != nil {
		log.Error(err)
	}

	return nil
}
