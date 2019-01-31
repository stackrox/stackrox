package providermetadata

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
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
func NewPipeline(clusters clusterDataStore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters: clusters,
	}
}

type pipelineImpl struct {
	clusters clusterDataStore.DataStore
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetProviderMetadata() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(msg *central.MsgFromSensor, _ pipeline.MsgInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ProviderMetadata)

	event := msg.GetEvent()
	return s.clusters.UpdateMetadata(event.GetClusterId(), event.GetProviderMetadata())
}

func (s *pipelineImpl) OnFinish() {}
