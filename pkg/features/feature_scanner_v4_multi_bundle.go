package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// ScannerV4MultiBundle enables Scanner V4 to consume vulnerabilities using multi-bundle archives.
var ScannerV4MultiBundle = registerFeature("Enables Scanner V4 to consume vulnerabilities using multi-bundle archives", "ROX_SCANNER_V4_MULTI_BUNDLE", enabled)
