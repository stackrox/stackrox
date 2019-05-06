package common

class Constants {
    static final ORCHESTRATOR_NAMESPACE = "qa"
    static final SCHEDULES_SUPPORTED = false
    static final CHECK_CVES_IN_COMPLIANCE = false
    static final RUN_FLAKEY_TESTS = false
    static final RUN_PROCESS_WHITELIST_TESTS = true
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
            "Monitoring" : ["CVSS >= 7"],
            "clairify" : ["Red Hat Package Manager Execution"],
    ]
}
