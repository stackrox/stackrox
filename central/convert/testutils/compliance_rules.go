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
	fixes := []*central.ComplianceOperatorRuleV2_Fix{
		{
			Platform:   "openshift",
			Disruption: "its broken",
		},
	}

	return &central.ComplianceOperatorRuleV2{
		RuleId:      ruleID,
		Id:          RuleUID,
		Name:        "ocp-cis",
		RuleType:    "node",
		Severity:    0,
		Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations: getAnnotations(),
		Title:       "test rule",
		Description: "test description",
		Rationale:   "test rationale",
		Fixes:       fixes,
		Warning:     "test warning",
	}
}

// GetRuleV1Storage -- returns V1 storage scan object
func GetRuleV1Storage(_ *testing.T) *storage.ComplianceOperatorRule {
	return &storage.ComplianceOperatorRule{
		Id:          RuleUID,
		RuleId:      ruleID,
		Name:        "ocp-cis",
		ClusterId:   fixtureconsts.Cluster1,
		Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations: getAnnotations(),
		Title:       "test rule",
		Description: "test description",
		Rationale:   "test rationale",
	}
}

// GetRuleV2Storage -- returns V2 storage rule
func GetRuleV2Storage(_ *testing.T) *storage.ComplianceOperatorRuleV2 {
	fixes := []*storage.ComplianceOperatorRuleV2_Fix{
		{
			Platform:   "openshift",
			Disruption: "its broken",
		},
	}

	controls := []*storage.RuleControls{
		{
			Standard: "NERC-CIP",
			Control:  "CIP-003-8 R6",
		},
		{
			Standard: "NERC-CIP",
			Control:  "CIP-004-6 R3",
		},
		{
			Standard: "NERC-CIP",
			Control:  "CIP-007-3 R6.1",
		},
		{
			Standard: "PCI-DSS",
			Control:  "Req-2.2",
		},
	}
	return &storage.ComplianceOperatorRuleV2{
		Id:          RuleUID,
		RuleId:      ruleID,
		Name:        "ocp-cis",
		RuleType:    "node",
		Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations: getAnnotations(),
		Title:       "test rule",
		Description: "test description",
		Rationale:   "test rationale",
		Fixes:       fixes,
		Warning:     "test warning",
		Controls:    controls,
		ClusterId:   fixtureconsts.Cluster1,
		RuleRefId:   internaltov2storage.BuildNameRefID(fixtureconsts.Cluster1, "random-rule-name"),
		ParentRule:  "random-rule-name",
	}
}

// GetRuleV2 -- returns V2 storage rule
func GetRuleV2(_ *testing.T) *apiV2.ComplianceRule {
	fixes := []*apiV2.ComplianceRule_Fix{
		{
			Platform:   "openshift",
			Disruption: "its broken",
		},
	}

	return &apiV2.ComplianceRule{
		Id:          RuleUID,
		RuleId:      ruleID,
		Name:        "ocp-cis",
		RuleType:    "node",
		Title:       "test rule",
		Description: "test description",
		Rationale:   "test rationale",
		Warning:     "test warning",
		Severity:    "UNSET_RULE_SEVERITY",
		Fixes:       fixes,
		ParentRule:  "random-rule-name",
	}
}

func getAnnotations() map[string]string {
	annotations := make(map[string]string, 3)
	annotations["policies.open-cluster-management.io/standards"] = "NERC-CIP,PCI-DSS"
	annotations["control.compliance.openshift.io/NERC-CIP"] = "CIP-003-8 R6;CIP-004-6 R3;CIP-007-3 R6.1"
	annotations["control.compliance.openshift.io/PCI-DSS"] = "Req-2.2"
	annotations["compliance.openshift.io/rule"] = "random-rule-name"

	return annotations
}
