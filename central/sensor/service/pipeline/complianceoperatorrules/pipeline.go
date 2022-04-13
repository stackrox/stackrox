package complianceoperatorrules

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/complianceoperator/manager"
	"github.com/stackrox/stackrox/central/complianceoperator/rules/datastore"
	countMetrics "github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/complianceoperator/api/v1alpha1"
	"github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/set"
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(datastore.Singleton(), manager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore, manager manager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		datastore: datastore,
		manager:   manager,
	}
}

type pipelineImpl struct {
	datastore datastore.DataStore
	manager   manager.Manager
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

	if val := rule.Annotations[v1alpha1.RuleIDAnnotationKey]; val == "" {
		return errors.Errorf("Rule %s is missing the annotation %s", rule.GetName(), v1alpha1.RuleIDAnnotationKey)
	}

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.manager.DeleteRule(rule)
	default:
		return s.manager.AddRule(rule)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
