package scanners

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries"
	clairifyScanner "github.com/stackrox/rox/pkg/scanners/clairify"
	clairV4Scanner "github.com/stackrox/rox/pkg/scanners/clairv4"
	googleScanner "github.com/stackrox/rox/pkg/scanners/google"
	quayScanner "github.com/stackrox/rox/pkg/scanners/quay"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// Factory provides a centralized location for creating Scanner from v1.ImageIntegrations.
//
//go:generate mockgen-wrapper
type Factory interface {
	CreateScanner(source *storage.ImageIntegration) (types.ImageScannerWithDataSource, error)
}

// NewFactory creates a new scanner factory.
func NewFactory(set registries.Set) Factory {
	reg := &factoryImpl{
		creators: make(map[string]Creator),
	}

	// Add image scanners to factory.
	/////////////////////////////////
	clairV4ScannerType, clairV4ScannerCreator := clairV4Scanner.Creator(set)
	reg.creators[clairV4ScannerType] = clairV4ScannerCreator

	clairifyScannerType, clairifyScannerCreator := clairifyScanner.Creator(set)
	reg.creators[clairifyScannerType] = clairifyScannerCreator

	googleScannerType, googleScannerCreator := googleScanner.Creator()
	reg.creators[googleScannerType] = googleScannerCreator

	quayScannerType, quayScannerCreator := quayScanner.Creator()
	reg.creators[quayScannerType] = quayScannerCreator

	return reg
}
