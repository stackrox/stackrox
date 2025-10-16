package enrichment

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/fetcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"google.golang.org/protobuf/proto"
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

// ImageIntegrationToNodeIntegration converts the given image integration into a node integration.
// Currently, only StackRox Scanner and Scanner v4 are supported node integrations.
// Assumes integration.GetCategories() includes storage.ImageIntegrationCategory_NODE_SCANNER.
func ImageIntegrationToNodeIntegration(integration *storage.ImageIntegration) (*storage.NodeIntegration, error) {
	i := &storage.NodeIntegration{}
	i.SetId(integration.GetId())
	i.SetName(integration.GetName())
	i.SetType(integration.GetType())

	switch integration.GetType() {
	case scannerTypes.ScannerV4:
		i.SetScannerv4(proto.ValueOrDefault(integration.GetScannerV4()))
	case scannerTypes.Clairify:
		i.SetClairify(proto.ValueOrDefault(integration.GetClairify()))
	default:
		return nil, errors.Errorf("unsupported integration type: %q.", integration.GetType())
	}
	log.Debugf("Created Node Integration %s / %s from Image integration", i.GetName(), i.GetType())

	return i, nil
}

func imageIntegrationToOrchestratorIntegration(integration *storage.ImageIntegration) (*storage.OrchestratorIntegration, error) {
	if integration.GetClairify() == nil {
		return nil, errors.Errorf("unsupported orchestrator scanner: %q", integration.GetName())
	}
	oi := &storage.OrchestratorIntegration{}
	oi.SetId(integration.GetId())
	oi.SetName(integration.GetName())
	oi.SetType(integration.GetType())
	oi.SetClairify(proto.ValueOrDefault(integration.GetClairify()))
	return oi, nil
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
	log.Debugf("Converting Integration to Node: %s / %s", integration.GetName(), integration.GetType())
	nodeIntegration, err := ImageIntegrationToNodeIntegration(integration)
	if err != nil {
		return err
	}
	err = m.nodeEnricher.UpsertNodeIntegration(nodeIntegration)
	if err != nil {
		return err
	}

	if integration.GetType() == scannerTypes.ScannerV4 {
		log.Debugf("Scanner v4 is not an orchestrator Scanner, exiting")
		return nil
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
