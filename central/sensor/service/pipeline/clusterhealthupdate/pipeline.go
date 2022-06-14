package clusterhealthupdate

import (
	"context"

	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/centralsensor"
	"github.com/stackrox/stackrox/pkg/clusterhealth"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/timestamp"
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
	return msg.GetClusterHealthInfo() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	cInfo := msg.GetClusterHealthInfo().GetCollectorHealthInfo()
	aInfo := msg.GetClusterHealthInfo().GetAdmissionControlHealthInfo()
	sInfo := msg.GetClusterHealthInfo().GetScannerHealthInfo()

	conn := connection.FromContext(ctx)
	clusterHealthStatus := &storage.ClusterHealthStatus{
		CollectorHealthInfo:          cInfo,
		AdmissionControlHealthInfo:   aInfo,
		ScannerHealthInfo:            sInfo,
		SensorHealthStatus:           storage.ClusterHealthStatus_HEALTHY,
		CollectorHealthStatus:        clusterhealth.PopulateCollectorStatus(cInfo),
		AdmissionControlHealthStatus: clusterhealth.PopulateAdmissionControlStatus(aInfo),
		ScannerHealthStatus:          clusterhealth.PopulateLocalScannerStatus(sInfo),
		LastContact:                  timestamp.Now().GogoProtobuf(),
		// When sensor health monitoring is revised update the sensor capability
		HealthInfoComplete: conn != nil && conn.HasCapability(centralsensor.HealthMonitoringCap),
	}
	clusterHealthStatus.OverallHealthStatus = clusterhealth.PopulateOverallClusterStatus(clusterHealthStatus)

	if err := s.clusters.UpdateClusterHealth(ctx, clusterID, clusterHealthStatus); err != nil {
		return err
	}
	return nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {
}
