package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// VulnMgmtWorkloadCVEs enables APIs and UI pages for the VM Workload CVE enhancements
var VulnMgmtWorkloadCVEs = registerFeature("Vuln Mgmt Workload CVEs", "ROX_VULN_MGMT_WORKLOAD_CVES", enabled, unchangeableInProd)
