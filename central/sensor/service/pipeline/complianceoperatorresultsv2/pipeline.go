package complianceoperatorresultsv2

import (
	"context"

	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(v2.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(v2Datastore v2.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		v2Datastore: v2Datastore,
	}
}

type pipelineImpl struct {
	v2Datastore v2.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	// Not currently deleting anything for purposes of building a result history for trending
	// and look back
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorResultV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorCheckResultV2)

	if !features.ComplianceEnhancements.Enabled() {
		return errors.New("Next gen compliance is disabled.  Message unexpected.")
	}

	event := msg.GetEvent()
	checkResult := event.GetComplianceOperatorResultV2()
	checkResult.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		// use V2 datastore
		return s.v2Datastore.DeleteResult(ctx, event.GetId())
	default:
		return s.v2Datastore.UpsertResult(ctx, internaltov2storage.ComplianceOperatorCheckResult(checkResult, clusterID))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
