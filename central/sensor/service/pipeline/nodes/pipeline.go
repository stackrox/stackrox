package nodes

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	countMetrics "github.com/stackrox/rox/central/metrics"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), nodeDatastore.Singleton(), enrichment.NodeEnricherSingleton(), manager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, nodes nodeDatastore.DataStore, enricher enricher.NodeEnricher, riskManager manager.Manager) pipeline.Fragment {
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
	riskManager   manager.Manager
}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := p.nodeDatastore.Search(ctx, query)
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_Node)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "nodes", func(id string) error {
		return p.processRemove(ctx, id)
	})
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNode() != nil
}

func (p *pipelineImpl) processRemove(ctx context.Context, id string) error {
	return p.nodeDatastore.DeleteNodes(ctx, id)
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Node)

	event := msg.GetEvent()
	node := event.GetNode()
	if node == nil {
		return errors.Errorf("unexpected resource type %T for cluster status", event.GetResource())
	}

	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		return p.processRemove(ctx, node.GetId())
	}

	node = node.Clone()
	node.ClusterId = clusterID
	clusterName, ok, err := p.clusterStore.GetClusterName(ctx, clusterID)
	if err == nil && ok {
		node.ClusterName = clusterName
	}

	if enricher.SupportsNodeScanning(node) {
		// If supports node scanning, this pipeline should only update the node's
		// metadata. We call upsert without scan. Upsert will read scan information from
		// the database before writing. This is safe because NodeInventory and Node
		// pipelines never run concurrently by the Sensor Event worker queues.
		node.Scan = nil
		if err := p.nodeDatastore.UpsertNode(ctx, node); err != nil {
			err = errors.Wrapf(err, "upserting node %s", nodeDatastore.NodeString(node))
			log.Error(err)
			return err
		}
		return nil
	}

	err = p.enricher.EnrichNode(node)
	if err != nil {
		log.Warnf("enriching node %s failed (the failure was ignored, vulnerability and "+
			"risk information will not be updated): %v", nodeDatastore.NodeString(node), err)
	}

	if err := p.riskManager.CalculateRiskAndUpsertNode(node); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (p *pipelineImpl) OnFinish(_ string) {}
