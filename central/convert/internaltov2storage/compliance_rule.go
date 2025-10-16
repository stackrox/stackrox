package internaltov2storage

import (
	"strings"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

const (
	standardsKey          = "policies.open-cluster-management.io/standards"
	controlAnnotationBase = "control.compliance.openshift.io/"
)

// ComplianceOperatorRule converts message from sensor to V2 storage
func ComplianceOperatorRule(sensorData *central.ComplianceOperatorRuleV2, clusterID string) *storage.ComplianceOperatorRuleV2 {
	fixes := make([]*storage.ComplianceOperatorRuleV2_Fix, 0, len(sensorData.GetFixes()))
	for _, fix := range sensorData.GetFixes() {
		cf := &storage.ComplianceOperatorRuleV2_Fix{}
		cf.SetPlatform(fix.GetPlatform())
		cf.SetDisruption(fix.GetDisruption())
		fixes = append(fixes, cf)
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
		controlAnnotationValues := strings.Split(sensorData.GetAnnotations()[controlAnnotationBase+standard], ";")

		// Add a control entry for each Control + Standard. This data is intentionally denormalized for easier querying.
		for _, controlValue := range controlAnnotationValues {
			rc := &storage.RuleControls{}
			rc.SetStandard(standard)
			rc.SetControl(controlValue)
			controls = append(controls, rc)
		}
	}

	parentRule := sensorData.GetAnnotations()[v1alpha1.RuleIDAnnotationKey]

	corv2 := &storage.ComplianceOperatorRuleV2{}
	corv2.SetId(sensorData.GetId())
	corv2.SetRuleId(sensorData.GetRuleId())
	corv2.SetName(sensorData.GetName())
	corv2.SetRuleType(sensorData.GetRuleType())
	corv2.SetSeverity(severityToV2[sensorData.GetSeverity()])
	corv2.SetLabels(sensorData.GetLabels())
	corv2.SetAnnotations(sensorData.GetAnnotations())
	corv2.SetTitle(sensorData.GetTitle())
	corv2.SetDescription(sensorData.GetDescription())
	corv2.SetRationale(sensorData.GetRationale())
	corv2.SetFixes(fixes)
	corv2.SetWarning(sensorData.GetWarning())
	corv2.SetControls(controls)
	corv2.SetClusterId(clusterID)
	corv2.SetRuleRefId(BuildNameRefID(clusterID, parentRule))
	corv2.SetInstructions(sensorData.GetInstructions())
	corv2.SetParentRule(parentRule)
	return corv2
}
