package dispatchers

import (
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func ruleToUnstructured(t *testing.T, rule *v1alpha1.Rule) *unstructured.Unstructured {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rule)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: obj}
}

func TestRuleProcessEvent_V2HasOperatorKindRule(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	rule := &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp4-api-server-anonymous-auth",
			Namespace: "openshift-compliance",
			UID:       "rule-uid",
		},
		RulePayload: v1alpha1.RulePayload{
			ID:        "xccdf_org.ssgproject.content_rule_api_server_anonymous_auth",
			Title:     "Ensure that anonymous requests are authorized",
			Severity:  "medium",
			CheckType: "Platform",
		},
	}

	dispatcher := NewRulesDispatcher()
	event := dispatcher.ProcessEvent(ruleToUnstructured(t, rule), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 2, "expected V1 + V2 events")

	v2Rule := event.ForwardMessages[1].GetComplianceOperatorRuleV2()
	require.NotNil(t, v2Rule)
	assert.Equal(t, central.ComplianceOperatorRuleV2_RULE, v2Rule.GetOperatorKind())
}

func TestRuleProcessEvent_WithoutV2Capability(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{})
	t.Cleanup(func() { centralcaps.Set(nil) })

	rule := &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp4-api-server-anonymous-auth",
			Namespace: "openshift-compliance",
			UID:       "rule-uid",
		},
		RulePayload: v1alpha1.RulePayload{
			ID:        "xccdf_org.ssgproject.content_rule_api_server_anonymous_auth",
			Title:     "Ensure that anonymous requests are authorized",
			Severity:  "medium",
			CheckType: "Platform",
		},
	}

	dispatcher := NewRulesDispatcher()
	event := dispatcher.ProcessEvent(ruleToUnstructured(t, rule), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1, "expected V1 event only")
	assert.IsType(t, &central.SensorEvent_ComplianceOperatorRule{}, event.ForwardMessages[0].GetResource())

	v1Rule := event.ForwardMessages[0].GetComplianceOperatorRule()
	require.NotNil(t, v1Rule)
	assert.Equal(t, "rule-uid", v1Rule.GetId())
	assert.Equal(t, "xccdf_org.ssgproject.content_rule_api_server_anonymous_auth", v1Rule.GetRuleId())
	assert.Equal(t, "ocp4-api-server-anonymous-auth", v1Rule.GetName())
	assert.Equal(t, "Ensure that anonymous requests are authorized", v1Rule.GetTitle())
}
