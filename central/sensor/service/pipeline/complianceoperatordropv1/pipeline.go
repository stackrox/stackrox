package complianceoperatordropv1

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns a pipeline fragment that silently drops deprecated v1
// compliance operator messages. An older sensor may still send these during
// rolling upgrades.
func GetPipeline() pipeline.Fragment {
	return &pipelineImpl{}
}

type pipelineImpl struct{}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	event := msg.GetEvent()
	return event.GetComplianceOperatorResult() != nil ||
		event.GetComplianceOperatorProfile() != nil ||
		event.GetComplianceOperatorRule() != nil ||
		event.GetComplianceOperatorScan() != nil ||
		event.GetComplianceOperatorScanSettingBinding() != nil
}

func (s *pipelineImpl) Run(_ context.Context, _ string, _ *central.MsgFromSensor, _ common.MessageInjector) error {
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
