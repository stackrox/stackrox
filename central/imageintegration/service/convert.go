package service

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// imageIntegrationToNodeIntegration converts the given image integration into a node integration.
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
