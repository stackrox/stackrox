package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// ScannerV4ReIndex enables Scanner V4 manifest re-indexing.
var ScannerV4ReIndex = registerFeature("Scanner V4 will re-index and delete unused manifests", "ROX_SCANNER_V4_REINDEX", enabled)
