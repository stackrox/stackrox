package scanners

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	clairScanner "bitbucket.org/stack-rox/apollo/pkg/scanners/clair"
	clairifyScanner "bitbucket.org/stack-rox/apollo/pkg/scanners/clairify"
	dtrScanner "bitbucket.org/stack-rox/apollo/pkg/scanners/dtr"
	googleScanner "bitbucket.org/stack-rox/apollo/pkg/scanners/google"
	quayScanner "bitbucket.org/stack-rox/apollo/pkg/scanners/quay"
	tenableScanner "bitbucket.org/stack-rox/apollo/pkg/scanners/tenable"
	"bitbucket.org/stack-rox/apollo/pkg/scanners/types"
)

// Creator is the func stub that defines how to instantiate an image scanner.
type Creator func(scanner *v1.ImageIntegration) (types.ImageScanner, error)

// Factory provides a centralized location for creating ImageScanner from v1.ImageIntegrations.
type Factory interface {
	CreateScanner(source *v1.ImageIntegration) (types.ImageScanner, error)
}

// NewFactory creates a new scanner factory.
func NewFactory(set registries.Set) Factory {
	reg := &factoryImpl{
		creators: make(map[string]Creator),
	}

	// Add scanners to factory.
	////////////////////////////
	clairScannerType, clairScannerCreator := clairScanner.Creator()
	reg.creators[clairScannerType] = clairScannerCreator

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
