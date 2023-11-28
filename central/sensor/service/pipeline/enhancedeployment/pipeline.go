package enhancedeployment

import (
	"context"

	"github.com/stackrox/rox/central/sensor/enhancement"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

type EnhancementWatcher interface {
	NotifyEnhancementReceived(clusterID string, msg *central.DeploymentEnhancementResponse)
}

func GetPipeline() pipeline.Fragment {
	return NewEnhanceDeploymentPipeline(enhancement.NewWatcher())
}

func NewEnhanceDeploymentPipeline(watcher EnhancementWatcher) pipeline.Fragment {
	return &pipelineImpl{
		watcher: watcher,
	}
}

type pipelineImpl struct {
	watcher EnhancementWatcher
}

func (p pipelineImpl) OnFinish(_ string) {
}

func (p pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetDeploymentEnhancementResponse() != nil
}

func (p pipelineImpl) Run(_ context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	p.watcher.NotifyEnhancementReceived(clusterID, msg.GetDeploymentEnhancementResponse())
	return nil
}

func (p pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
