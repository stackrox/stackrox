package complianceoperatorinfo

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
)

const minimalComplianceOperatorVersion = "v1.6.0"

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this compliance operator info pipeline.
func GetPipeline() pipeline.Fragment {
	return NewPipeline(compliancemanager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(manager compliancemanager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		manager: manager,
	}
}

type pipelineImpl struct {
	manager compliancemanager.Manager
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

	if msg.GetComplianceOperatorInfo().GetStatusError() != "" {
		operatorErrors = append(operatorErrors, msg.GetComplianceOperatorInfo().GetStatusError())
	}

	desiredPods := msg.GetComplianceOperatorInfo().GetTotalDesiredPods()
	readyPods := msg.GetComplianceOperatorInfo().GetTotalReadyPods()

	// if not ready, add it to the status errors
	if readyPods < desiredPods {
		operatorErrors = append(operatorErrors, fmt.Sprintf("compliance operator not ready. Only %d pods are ready when %d are desired.", readyPods, desiredPods))
	}

	// we support only newer versions of compliance operator
	minVersion, _ := semver.NewVersion(minimalComplianceOperatorVersion)
	complianceOperatorVersion, err := semver.NewVersion(operatorInfo.GetVersion())
	if complianceOperatorVersion == nil || err != nil {
		operatorErrors = append(operatorErrors, fmt.Sprintf("invalid compliance operator version %q", operatorInfo.GetVersion()))
	} else if complianceOperatorVersion.LessThan(minVersion) {
		operatorErrors = append(operatorErrors, fmt.Sprintf("compliance operator version %q is not supported. Minimal required version is %q", operatorInfo.GetVersion(), minimalComplianceOperatorVersion))
	}

	operatorInfo.StatusErrors = operatorErrors
	operatorInfo.OperatorStatus = storage.COStatus_UNHEALTHY
	if len(operatorInfo.StatusErrors) == 0 {
		operatorInfo.OperatorStatus = storage.COStatus_HEALTHY
	}
	return s.manager.ProcessComplianceOperatorInfo(ctx, operatorInfo)
}

func (s *pipelineImpl) OnFinish(_ string) {}
