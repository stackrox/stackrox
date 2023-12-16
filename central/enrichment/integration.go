package enrichment

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/fetcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
)

// Manager implements a bit of multiplexing logic between ImageIntegrations and NodeIntegrations
//
//go:generate mockgen-wrapper
type Manager interface {
	Upsert(integration *storage.ImageIntegration) error
	Remove(id string) error
}

func newManager(imageIntegrationSet integration.Set, nodeEnricher enricher.NodeEnricher, cveFetcher fetcher.OrchestratorIstioCVEManager) Manager {
	return &managerImpl{
		imageIntegrationSet: imageIntegrationSet,
		nodeEnricher:        nodeEnricher,
		cveFetcher:          cveFetcher,
	}
}

type managerImpl struct {
	imageIntegrationSet integration.Set
	nodeEnricher        enricher.NodeEnricher
	cveFetcher          fetcher.OrchestratorIstioCVEManager
}

// isNodeIntegration returns "true" if the image integration is also a node integration.
// It loops through the categories, which is a very small slice.
func isNodeIntegration(integration *storage.ImageIntegration) bool {
	for _, category := range integration.GetCategories() {
		if category == storage.ImageIntegrationCategory_NODE_SCANNER {
			return true
		}
	}
	return false
}

// imageIntegrationToNodeIntegration converts the given image integration into a node integration.
// Currently, only StackRox Scanner is a supported node integration.
// Assumes integration.GetCategories() includes storage.ImageIntegrationCategory_NODE.
func imageIntegrationToNodeIntegration(integration *storage.ImageIntegration) (*storage.NodeIntegration, error) {
	if integration.GetType() != scannerTypes.Clairify {
		return nil, errors.Errorf("requires a clarify config: %q", integration.GetName())
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

func imageIntegrationToOrchestratorIntegration(integration *storage.ImageIntegration) (*storage.OrchestratorIntegration, error) {
	if integration.GetClairify() == nil {
		return nil, errors.Errorf("unsupported orchestrator scanner: %q", integration.GetName())
	}
	return &storage.OrchestratorIntegration{
		Id:   integration.GetId(),
		Name: integration.GetName(),
		Type: integration.GetType(),
		IntegrationConfig: &storage.OrchestratorIntegration_Clairify{
			Clairify: integration.GetClairify(),
		},
	}, nil
}

func (m *managerImpl) Upsert(integration *storage.ImageIntegration) error {
	if err := m.imageIntegrationSet.UpdateImageIntegration(integration); err != nil {
		return err
	}
	if !isNodeIntegration(integration) {
		m.nodeEnricher.RemoveNodeIntegration(integration.GetId())
		// Use node integration for now because node scanner is also orchestrator scanner.
		m.cveFetcher.RemoveIntegration(integration.GetId())
		return nil
	}
	nodeIntegration, err := imageIntegrationToNodeIntegration(integration)
	if err != nil {
		return err
	}
	err = m.nodeEnricher.UpsertNodeIntegration(nodeIntegration)
	if err != nil {
		return err
	}

	orchestratorIntegration, err := imageIntegrationToOrchestratorIntegration(integration)
	if err != nil {
		return err
	}
	return m.cveFetcher.UpsertOrchestratorIntegration(orchestratorIntegration)
}

func (m *managerImpl) Remove(id string) error {
	if err := m.imageIntegrationSet.RemoveImageIntegration(id); err != nil {
		return err
	}
	m.nodeEnricher.RemoveNodeIntegration(id)
	return nil
}
