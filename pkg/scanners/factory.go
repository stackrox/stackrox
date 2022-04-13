package scanners

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/registries"
	anchoreScanner "github.com/stackrox/stackrox/pkg/scanners/anchore"
	clairScanner "github.com/stackrox/stackrox/pkg/scanners/clair"
	clairifyScanner "github.com/stackrox/stackrox/pkg/scanners/clairify"
	dtrScanner "github.com/stackrox/stackrox/pkg/scanners/dtr"
	googleScanner "github.com/stackrox/stackrox/pkg/scanners/google"
	quayScanner "github.com/stackrox/stackrox/pkg/scanners/quay"
	tenableScanner "github.com/stackrox/stackrox/pkg/scanners/tenable"
	"github.com/stackrox/stackrox/pkg/scanners/types"
)

// Factory provides a centralized location for creating Scanner from v1.ImageIntegrations.
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
	clairScannerType, clairScannerCreator := clairScanner.Creator()
	reg.creators[clairScannerType] = clairScannerCreator

	anchoreScannerType, anchoreScannerCreator := anchoreScanner.Creator(set)
	reg.creators[anchoreScannerType] = anchoreScannerCreator

	clairifyScannerType, clairifyScannerCreator := clairifyScanner.Creator(set)
	reg.creators[clairifyScannerType] = clairifyScannerCreator

	dtrScannerType, dtrScannerCreator := dtrScanner.Creator()
	reg.creators[dtrScannerType] = dtrScannerCreator

	googleScannerType, googleScannerCreator := googleScanner.Creator()
	reg.creators[googleScannerType] = googleScannerCreator

	quayScannerType, quayScannerCreator := quayScanner.Creator()
	reg.creators[quayScannerType] = quayScannerCreator

	tenableScannerType, tenableScannerCreator := tenableScanner.Creator()
	reg.creators[tenableScannerType] = tenableScannerCreator

	return reg
}
