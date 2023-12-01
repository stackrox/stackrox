package augment

import (
	"context"

	"github.com/stackrox/rox/central/sensor/augmentation"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

type AugmentationBroker interface {
	NotifyDeploymentReceived(msg *central.DeploymentEnhancementResponse)
}

type pipelineImpl struct {
	broker AugmentationBroker
}

// NewAugmentPipeline returns a new instance of the Augmentation Pipeline
func NewAugmentPipeline(broker AugmentationBroker) pipeline.Fragment {
	return &pipelineImpl{broker: broker}
}

func GetPipeline() pipeline.Fragment {
	return NewAugmentPipeline(augmentation.BrokerSingleton())
}

// OnFinish .
func (p pipelineImpl) OnFinish(clusterID string) {
}

// Capabilities .
func (p pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

// Match .
func (p pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetDeploymentEnhancementResponse() != nil
}

func (p pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	p.broker.NotifyDeploymentReceived(msg.GetDeploymentEnhancementResponse())
	return nil
}

func (p pipelineImpl) Reconcile(ctx context.Context, clusterID string, reconciliationStore *reconciliation.StoreMap) error {
	return nil
}
