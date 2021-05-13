package scanners

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// Creator is the func stub that defines how to instantiate an image scanner.
type Creator func(scanner *storage.ImageIntegration) (types.Scanner, error)

// NodeScannerCreator is the func stub that defines how to instantiate a node scanner.
type NodeScannerCreator func(scanner *storage.NodeIntegration) (types.NodeScanner, error)

// OrchestratorScannerCreator is a func stub that defines how to instantiate an orchestrator scanner.
type OrchestratorScannerCreator func(scanner *storage.OrchestratorIntegration) (types.OrchestratorScanner, error)
