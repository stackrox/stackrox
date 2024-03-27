package complianceoperatorrulesv2

import (
	"context"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/pkg/errors"
	v2Datastore "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(v2Datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(v2RuleDatastore v2Datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		v2RuleDatastore: v2RuleDatastore,
	}
}

type pipelineImpl struct {
	v2RuleDatastore v2Datastore.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	// Nothing to do in this case
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	existingIDs := set.NewStringSet()
	rules, err := s.v2RuleDatastore.GetRulesByCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		// The UID is used for reconciliation
		existingIDs.Add(rule.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorRuleV2)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorrulesv2", func(id string) error {
		return s.v2RuleDatastore.DeleteRule(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorRuleV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorRuleV2)

	if !features.ComplianceEnhancements.Enabled() {
		return errors.New("Next gen compliance is disabled.  Message unexpected.")
	}

	event := msg.GetEvent()
	rule := event.GetComplianceOperatorRuleV2()

	if val := rule.Annotations[v1alpha1.RuleIDAnnotationKey]; val == "" {
		return errors.Errorf("Rule %s is missing the annotation %s", rule.GetName(), v1alpha1.RuleIDAnnotationKey)
	}

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.v2RuleDatastore.DeleteRule(ctx, rule.GetId())
	default:
		return s.v2RuleDatastore.UpsertRule(ctx, internaltov2storage.ComplianceOperatorRule(rule, clusterID))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
