package complianceoperatorrules

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/manager"
	"github.com/stackrox/rox/central/complianceoperator/rules/datastore"
	v2Datastore "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	"github.com/stackrox/rox/central/convert/internaltov1storage"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	if features.ComplianceEnhancements.Enabled() {
		return NewPipeline(datastore.Singleton(), manager.Singleton(), v2Datastore.Singleton())
	}
	return NewPipeline(datastore.Singleton(), manager.Singleton(), nil)
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore, manager manager.Manager, v2RuleDatastore v2Datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		datastore:       datastore,
		manager:         manager,
		v2RuleDatastore: v2RuleDatastore,
	}
}

type pipelineImpl struct {
	datastore       datastore.DataStore
	manager         manager.Manager
	v2RuleDatastore v2Datastore.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	existingIDs := set.NewStringSet()
	walkFn := func() error {
		existingIDs.Clear()
		return s.datastore.Walk(ctx, func(rule *storage.ComplianceOperatorRule) error {
			if rule.GetClusterId() == clusterID {
				existingIDs.Add(rule.GetId())
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return err
	}

	// For now if nextgen compliance is enabled, we have to reconcile both versions of compliance.
	if features.ComplianceEnhancements.Enabled() {
		rules, err := s.v2RuleDatastore.GetRulesByCluster(ctx, clusterID)
		if err != nil {
			return err
		}

		for _, rule := range rules {
			// The UID is used for reconciliation
			existingIDs.Add(rule.GetId())
		}
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorRule)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorrules", func(id string) error {
		if features.ComplianceEnhancements.Enabled() {
			if err := s.v2RuleDatastore.DeleteRule(ctx, id); err != nil {
				return err
			}
		}

		return s.datastore.Delete(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	if features.ComplianceEnhancements.Enabled() {
		return msg.GetEvent().GetComplianceOperatorRuleV2() != nil || msg.GetEvent().GetComplianceOperatorRule() != nil
	}
	return msg.GetEvent().GetComplianceOperatorRule() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorRule)

	event := msg.GetEvent()
	// If a sensor sends in a v1 compliance message we will still process it the v1 way in the event
	// a sensor is not updated.
	switch event.Resource.(type) {
	case *central.SensorEvent_ComplianceOperatorRule:
		return s.processComplianceRule(ctx, event, clusterID)
	case *central.SensorEvent_ComplianceOperatorRuleV2:
		if !features.ComplianceEnhancements.Enabled() {
			return errors.New("Next gen compliance is disabled.  Message unexpected.")
		}
		return s.processComplianceRuleV2(ctx, event, clusterID)
	}

	return errors.Errorf("unexpected message %t.", event.Resource)
}

func (s *pipelineImpl) processComplianceRule(_ context.Context, event *central.SensorEvent, clusterID string) error {
	rule := event.GetComplianceOperatorRule()
	rule.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.manager.DeleteRule(rule)
	default:
		return s.manager.AddRule(rule)
	}
}

func (s *pipelineImpl) processComplianceRuleV2(ctx context.Context, event *central.SensorEvent, clusterID string) error {
	if !features.ComplianceEnhancements.Enabled() {
		return errors.New("Next gen compliance is disabled.  Message unexpected.")
	}

	rule := event.GetComplianceOperatorRuleV2()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		// For now, we need to process V1 rules as well to ensure we retain the full capability of V1 compliance
		if err := s.manager.DeleteRule(internaltov1storage.ComplianceOperatorRule(rule, clusterID)); err != nil {
			return err
		}
		return s.v2RuleDatastore.DeleteRule(ctx, rule.GetId())
	default:
		// For now, we need to process V1 rules as well to ensure we retain the full capability of V1 compliance
		if err := s.manager.AddRule(internaltov1storage.ComplianceOperatorRule(rule, clusterID)); err != nil {
			return err
		}

		return s.v2RuleDatastore.UpsertRule(ctx, internaltov2storage.ComplianceOperatorRule(rule, clusterID))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
