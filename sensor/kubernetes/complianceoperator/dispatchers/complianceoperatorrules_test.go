package dispatchers

import (
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
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
	rule := testRule()
	dispatcher := NewRulesDispatcher()
	event := dispatcher.ProcessEvent(ruleToUnstructured(t, rule), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1)

	v2Rule := event.ForwardMessages[0].GetComplianceOperatorRuleV2()
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
