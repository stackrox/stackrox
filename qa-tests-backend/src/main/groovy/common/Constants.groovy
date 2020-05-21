package common

class Constants {
    static final ORCHESTRATOR_NAMESPACE = "qa"
    static final STACKROX_NAMESPACE = "stackrox"
    static final SCHEDULES_SUPPORTED = false
    static final CHECK_CVES_IN_COMPLIANCE = false
    static final RUN_FLAKEY_TESTS = false
    static final EMAIL_NOTIFER_FROM = "stackrox"
    static final EMAIL_NOTIFER_SENDER = "${UUID.randomUUID()}@stackrox.com"
    static final EMAIL_NOTIFER_FULL_FROM = "${EMAIL_NOTIFER_FROM} <${EMAIL_NOTIFER_SENDER}>"
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
    static final VIOLATIONS_WHITELIST = [
            // TODO(ROX-2659) Remove the fixable CVSS one from here, that's not okay.
            "monitoring" : ["CVSS >= 7", "Ubuntu Package Manager in Image", "Curl in Image", "Fixable CVSS >= 7"],
            "scanner" : ["Red Hat Package Manager Execution", "Red Hat Package Manager in Image", "Curl in Image"],
            "collector": ["Ubuntu Package Manager in Image"],
            "authorization-plugin" : ["Latest tag", "90-Day Image Age"],
            "webhookserver" : ["90-Day Image Age"],
    ]

    /*
        StackRox Product Feature Flags

        We need to manually maintain this list here
     */
}
