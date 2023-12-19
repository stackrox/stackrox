package enhancements

import (
	"context"

	"github.com/stackrox/rox/central/sensor/enhancement"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// EnhancementBroker is the interface that will be notified when an augmented deployment from Sensor arrives
type EnhancementBroker interface {
	NotifyDeploymentReceived(msg *central.DeploymentEnhancementResponse)
}

type pipelineImpl struct {
	broker EnhancementBroker
}

// NewAugmentPipeline returns a new instance of the Augmentation Pipeline
func NewAugmentPipeline(broker EnhancementBroker) pipeline.Fragment {
	return &pipelineImpl{broker: broker}
}

// GetPipeline returns a new pipeline
func GetPipeline() pipeline.Fragment {
	return NewAugmentPipeline(enhancement.BrokerSingleton())
}

// OnFinish .
func (p pipelineImpl) OnFinish(_ string) {
}

// Capabilities .
func (p pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

// Match .
func (p pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetDeploymentEnhancementResponse() != nil
}

// Run .
func (p pipelineImpl) Run(_ context.Context, _ string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	p.broker.NotifyDeploymentReceived(msg.GetDeploymentEnhancementResponse())
	return nil
}

// Reconcile .
func (p pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
