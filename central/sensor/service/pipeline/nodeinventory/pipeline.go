package nodeinventory

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	countMetrics "github.com/stackrox/rox/central/metrics"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/nodes/enricher"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), nodeDatastore.Singleton(), enrichment.NodeEnricherSingleton(), riskManager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, nodes nodeDatastore.DataStore, enricher enricher.NodeEnricher, riskManager riskManager.Manager) pipeline.Fragment {
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

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNodeInventory() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.NodeInventory)

	event := msg.GetEvent()
	ninv := event.GetNodeInventory()
	if ninv == nil {
		return errors.Errorf("unexpected resource type %T for node inventory", event.GetResource())
	}

	log.Infof("Central received NodeInventory for Node name='%s' ID='%s'", ninv.GetNodeName(), ninv.GetNodeId())

	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		// NodeInventory will never be deleted
		return nil
	}

	ninv = ninv.Clone()

	// TODO(ROX-14484): Resolve the race between pipelines - Start of critical section
	node, found, err := p.nodeDatastore.GetNode(ctx, ninv.GetNodeId())
	if err != nil || !found {
		log.Warnf("Node ID %s not found when processing NodeInventory", ninv.GetNodeId())
		return errors.WithMessagef(err, "processing node inventory for node '%s'", ninv.GetNodeId())
	}
	log.Debugf("Node ID %s found. Will enrich Node with NodeInventory", ninv.GetNodeId())

	err = p.enricher.EnrichNodeWithInventory(node, ninv)
	if err != nil {
		log.Warnf("enriching node with node inventory %s:%s: %v", node.GetClusterName(), node.GetName(), err)
	}

	// Here NodeInventory stops to matter. All data required for the DB and UI is in node.NodeScan already

	if err := p.riskManager.CalculateRiskAndUpsertNode(node); err != nil {
		err = errors.Wrapf(err, "upserting node %s:%s into datastore", node.GetClusterName(), node.GetName())
		log.Error(err)
		return err
	}
	// TODO(ROX-14484): Resolve the race between pipelines - End of critical section (when CalculateRiskAndUpsertNode finishes)
	// We will loose data written in the node pipeline if the node pipeline writes an update to the DB
	// while this pipeline is in the critical section!
	return nil
}

func (p *pipelineImpl) OnFinish(_ string) {}
