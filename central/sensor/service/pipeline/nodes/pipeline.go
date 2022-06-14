package nodes

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/enrichment"
	countMetrics "github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/node/datastore"
	"github.com/stackrox/stackrox/central/node/globaldatastore"
	"github.com/stackrox/stackrox/central/risk/manager"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/nodes/enricher"
	"github.com/stackrox/stackrox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), globaldatastore.Singleton(), enrichment.NodeEnricherSingleton(), manager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, nodes globaldatastore.GlobalDataStore, enricher enricher.NodeEnricher, riskManager manager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore: clusters,
		nodeStore:    nodes,
		enricher:     enricher,
		riskManager:  riskManager,
	}
}

type pipelineImpl struct {
	clusterStore clusterDataStore.DataStore
	nodeStore    globaldatastore.GlobalDataStore
	enricher     enricher.NodeEnricher
	riskManager  manager.Manager
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := p.nodeStore.Search(ctx, query)
	if err != nil {
		return err
	}

	clusterStore, err := p.nodeStore.GetClusterNodeStore(ctx, clusterID, true)
	if err != nil {
		return errors.Wrap(err, "getting cluster-local node store")
	}

	store := storeMap.Get((*central.SensorEvent_Node)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "nodes", func(id string) error {
		return p.processRemove(clusterStore, &storage.Node{Id: id})
	})
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNode() != nil
}

func (p *pipelineImpl) processRemove(ds datastore.DataStore, n *storage.Node) error {
	return ds.RemoveNode(n.GetId())
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Node)

	event := msg.GetEvent()
	node := event.GetNode()
	if node == nil {
		return errors.Errorf("unexpected resource type %T for cluster status", event.GetResource())
	}

	store, err := p.nodeStore.GetClusterNodeStore(ctx, clusterID, true)
	if err != nil {
		return errors.Wrap(err, "getting cluster-local node store")
	}

	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		return p.processRemove(store, node)
	}

	node = node.Clone()
	node.ClusterId = clusterID
	clusterName, ok, err := p.clusterStore.GetClusterName(ctx, clusterID)
	if err == nil && ok {
		node.ClusterName = clusterName
	}

	err = p.enricher.EnrichNode(node)
	if err != nil {
		log.Warnf("enriching node %s:%s: %v", node.GetClusterName(), node.GetName(), err)
	}

	if err := p.riskManager.CalculateRiskAndUpsertNode(node); err != nil {
		err = errors.Wrapf(err, "upserting node %s:%s into datastore", node.GetClusterName(), node.GetName())
		log.Error(err)
		return err
	}

	return nil
}

func (p *pipelineImpl) OnFinish(_ string) {}
