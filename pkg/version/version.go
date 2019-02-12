package version

var (
	mainVersion      string
	collectorVersion string
	scannerVersion   string
)

// GetMainVersion returns the tag of Prevent
func GetMainVersion() string {
	return mainVersion
}

// GetCollectorVersion returns the current collector tag
func GetCollectorVersion() string {
	return collectorVersion
}

// GetScannerVersion returns the current scanner tag
func GetScannerVersion() string {
	return scannerVersion
}
