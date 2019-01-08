package nodes

import (
	"fmt"

	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(nodes store.GlobalStore) pipeline.Fragment {
	return &pipelineImpl{
		nodeStore: nodes,
	}
}

type pipelineImpl struct {
	nodeStore store.GlobalStore
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNode() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(msg *central.MsgFromSensor, _ pipeline.MsgInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Node)

	event := msg.GetEvent()
	clusterID := event.GetClusterId()

	store, err := p.nodeStore.GetClusterNodeStore(clusterID)
	if err != nil {
		return fmt.Errorf("getting cluster-local node store: %v", err)
	}

	nodeOneof, ok := event.Resource.(*central.SensorEvent_Node)
	if !ok {
		return fmt.Errorf("unexpected resource type %T for cluster status", event.Resource)
	}
	node := nodeOneof.Node

	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		return store.RemoveNode(node.GetId())
	}
	return store.UpsertNode(node)
}
