package dispatchers

// Constants inlined from github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1
// to avoid importing the v1alpha1 package, which transitively pulls in cel-go (21+ packages).

const (
	// ComplianceScanLabel serves as an indicator for which ComplianceScan owns the referenced object.
	complianceScanLabel = "compliance.openshift.io/scan-name"

	// SuiteLabel indicates that an object belongs to a certain ComplianceSuite.
	suiteLabel = "compliance.openshift.io/suite"

	// CustomRuleProfileAnnotation specifies that a TailoredProfile contains CustomRules.
	customRuleProfileAnnotation = "compliance.openshift.io/tailored-profile-contains-custom-rules"

	// RemediationEnforcementTypeAnnotation specifies the policy enforcement type for a remediation.
	remediationEnforcementTypeAnnotation = "compliance.openshift.io/enforcement-type"

	// Compliance check status values.
	checkResultPass          = "PASS"
	checkResultFail          = "FAIL"
	checkResultInfo          = "INFO"
	checkResultManual        = "MANUAL"
	checkResultError         = "ERROR"
	checkResultNotApplicable = "NOT-APPLICABLE"
	checkResultInconsistent  = "INCONSISTENT"

	// Compliance check result severity values.
	checkResultSeverityUnknown = "unknown"
	checkResultSeverityInfo    = "info"
	checkResultSeverityLow     = "low"
	checkResultSeverityMedium  = "medium"
	checkResultSeverityHigh    = "high"

	// Remediation application states used by isRemediationApplied.
	remediationApplied             = "Applied"
	remediationOutdated            = "Outdated"
	remediationMissingDependencies = "MissingDependencies"
)
