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

	// CustomRuleUID is the UID used for custom rule test objects.
	CustomRuleUID = uuid.NewV4().String()

	customRuleID = uuid.NewV4().String()
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
		RuleId:       ruleID,
		Id:           RuleUID,
		Name:         "ocp-cis",
		RuleType:     "node",
		Severity:     0,
		Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations:  getAnnotations(),
		Title:        "test rule",
		Description:  "test description",
		Rationale:    "test rationale",
		Fixes:        fixes,
		Warning:      "test warning",
		OperatorKind: central.ComplianceOperatorRuleV2_RULE,
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
		Id:           RuleUID,
		RuleId:       ruleID,
		Name:         "ocp-cis",
		RuleType:     "node",
		Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
		Annotations:  getAnnotations(),
		Title:        "test rule",
		Description:  "test description",
		Rationale:    "test rationale",
		Fixes:        fixes,
		Warning:      "test warning",
		Controls:     controls,
		ClusterId:    fixtureconsts.Cluster1,
		RuleRefId:    internaltov2storage.BuildNameRefID(fixtureconsts.Cluster1, "random-rule-name"),
		ParentRule:   "random-rule-name",
		OperatorKind: storage.ComplianceOperatorRuleV2_RULE,
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
		Id:           RuleUID,
		RuleId:       ruleID,
		Name:         "ocp-cis",
		RuleType:     "node",
		Title:        "test rule",
		Description:  "test description",
		Rationale:    "test rationale",
		Warning:      "test warning",
		Severity:     "UNSET_RULE_SEVERITY",
		Fixes:        fixes,
		ParentRule:   "random-rule-name",
		OperatorKind: apiV2.ComplianceRule_RULE,
	}
}

// GetCustomRuleV2SensorMsg returns a V2 sensor message for a custom rule with CEL fields.
func GetCustomRuleV2SensorMsg(_ *testing.T) *central.ComplianceOperatorRuleV2 {
	return &central.ComplianceOperatorRuleV2{
		RuleId:        customRuleID,
		Id:            CustomRuleUID,
		Name:          "check-cm-marker",
		RuleType:      "Platform",
		Severity:      central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY,
		Labels:        map[string]string{v1alpha1.SuiteLabel: "custom-suite"},
		Annotations:   map[string]string{},
		Title:         "Check CM Marker",
		Description:   "Checks that a configmap marker exists",
		OperatorKind:  central.ComplianceOperatorRuleV2_CUSTOM_RULE,
		ScannerType:   "CEL",
		Expression:    `input.configmap.data["marker"] == "present"`,
		FailureReason: "ConfigMap marker not present",
		Inputs: []*central.ComplianceOperatorCelInput{
			{
				Name:              "configmap",
				ApiVersion:        "v1",
				Resource:          "configmaps",
				ResourceNamespace: "default",
				ResourceName:      "test-cm",
			},
		},
		CustomRuleDetails: &central.ComplianceOperatorRuleV2_CustomRuleDetails{
			Phase: "Ready",
		},
	}
}

// GetCustomRuleV2Storage returns a V2 storage custom rule with CEL fields.
func GetCustomRuleV2Storage(_ *testing.T) *storage.ComplianceOperatorRuleV2 {
	return &storage.ComplianceOperatorRuleV2{
		Id:            CustomRuleUID,
		RuleId:        customRuleID,
		Name:          "check-cm-marker",
		RuleType:      "Platform",
		Severity:      storage.RuleSeverity_HIGH_RULE_SEVERITY,
		Labels:        map[string]string{v1alpha1.SuiteLabel: "custom-suite"},
		Annotations:   map[string]string{},
		Title:         "Check CM Marker",
		Description:   "Checks that a configmap marker exists",
		ClusterId:     fixtureconsts.Cluster1,
		ParentRule:    customRuleID,
		RuleRefId:     internaltov2storage.BuildNameRefID(fixtureconsts.Cluster1, customRuleID),
		OperatorKind:  storage.ComplianceOperatorRuleV2_CUSTOM_RULE,
		ScannerType:   "CEL",
		Expression:    `input.configmap.data["marker"] == "present"`,
		FailureReason: "ConfigMap marker not present",
		Inputs: []*storage.ComplianceOperatorCelInput{
			{
				Name:              "configmap",
				ApiVersion:        "v1",
				Resource:          "configmaps",
				ResourceNamespace: "default",
				ResourceName:      "test-cm",
			},
		},
		CustomRuleDetails: &storage.ComplianceOperatorRuleV2_CustomRuleDetails{
			Phase: "Ready",
		},
	}
}

// GetCustomRuleV2 returns a V2 API custom rule with CEL fields.
func GetCustomRuleV2(_ *testing.T) *apiV2.ComplianceRule {
	return &apiV2.ComplianceRule{
		Id:            CustomRuleUID,
		RuleId:        customRuleID,
		Name:          "check-cm-marker",
		RuleType:      "Platform",
		Severity:      "HIGH_RULE_SEVERITY",
		Title:         "Check CM Marker",
		Description:   "Checks that a configmap marker exists",
		OperatorKind:  apiV2.ComplianceRule_CUSTOM_RULE,
		ScannerType:   "CEL",
		Expression:    `input.configmap.data["marker"] == "present"`,
		FailureReason: "ConfigMap marker not present",
		Inputs: []*apiV2.ComplianceRule_CelInput{
			{
				Name:              "configmap",
				ApiVersion:        "v1",
				Resource:          "configmaps",
				ResourceNamespace: "default",
				ResourceName:      "test-cm",
			},
		},
		CustomRuleDetails: &apiV2.ComplianceRule_CustomRuleDetails{
			Phase: "Ready",
		},
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
