package auditlogstateupdate

import (
	"context"

	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/protoutils"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters: clusters,
	}
}

type pipelineImpl struct {
	clusters clusterDataStore.DataStore
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	// Nothing to reconcile
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetAuditLogStatusInfo() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	log.Debugf("Received audit log state from sensor: %+v", protoutils.NewWrapper(msg.GetAuditLogStatusInfo()))

	if err := s.clusters.UpdateAuditLogFileStates(ctx, clusterID, msg.GetAuditLogStatusInfo().GetNodeAuditLogFileStates()); err != nil {
		return err
	}
	return nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {
}
