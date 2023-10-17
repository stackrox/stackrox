package common

class Constants {
    static final String ORCHESTRATOR_NAMESPACE = "qa"
    static final String STACKROX_NAMESPACE = "stackrox"
    static final boolean SCHEDULES_SUPPORTED = false
    static final boolean CHECK_CVES_IN_COMPLIANCE = false
    static final boolean RUN_FLAKEY_TESTS = false
    static final String EMAIL_NOTIFER_FROM = "stackrox"
    static final String EMAIL_NOTIFER_SENDER = "${UUID.randomUUID()}@stackrox.com"
    static final String EMAIL_NOTIFER_FULL_FROM = "${EMAIL_NOTIFER_FROM} <${EMAIL_NOTIFER_SENDER}>"
    static final String EMAIL_NOTIFIER_RECIPIENT = "stackrox.qa@gmail.com"
    static final FAILURE_DEBUG_LIMIT = 10
    static final String AUTO_REGISTERED_STACKROX_SCANNER_INTEGRATION = "Stackrox Scanner"
    static final String ANY_FIXED_VULN_POLICY = "any-fixed-vulnerabilities"
    static final Map<String, String> CSV_COLUMN_MAPPING = [
            "Standard" : "standard",
            "Cluster" : "cluster",
            "Namespace" : "namespace",
            "Object Type" : "objectType",
            "Object Name" : "objectName",
            "Control" : "control",
            "Control Description" : "controlDescription",
            "State" : "state",
            "Evidence" : "evidence",
            "Assessment Time" : "timestamp",
    ]
    static final VIOLATIONS_ALLOWLIST = [
            // TODO(ROX-2659) Remove the fixable CVSS one from here, that's not okay.
            "monitoring" : ["CVSS >= 7", "Ubuntu Package Manager in Image", "Curl in Image", "Fixable CVSS >= 7",
                            ANY_FIXED_VULN_POLICY, "90-Day Image Age", "Fixable Severity at least Important"],
            "scanner" : ["Red Hat Package Manager Execution", "Red Hat Package Manager in Image", "Curl in Image"],
            "collector": ["Ubuntu Package Manager in Image"],
            "authorization-plugin" : ["Latest tag", "90-Day Image Age"],
            "webhookserver" : ["90-Day Image Age"],
    ]
    static final VIOLATIONS_BY_POLICY_ALLOWLIST = [
            "OpenShift: Advanced Cluster Security Central Admin Secret Accessed"
    ]
    static final INTERNET_EXTERNAL_SOURCE_ID = "afa12424-bde3-4313-b810-bb463cbe8f90" // pkg/networkgraph/constants.go
    static final STACKROX_ANNOTATION_TRUNCATION_LENGTH = 254
    static final CORE_IMAGE_INTEGRATION_NAME = "core quay"
    // Padding required for test feature timeouts. This is used to configure the
    // globalTimeout for long running tests. The value takes into consideration
    // common functionality that occurs for all tests such as debug gathering when
    // tests fail.
    static final TEST_FEATURE_TIMEOUT_PAD = 360
    static final String SPLUNK_TEST_NAMESPACE = "qa-splunk"

    /*
        StackRox Product Feature Flags

        We need to manually maintain this list here
     */
}
