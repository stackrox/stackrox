package providers

import (
	"context"
	"errors"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/aws"
	"github.com/stackrox/rox/pkg/cloudproviders/azure"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// GetMetadata returns the metadata for specific cloud providers
func GetMetadata(ctx context.Context) *storage.ProviderMetadata {
	var metadataErrs error
	metadata, err := gcp.GetMetadata(ctx)
	if metadata != nil {
		return metadata
	}
	metadataErrs = errors.Join(metadataErrs, pkgErrors.Wrap(err, "GKE"))

	metadata, err = aws.GetMetadata(ctx)
	if metadata != nil {
		return metadata
	}
	metadataErrs = errors.Join(metadataErrs, pkgErrors.Wrap(err, "AWS"))

	metadata, err = azure.GetMetadata(ctx)
	if metadata != nil {
		return metadata
	}
	metadataErrs = errors.Join(metadataErrs, pkgErrors.Wrap(err, "Azure"))

	if metadataErrs != nil {
		log.Error(metadata)
	}

	return nil
}
