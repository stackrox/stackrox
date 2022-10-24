package nodescansv2

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/node/globaldatastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/nodes/enricher"
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
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNodeScanV2() != nil
}

func (p *pipelineImpl) processRemove(ds datastore.DataStore, n *storage.Node) error {
	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.NodeScanV2)

	event := msg.GetEvent()
	log.Infof("nodescansv2: received event: %+v", msg.GetEvent().String())
	nodeScan := event.GetNodeScanV2()
	if nodeScan == nil {
		return errors.Errorf("unexpected resource type %T for cluster status", event.GetResource())
	}
	log.Infof("Central received NodeScanV2: %+v", nodeScan)

	return nil
}

func (p *pipelineImpl) OnFinish(_ string) {}
