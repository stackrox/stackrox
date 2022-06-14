package complianceoperatorresults

import (
	"context"

	"github.com/stackrox/stackrox/central/complianceoperator/checkresults/datastore"
	countMetrics "github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/set"
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		datastore: datastore,
	}
}

type pipelineImpl struct {
	datastore datastore.DataStore
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	existingIDs := set.NewStringSet()
	err := s.datastore.Walk(ctx, func(check *storage.ComplianceOperatorCheckResult) error {
		if check.GetClusterId() == clusterID {
			existingIDs.Add(check.GetId())
		}
		return nil
	})
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorResult)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorcheckresults", func(id string) error {
		return s.datastore.Delete(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorResult() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorCheckResult)

	event := msg.GetEvent()
	checkResult := event.GetComplianceOperatorResult()
	checkResult.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.datastore.Delete(ctx, event.GetId())
	default:
		return s.datastore.Upsert(ctx, checkResult)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
