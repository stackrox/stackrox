package providers

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/gcp"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// GetMetadata returns the metadata for specific cloud providers
func GetMetadata() *storage.ProviderMetadata {
	metadata, err := gcp.GetMetadata()
	if err == nil {
		return metadata
	}
	log.Errorf("Error getting metadata for GCP: %v", err)

	return nil
}
