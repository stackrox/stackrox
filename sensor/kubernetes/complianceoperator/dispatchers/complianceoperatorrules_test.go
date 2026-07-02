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
	t.Helper()
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rule)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: obj}
}

func testRule() *v1alpha1.Rule {
	return &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp4-api-server-anonymous-auth",
			Namespace: "openshift-compliance",
			UID:       "rule-uid",
			Labels:    map[string]string{"app": "compliance"},
			Annotations: map[string]string{
				"note": "test annotation",
			},
		},
		RulePayload: v1alpha1.RulePayload{
			ID:        "xccdf_org.ssgproject.content_rule_api_server_anonymous_auth",
			Title:     "Ensure that anonymous requests are authorized",
			Severity:  "medium",
			CheckType: "Platform",
		},
	}
}

func TestRuleProcessEvent_V2HasOperatorKindRule(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	rule := testRule()
	dispatcher := NewRulesDispatcher()
	event := dispatcher.ProcessEvent(ruleToUnstructured(t, rule), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 2, "expected V1 + V2 events")

	v2Rule := event.ForwardMessages[1].GetComplianceOperatorRuleV2()
	require.NotNil(t, v2Rule)
	assert.Equal(t, string(rule.GetUID()), v2Rule.GetId())
	assert.Equal(t, rule.GetName(), v2Rule.GetName())
	assert.Equal(t, rule.ID, v2Rule.GetRuleId())
	assert.Equal(t, rule.Title, v2Rule.GetTitle())
	assert.Equal(t, rule.Description, v2Rule.GetDescription())
	assert.Equal(t, rule.CheckType, v2Rule.GetRuleType())
	assert.Equal(t, central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY, v2Rule.GetSeverity())
	assert.Equal(t, central.ComplianceOperatorRuleV2_RULE, v2Rule.GetOperatorKind())
}

func TestRuleProcessEvent_WithoutV2Capability(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{})
	t.Cleanup(func() { centralcaps.Set(nil) })

	rule := testRule()
	dispatcher := NewRulesDispatcher()
	event := dispatcher.ProcessEvent(ruleToUnstructured(t, rule), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1, "expected V1 event only")
	assert.IsType(t, &central.SensorEvent_ComplianceOperatorRule{}, event.ForwardMessages[0].GetResource())

	v1Rule := event.ForwardMessages[0].GetComplianceOperatorRule()
	require.NotNil(t, v1Rule)
	assert.Equal(t, string(rule.GetUID()), v1Rule.GetId())
	assert.Equal(t, rule.ID, v1Rule.GetRuleId())
	assert.Equal(t, rule.GetName(), v1Rule.GetName())
	assert.Equal(t, rule.Title, v1Rule.GetTitle())
}

func TestRuleProcessEvent_CelFieldsFromUnstructured(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	rule := testRule()
	obj := ruleToUnstructured(t, rule)

	// Simulate a CO >= 1.9.0 Rule that has CEL fields in spec (not in the v1.8.2 Go struct).
	spec, _ := obj.Object["spec"].(map[string]interface{})
	if spec == nil {
		spec = make(map[string]interface{})
		obj.Object["spec"] = spec
	}
	spec["scannerType"] = "CEL"
	spec["expression"] = `input.node.metadata.labels["secure"] == "true"`
	spec["failureReason"] = "Node is not labeled secure"
	spec["inputs"] = []interface{}{
		map[string]interface{}{
			"name": "node",
			"kubernetesInputSpec": map[string]interface{}{
				"apiVersion": "v1",
				"resource":   "nodes",
				"group":      "",
			},
		},
	}

	dispatcher := NewRulesDispatcher()
	event := dispatcher.ProcessEvent(obj, nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 2)

	v2Rule := event.ForwardMessages[1].GetComplianceOperatorRuleV2()
	require.NotNil(t, v2Rule)
	assert.Equal(t, "CEL", v2Rule.GetScannerType())
	assert.Equal(t, `input.node.metadata.labels["secure"] == "true"`, v2Rule.GetExpression())
	assert.Equal(t, "Node is not labeled secure", v2Rule.GetFailureReason())
	require.Len(t, v2Rule.GetInputs(), 1)
	assert.Equal(t, "node", v2Rule.GetInputs()[0].GetName())
	assert.Equal(t, "v1", v2Rule.GetInputs()[0].GetApiVersion())
	assert.Equal(t, "nodes", v2Rule.GetInputs()[0].GetResource())

	assert.Nil(t, v2Rule.GetCustomRuleDetails())
}

func TestRuleProcessEvent_NoCelFields(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	rule := testRule()
	dispatcher := NewRulesDispatcher()
	event := dispatcher.ProcessEvent(ruleToUnstructured(t, rule), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	v2Rule := event.ForwardMessages[1].GetComplianceOperatorRuleV2()
	require.NotNil(t, v2Rule)
	assert.Empty(t, v2Rule.GetScannerType())
	assert.Empty(t, v2Rule.GetExpression())
	assert.Empty(t, v2Rule.GetFailureReason())
	assert.Empty(t, v2Rule.GetInputs())
	assert.Nil(t, v2Rule.GetCustomRuleDetails())
}
