package internaltov2storage

import (
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/assert"
)

func Test_idToDNSFriendlyName(t *testing.T) {
	cases := map[string]struct {
		input    string
		expected string
	}{
		"ssgproject rule ID strips prefix": {
			input:    "xccdf_org.ssgproject.content_rule_kubelet_configure_tls_cert",
			expected: "kubelet-configure-tls-cert",
		},
		"ssgproject rule ID without content_ segment does not strip prefix": {
			input:    "xccdf_org.ssgproject.rule_abc",
			expected: "xccdf-org.ssgproject.rule-abc",
		},
		"custom rule ID with non-ssgproject prefix keeps prefix": {
			input:    "xccdf_org.example_rule_check_no_latest_tag",
			expected: "xccdf-org.example-rule-check-no-latest-tag",
		},
		"already lowercase, no underscores": {
			input:    "some-rule",
			expected: "some-rule",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, idToDNSFriendlyName(tc.input))
		})
	}
}

func TestComplianceOperatorRule_ParentRuleAndRuleRefId(t *testing.T) {
	const clusterID = fixtureconsts.Cluster1

	t.Run("regular rule uses annotation for parentRule", func(t *testing.T) {
		msg := &central.ComplianceOperatorRuleV2{
			Id:           "rule-uid",
			RuleId:       "xccdf_org.ssgproject.content_rule_some_other_rule",
			Name:         "kubelet-configure-tls-cert",
			OperatorKind: central.ComplianceOperatorRuleV2_RULE,
			Annotations: map[string]string{
				// The annotation value differs from idToDNSFriendlyName(RuleId) to confirm
				// that parentRule is read from the annotation, not derived from RuleId.
				v1alpha1.RuleIDAnnotationKey: "kubelet-configure-tls-cert",
			},
		}

		result := ComplianceOperatorRule(msg, clusterID)

		assert.Equal(t, "kubelet-configure-tls-cert", result.GetParentRule())
		assert.Equal(t, BuildNameRefID(clusterID, "kubelet-configure-tls-cert"), result.GetRuleRefId())
	})

	t.Run("custom rule derives parentRule from RuleId, ignoring RuleIDAnnotationKey annotation", func(t *testing.T) {
		msg := &central.ComplianceOperatorRuleV2{
			Id:           "custom-rule-uid",
			RuleId:       "xccdf_org.example_rule_check_no_latest_tag",
			Name:         "check-no-latest-tag",
			OperatorKind: central.ComplianceOperatorRuleV2_CUSTOM_RULE,
			Annotations: map[string]string{
				// CustomRules do not have compliance.openshift.io/rule set by CO, but even
				// if one were present it must be ignored: parentRule always comes from RuleId.
				v1alpha1.RuleIDAnnotationKey: "some-ignored-annotation",
			},
		}

		result := ComplianceOperatorRule(msg, clusterID)

		// CO sets compliance.openshift.io/rule on custom rule check results to
		// IDToDNSFriendlyName(Spec.ID). parentRule must equal that value so that
		// BuildNameRefID produces matching RuleRefIds on both sides of the join.
		expectedParentRule := "xccdf-org.example-rule-check-no-latest-tag"
		assert.Equal(t, expectedParentRule, result.GetParentRule())
		assert.Equal(t, BuildNameRefID(clusterID, expectedParentRule), result.GetRuleRefId())
	})
}
