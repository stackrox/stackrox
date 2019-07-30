package scannerv2

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners/anchore"
	"github.com/stackrox/rox/pkg/scanners/types"
)

const (
	typeString = "scanner"
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (types.ImageScanner, error)) {
	// Just piggyback on the Anchore implementation.
	_, anchoreCreator := anchore.Creator(set)
	anchoreWrapper := func(integration *storage.ImageIntegration) (types.ImageScanner, error) {
		endpoint := integration.GetScannerv2().GetEndpoint()
		anchoreConfig := storage.AnchoreConfig{
			Endpoint: endpoint,
			Username: "admin",
			// TODO(viswa): Should we make this parameterized? Unclear.
			Password: "foobar",
		}
		newIntegration := protoutils.CloneStorageImageIntegration(integration)
		newIntegration.IntegrationConfig = &storage.ImageIntegration_Anchore{Anchore: &anchoreConfig}
		return anchoreCreator(newIntegration)
	}
	return typeString, anchoreWrapper
}
