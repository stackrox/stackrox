package scanners

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries"
	anchoreScanner "github.com/stackrox/rox/pkg/scanners/anchore"
	clairScanner "github.com/stackrox/rox/pkg/scanners/clair"
	clairifyScanner "github.com/stackrox/rox/pkg/scanners/clairify"
	dtrScanner "github.com/stackrox/rox/pkg/scanners/dtr"
	googleScanner "github.com/stackrox/rox/pkg/scanners/google"
	quayScanner "github.com/stackrox/rox/pkg/scanners/quay"
	tenableScanner "github.com/stackrox/rox/pkg/scanners/tenable"
	"github.com/stackrox/rox/pkg/scanners/types"
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
