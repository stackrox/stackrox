package testutils

import (
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// RuleUID -- rule UID used in test objects
	RuleUID = uuid.NewV4().String()

	ruleID = uuid.NewV4().String()
)

// GetRuleV2SensorMsg -- returns a V2 message from sensor
func GetRuleV2SensorMsg(_ *testing.T) *central.ComplianceOperatorRuleV2 {
	cf := &central.ComplianceOperatorRuleV2_Fix{}
	cf.SetPlatform("openshift")
	cf.SetDisruption("its broken")
	fixes := []*central.ComplianceOperatorRuleV2_Fix{
		cf,
	}

	corv2 := &central.ComplianceOperatorRuleV2{}
	corv2.SetRuleId(ruleID)
	corv2.SetId(RuleUID)
	corv2.SetName("ocp-cis")
	corv2.SetRuleType("node")
	corv2.SetSeverity(0)
	corv2.SetLabels(map[string]string{v1alpha1.SuiteLabel: "ocp-cis"})
	corv2.SetAnnotations(getAnnotations())
	corv2.SetTitle("test rule")
	corv2.SetDescription("test description")
	corv2.SetRationale("test rationale")
	corv2.SetFixes(fixes)
	corv2.SetWarning("test warning")
	return corv2
}

// GetRuleV1Storage -- returns V1 storage scan object
func GetRuleV1Storage(_ *testing.T) *storage.ComplianceOperatorRule {
	cor := &storage.ComplianceOperatorRule{}
	cor.SetId(RuleUID)
	cor.SetRuleId(ruleID)
	cor.SetName("ocp-cis")
	cor.SetClusterId(fixtureconsts.Cluster1)
	cor.SetLabels(map[string]string{v1alpha1.SuiteLabel: "ocp-cis"})
	cor.SetAnnotations(getAnnotations())
	cor.SetTitle("test rule")
	cor.SetDescription("test description")
	cor.SetRationale("test rationale")
	return cor
}

// GetRuleV2Storage -- returns V2 storage rule
func GetRuleV2Storage(_ *testing.T) *storage.ComplianceOperatorRuleV2 {
	cf := &storage.ComplianceOperatorRuleV2_Fix{}
	cf.SetPlatform("openshift")
	cf.SetDisruption("its broken")
	fixes := []*storage.ComplianceOperatorRuleV2_Fix{
		cf,
	}

	controls := []*storage.RuleControls{
		storage.RuleControls_builder{
			Standard: "NERC-CIP",
			Control:  "CIP-003-8 R6",
		}.Build(),
		storage.RuleControls_builder{
			Standard: "NERC-CIP",
			Control:  "CIP-004-6 R3",
		}.Build(),
		storage.RuleControls_builder{
			Standard: "NERC-CIP",
			Control:  "CIP-007-3 R6.1",
		}.Build(),
		storage.RuleControls_builder{
			Standard: "PCI-DSS",
			Control:  "Req-2.2",
		}.Build(),
	}
	corv2 := &storage.ComplianceOperatorRuleV2{}
	corv2.SetId(RuleUID)
	corv2.SetRuleId(ruleID)
	corv2.SetName("ocp-cis")
	corv2.SetRuleType("node")
	corv2.SetLabels(map[string]string{v1alpha1.SuiteLabel: "ocp-cis"})
	corv2.SetAnnotations(getAnnotations())
	corv2.SetTitle("test rule")
	corv2.SetDescription("test description")
	corv2.SetRationale("test rationale")
	corv2.SetFixes(fixes)
	corv2.SetWarning("test warning")
	corv2.SetControls(controls)
	corv2.SetClusterId(fixtureconsts.Cluster1)
	corv2.SetRuleRefId(internaltov2storage.BuildNameRefID(fixtureconsts.Cluster1, "random-rule-name"))
	corv2.SetParentRule("random-rule-name")
	return corv2
}

// GetRuleV2 -- returns V2 storage rule
func GetRuleV2(_ *testing.T) *apiV2.ComplianceRule {
	cf := &apiV2.ComplianceRule_Fix{}
	cf.SetPlatform("openshift")
	cf.SetDisruption("its broken")
	fixes := []*apiV2.ComplianceRule_Fix{
		cf,
	}

	cr := &apiV2.ComplianceRule{}
	cr.SetId(RuleUID)
	cr.SetRuleId(ruleID)
	cr.SetName("ocp-cis")
	cr.SetRuleType("node")
	cr.SetTitle("test rule")
	cr.SetDescription("test description")
	cr.SetRationale("test rationale")
	cr.SetWarning("test warning")
	cr.SetSeverity("UNSET_RULE_SEVERITY")
	cr.SetFixes(fixes)
	cr.SetParentRule("random-rule-name")
	return cr
}

func getAnnotations() map[string]string {
	annotations := make(map[string]string, 3)
	annotations["policies.open-cluster-management.io/standards"] = "NERC-CIP,PCI-DSS"
	annotations["control.compliance.openshift.io/NERC-CIP"] = "CIP-003-8 R6;CIP-004-6 R3;CIP-007-3 R6.1"
	annotations["control.compliance.openshift.io/PCI-DSS"] = "Req-2.2"
	annotations["compliance.openshift.io/rule"] = "random-rule-name"

	return annotations
}
