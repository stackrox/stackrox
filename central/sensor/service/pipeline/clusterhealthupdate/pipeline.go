package clusterhealthupdate

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clusterhealth"
	"github.com/stackrox/rox/pkg/timestamp"
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
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor,
	injector common.MessageInjector) error {
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

	// We added a response from the ClusterHealth message from Sensor to ensure the Sensor <-> Central connection does
	// not become stale. Since Sensor drops unknown messages, and we do not require acknowledgement from Sensor, we
	// do not have to check for specific sensor capabilities here.
	if err := injector.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_ClusterHealthResponse{},
	}); err != nil {
		return errors.Wrapf(err, "sending cluster health response to cluster %q", clusterID)
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {
}
