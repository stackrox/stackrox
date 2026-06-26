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

func customRuleToUnstructured(t *testing.T, cr *v1alpha1.CustomRule) *unstructured.Unstructured {
	t.Helper()
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: obj}
}

func testCustomRule() *v1alpha1.CustomRule {
	return &v1alpha1.CustomRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "check-cm-marker",
			Namespace: "openshift-compliance",
			UID:       "custom-rule-uid",
			Labels:    map[string]string{"app": "compliance"},
			Annotations: map[string]string{
				"note": "test annotation",
			},
		},
		Spec: v1alpha1.CustomRuleSpec{
			RulePayload: v1alpha1.RulePayload{
				ID:           "xccdf_org.example_rule_check_cm_marker",
				Title:        "Check CM Marker",
				Description:  "Checks that a configmap marker exists",
				Severity:     "high",
				CheckType:    "Platform",
				Instructions: "Ensure the configmap has the marker key",
				AvailableFixes: []v1alpha1.FixDefinition{
					{Platform: "ocp4", Disruption: "low"},
				},
			},
			CustomRulePayload: v1alpha1.CustomRulePayload{
				ScannerType:   v1alpha1.ScannerTypeCEL,
				Expression:    `input.configmap.data["marker"] == "present"`,
				FailureReason: "ConfigMap marker not present",
				Inputs: []v1alpha1.InputPayload{
					{
						Name: "configmap",
						KubernetesInputSpec: v1alpha1.KubernetesInputSpec{
							APIVersion:        "v1",
							Resource:          "configmaps",
							ResourceNamespace: "default",
							ResourceName:      "test-cm",
						},
					},
				},
			},
		},
		Status: v1alpha1.CustomRuleStatus{
			Phase: v1alpha1.CustomRulePhaseReady,
		},
	}
}

func TestCustomRuleProcessEvent_WithCapability(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2TailoredProfiles})
	t.Cleanup(func() { centralcaps.Set(nil) })

	cr := testCustomRule()
	dispatcher := NewCustomRuleDispatcher()
	event := dispatcher.ProcessEvent(customRuleToUnstructured(t, cr), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1)

	rule := event.ForwardMessages[0].GetComplianceOperatorRuleV2()
	require.NotNil(t, rule)

	assert.Equal(t, string(cr.GetUID()), rule.GetId())
	assert.Equal(t, cr.GetName(), rule.GetName())
	assert.Equal(t, cr.Spec.ID, rule.GetRuleId())
	assert.Equal(t, cr.Spec.Title, rule.GetTitle())
	assert.Equal(t, cr.Spec.Description, rule.GetDescription())
	assert.Equal(t, cr.Spec.CheckType, rule.GetRuleType())
	assert.Equal(t, central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY, rule.GetSeverity())
	assert.Equal(t, central.ComplianceOperatorRuleV2_CUSTOM_RULE, rule.GetOperatorKind())
	assert.Equal(t, cr.Spec.Instructions, rule.GetInstructions())
	require.Len(t, rule.GetFixes(), 1)
	assert.Equal(t, cr.Spec.AvailableFixes[0].Platform, rule.GetFixes()[0].GetPlatform())

	assert.Equal(t, "CEL", rule.GetScannerType())
	assert.Equal(t, `input.configmap.data["marker"] == "present"`, rule.GetExpression())
	assert.Equal(t, "ConfigMap marker not present", rule.GetFailureReason())
	require.Len(t, rule.GetInputs(), 1)
	assert.Equal(t, "configmap", rule.GetInputs()[0].GetName())
	assert.Equal(t, "v1", rule.GetInputs()[0].GetApiVersion())
	assert.Equal(t, "configmaps", rule.GetInputs()[0].GetResource())
	assert.Equal(t, "default", rule.GetInputs()[0].GetResourceNamespace())
	assert.Equal(t, "test-cm", rule.GetInputs()[0].GetResourceName())

	require.NotNil(t, rule.GetCustomRuleDetails())
	assert.Equal(t, "Ready", rule.GetCustomRuleDetails().GetPhase())
	assert.Empty(t, rule.GetCustomRuleDetails().GetErrorMessage())
}

func TestCustomRuleProcessEvent_ErrorPhase(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2TailoredProfiles})
	t.Cleanup(func() { centralcaps.Set(nil) })

	cr := testCustomRule()
	cr.Status.Phase = v1alpha1.CustomRulePhaseError
	cr.Status.ErrorMessage = "invalid CEL expression"

	dispatcher := NewCustomRuleDispatcher()
	event := dispatcher.ProcessEvent(customRuleToUnstructured(t, cr), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	rule := event.ForwardMessages[0].GetComplianceOperatorRuleV2()
	require.NotNil(t, rule.GetCustomRuleDetails())
	assert.Equal(t, "Error", rule.GetCustomRuleDetails().GetPhase())
	assert.Equal(t, "invalid CEL expression", rule.GetCustomRuleDetails().GetErrorMessage())
}

func TestCustomRuleProcessEvent_WithoutCapability(t *testing.T) {
	cases := map[string][]centralsensor.CentralCapability{
		"only ComplianceV2Integrations": {centralsensor.ComplianceV2Integrations},
		"no capabilities":               {},
	}
	for name, caps := range cases {
		t.Run(name, func(t *testing.T) {
			centralcaps.Set(caps)
			t.Cleanup(func() { centralcaps.Set(nil) })

			dispatcher := NewCustomRuleDispatcher()
			event := dispatcher.ProcessEvent(customRuleToUnstructured(t, testCustomRule()), nil, central.ResourceAction_CREATE_RESOURCE)

			assert.Nil(t, event)
		})
	}
}
