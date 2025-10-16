package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RulesDispatcher handles compliance operator rules
type RulesDispatcher struct{}

// NewRulesDispatcher creates and returns a new compliance rule dispatcher.
func NewRulesDispatcher() *RulesDispatcher {
	return &RulesDispatcher{}
}

// ProcessEvent processes a rule event
func (c *RulesDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var complianceRule v1alpha1.Rule

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceRule); err != nil {
		log.Errorf("error converting unstructured to compliance rule: %v", err)
		return nil
	}
	id := string(complianceRule.UID)
	// We are pulling additional data for rules and using the storage object even in an internal api
	// is a bad practice, so we will make that split now.  V1 and V2 compliance will both need to work for a period
	// of time.  However, we should not need to send the same rule twice, the pipeline can convert the V2 sensor message
	// so V1 and V2 objects can both be stored.

	events := []*central.SensorEvent{
		central.SensorEvent_builder{
			Id:     id,
			Action: action,
			ComplianceOperatorRule: storage.ComplianceOperatorRule_builder{
				Id:          id,
				RuleId:      complianceRule.ID,
				Name:        complianceRule.Name,
				Title:       complianceRule.Title,
				Labels:      complianceRule.Labels,
				Annotations: complianceRule.Annotations,
				Description: complianceRule.Description,
				Rationale:   complianceRule.Rationale,
			}.Build(),
		}.Build(),
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		fixes := make([]*central.ComplianceOperatorRuleV2_Fix, 0, len(complianceRule.AvailableFixes))
		for _, r := range complianceRule.AvailableFixes {
			cf := &central.ComplianceOperatorRuleV2_Fix{}
			cf.SetPlatform(r.Platform)
			cf.SetDisruption(r.Disruption)
			fixes = append(fixes, cf)
		}

		corv2 := &central.ComplianceOperatorRuleV2{}
		corv2.SetRuleId(complianceRule.ID)
		corv2.SetId(id)
		corv2.SetName(complianceRule.Name)
		corv2.SetRuleType(complianceRule.CheckType)
		corv2.SetSeverity(ruleSeverityToV2Severity(complianceRule.Severity))
		corv2.SetLabels(complianceRule.Labels)
		corv2.SetAnnotations(complianceRule.Annotations)
		corv2.SetTitle(complianceRule.Title)
		corv2.SetDescription(complianceRule.Description)
		corv2.SetRationale(complianceRule.Rationale)
		corv2.SetFixes(fixes)
		corv2.SetWarning(complianceRule.Warning)
		corv2.SetInstructions(complianceRule.Instructions)
		se := &central.SensorEvent{}
		se.SetId(id)
		se.SetAction(action)
		se.SetComplianceOperatorRuleV2(proto.ValueOrDefault(corv2))
		events = append(events, se)
	}

	return component.NewEvent(events...)
}
