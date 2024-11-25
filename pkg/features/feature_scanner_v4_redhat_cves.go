package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// ScannerV4RedHatCVEs enables displaying CVEs instead of RHSAs/RHEAs/RHBAs in the place of fixed vulnerabilities affected Red Hat products.
// TODO(ROX-26672): Remove this once we can show both CVEs and RHSAs in the UI + reports.
var ScannerV4RedHatCVEs = registerFeature("Scanner V4 will output CVEs instead of RHSAs/RHBAs/RHEAs for fixed Red Hat vulnerabilities", "ROX_SCANNER_V4_RED_HAT_CVES")
