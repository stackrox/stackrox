package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// VulnMgmtLegacySnooze enables APIs and UI for the legacy VM 1.0 "snooze CVE" functionality in the new VM 2.0 sections
var VulnMgmtLegacySnooze = registerFeature("Enables the ability to snooze Node and Platform CVEs in VM 2.0", "ROX_VULN_MGMT_LEGACY_SNOOZE")
