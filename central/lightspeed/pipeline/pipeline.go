package lightspeedpipeline

import (
	"context"

	"github.com/stackrox/rox/central/lightspeed/datastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)

	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this lightspeed pipeline.
func GetPipeline() pipeline.Fragment {
	return NewPipeline(datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(ds datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		datastore: ds,
	}
}

type pipelineImpl struct {
	datastore datastore.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetLightspeedInfo() != nil
}

func (s *pipelineImpl) Run(_ context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	info := msg.GetLightspeedInfo()
	log.Debugf("Received Lightspeed info from cluster %s: available=%v endpoint=%s", clusterID, info.GetIsAvailable(), info.GetEndpoint())
	s.datastore.Update(clusterID, datastore.LightspeedInfo{
		IsAvailable: info.GetIsAvailable(),
		Endpoint:    info.GetEndpoint(),
	})
	return nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {
	log.Debugf("Sensor disconnected for cluster %s, marking Lightspeed unavailable", clusterID)
	s.datastore.Remove(clusterID)
}
