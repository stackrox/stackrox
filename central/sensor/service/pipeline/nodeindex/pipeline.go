package nodeindex

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/nodes/enricher"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return newPipeline(clusterDataStore.Singleton(), nodeDatastore.Singleton(), enrichment.NodeEnricherSingleton(), riskManager.Singleton())
}

// newPipeline returns a new instance of Pipeline.
func newPipeline(clusters clusterDataStore.DataStore, nodes nodeDatastore.DataStore, enricher enricher.NodeEnricher, riskManager riskManager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore:  clusters,
		nodeDatastore: nodes,
		enricher:      enricher,
		riskManager:   riskManager,
	}
}

type pipelineImpl struct {
	clusterStore  clusterDataStore.DataStore
	nodeDatastore nodeDatastore.DataStore
	enricher      enricher.NodeEnricher
	riskManager   riskManager.Manager
}

func (p pipelineImpl) OnFinish(_ string) {
}

func (p pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetIndexReport() != nil
}

func (p pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	event := msg.GetEvent()
	report := event.GetIndexReport()
	if report == nil {
		return errors.Errorf("unexpected resource type %T for index report", event.GetResource())
	}
	if event.GetAction() != central.ResourceAction_UNSET_ACTION_RESOURCE {
		log.Errorf("index report from node %s has unsupported action: %q", event.GetNode().GetName(), event.GetAction())
		return nil
	}
	log.Debugf("received node index report with %d packages from %d content sets for node %s",
		len(report.GetContents().Packages), len(report.GetContents().Repositories), event.GetId())
	cr := report.CloneVT()

	// Read the node from the database, if not found we fail.
	node, found, err := p.nodeDatastore.GetNode(ctx, event.GetId())
	if err != nil {
		log.Errorf("fetching node %s from the database: %v", event.GetId(), err)
		return errors.WithMessagef(err, "fetching node: %s", event.GetId())
	}
	if !found {
		log.Errorf("fetching node %s from the database: node does not exist", event.GetId())
		return errors.WithMessagef(err, "node does not exist: %s", event.GetId())
	}

	// Send the Node and Index Report to Scanner for enrichment
	err = p.enricher.EnrichNodeWithInventory(node, nil, cr)
	if err != nil {
		log.Errorf("enriching node %s with index report: %v", event.GetId(), err)
		return errors.WithMessagef(err, "enriching node %s with index report", event.GetId())
	}

	// TODO(ROX-26089): Update the whole node in the database with the new and previous information after conversion
	/*
		err = p.riskManager.CalculateRiskAndUpsertNode(node)
		if err != nil {
			log.Error(err)
			return err
		}
	*/

	return nil

}

func (p pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
