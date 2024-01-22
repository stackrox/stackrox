package internaltov2storage

import (
	"strings"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

const (
	standardsKey = "policies.open-cluster-management.io/standards"

	controlAnnotationBase = "control.compliance.openshift.io/"
)

// ComplianceOperatorRule converts message from sensor to V2 storage
func ComplianceOperatorRule(sensorData *central.ComplianceOperatorRuleV2, clusterID string) *storage.ComplianceOperatorRuleV2 {
	fixes := make([]*storage.ComplianceOperatorRuleV2_Fix, 0, len(sensorData.Fixes))
	for _, fix := range sensorData.Fixes {
		fixes = append(fixes, &storage.ComplianceOperatorRuleV2_Fix{
			Platform:   fix.GetPlatform(),
			Disruption: fix.GetDisruption(),
		})
	}

	// The standards and controls that a rule applies to are stored within the annotations of the Rule CR.
	// For example:
	/*
		metadata:
		annotations:
			compliance.openshift.io/image-digest: pb-ocp4vgws6
			compliance.openshift.io/rule: kubelet-configure-tls-cert
			control.compliance.openshift.io/CIS-OCP: 4.2.10
			control.compliance.openshift.io/NERC-CIP: CIP-003-8 R4.2;CIP-007-3 R5.1
			control.compliance.openshift.io/NIST-800-53: SC-8;SC-8(1);SC-8(2)
			control.compliance.openshift.io/PCI-DSS: Req-2.2;Req-2.2.3;Req-2.3
			policies.open-cluster-management.io/controls: CIP-003-8 R4.2,CIP-007-3 R5.1,SC-8,SC-8(1),SC-8(2),Req-2.2,Req-2.2.3,Req-2.3,4.2.10
			policies.open-cluster-management.io/standards: NERC-CIP,NIST-800-53,PCI-DSS,CIS-OCP
	*/
	// In order to map standards and controls, we first need to use policies.open-cluster-management.io/standards
	// to get the list of standards.  Then for each standard we can get the contols by using:
	// control.compliance.openshift.io/STANDARD.  This will allow us to track the list of
	// standards and controls a given rule applies to.  This will be important when
	// building reports in the future.
	standards := strings.Split(sensorData.GetAnnotations()[standardsKey], ",")
	controls := make([]*storage.RuleControls, 0, len(standards))
	for _, standard := range standards {
		controls = append(controls, &storage.RuleControls{
			Standard: standard,
			Controls: strings.Split(sensorData.GetAnnotations()[controlAnnotationBase+standard], ";"),
		})
	}

	return &storage.ComplianceOperatorRuleV2{
		Id:          sensorData.GetId(),
		RuleId:      sensorData.GetRuleId(),
		Name:        sensorData.GetName(),
		RuleType:    sensorData.GetRuleType(),
		Severity:    severityToV2[sensorData.GetSeverity()],
		Labels:      sensorData.GetLabels(),
		Annotations: sensorData.GetAnnotations(),
		Title:       sensorData.GetTitle(),
		Description: sensorData.GetDescription(),
		Rationale:   sensorData.GetRationale(),
		Fixes:       fixes,
		Warning:     sensorData.GetWarning(),
		Controls:    controls,
		ClusterId:   clusterID,
	}
}
