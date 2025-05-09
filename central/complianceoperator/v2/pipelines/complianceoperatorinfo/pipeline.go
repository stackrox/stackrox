package complianceoperatorinfo

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	_   pipeline.Fragment = (*pipelineImpl)(nil)
	log                   = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this compliance operator info pipeline.
func GetPipeline() pipeline.Fragment {
	return NewPipeline(compliancemanager.Singleton(), TelemetrySingleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(manager compliancemanager.Manager, cot telemetry) pipeline.Fragment {
	return &pipelineImpl{
		manager: manager,
		cot:     cot,
	}
}

type pipelineImpl struct {
	manager compliancemanager.Manager
	cot     telemetry
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetComplianceOperatorInfo() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorInfo)
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	var operatorErrors []string
	operatorInfo := &storage.ComplianceIntegration{
		Version:             msg.GetComplianceOperatorInfo().GetVersion(),
		ClusterId:           clusterID,
		ComplianceNamespace: msg.GetComplianceOperatorInfo().GetNamespace(),
		OperatorInstalled:   msg.GetComplianceOperatorInfo().GetIsInstalled(),
	}
	if updated := s.cot.TrackVersion(operatorInfo.GetVersion()); updated {
		log.Info("Updating telemetry for version of the compliance operator: %s", operatorInfo.GetVersion())
	}

	if msg.GetComplianceOperatorInfo().GetStatusError() != "" {
		operatorErrors = append(operatorErrors, msg.GetComplianceOperatorInfo().GetStatusError())
	}

	desiredPods := msg.GetComplianceOperatorInfo().GetTotalDesiredPods()
	readyPods := msg.GetComplianceOperatorInfo().GetTotalReadyPods()

	// if not ready, add it to the status errors
	if readyPods < desiredPods {
		operatorErrors = append(operatorErrors, fmt.Sprintf("compliance operator not ready.  Only %d pods are ready when %d are desired.", readyPods, desiredPods))
	}

	operatorInfo.StatusErrors = operatorErrors
	operatorInfo.OperatorStatus = storage.COStatus_UNHEALTHY
	if len(operatorInfo.StatusErrors) == 0 {
		operatorInfo.OperatorStatus = storage.COStatus_HEALTHY
	}
	return s.manager.ProcessComplianceOperatorInfo(ctx, operatorInfo)
}

func (s *pipelineImpl) OnFinish(_ string) {}
