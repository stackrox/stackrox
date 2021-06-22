package complianceoperatorrules

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/rules/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
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

	err := s.datastore.Walk(ctx, func(rule *storage.ComplianceOperatorRule) error {
		if rule.GetClusterId() == clusterID {
			existingIDs.Add(rule.GetId())
		}
		return nil
	})
	if err != nil {
		return err
	}
	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorRule)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorrules", func(id string) error {
		return s.datastore.Delete(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorRule() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorRule)

	event := msg.GetEvent()
	rule := event.GetComplianceOperatorRule()
	rule.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.datastore.Delete(ctx, rule.GetId())
	default:
		return s.datastore.Upsert(ctx, rule)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
