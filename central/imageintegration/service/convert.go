package service

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
)

// FIXME: Why do we have duplication of this fn in Central?
/*// imageIntegrationToNodeIntegration converts the given image integration into a node integration.
// Currently, only Clairify is a supported node integration.
// Assumes integration.GetCategories() includes storage.ImageIntegrationCategory_NODE.
func imageIntegrationToNodeIntegration(integration *storage.ImageIntegration) (*storage.NodeIntegration, error) {
	if integration.GetClairify() == nil {
		return nil, errors.Errorf("unsupported node scanner: %q", integration.GetName())
	}

	return &storage.NodeIntegration{
		Id:   integration.GetId(),
		Name: integration.GetName(),
		Type: integration.GetType(),
		IntegrationConfig: &storage.NodeIntegration_Clairify{
			Clairify: integration.GetClairify(),
		},
	}, nil
}
*/

// imageIntegrationToNodeIntegration converts the given image integration into a node integration.
// Currently, only StackRox Scanner and Scanner v4 are supported node integrations.
// Assumes integration.GetCategories() includes storage.ImageIntegrationCategory_NODE_SCANNER.
func imageIntegrationToNodeIntegration(integration *storage.ImageIntegration) (*storage.NodeIntegration, error) {
	if integration.GetType() != scannerTypes.Clairify && integration.GetType() != scannerTypes.ScannerV4 {
		return nil, errors.Errorf("requires a %s or %s config: %q", scannerTypes.Clairify, scannerTypes.ScannerV4, integration.GetName())
	}
	i := &storage.NodeIntegration{
		Id:   integration.GetId(),
		Name: integration.GetName(),
		Type: integration.GetType(),
	}

	switch integration.GetType() {
	case scannerTypes.ScannerV4:
		i.IntegrationConfig = &storage.NodeIntegration_Scannerv4{
			Scannerv4: integration.GetScannerV4(),
		}
	case scannerTypes.Clairify:
		i.IntegrationConfig = &storage.NodeIntegration_Clairify{
			Clairify: integration.GetClairify(),
		}
	default:
		return nil, errors.Errorf("unsupported integration type: %q", integration.GetType())
	}

	return i, nil
}
