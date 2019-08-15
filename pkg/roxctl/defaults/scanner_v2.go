package defaults

// ScannerV2PVName returns the default PV name for scanner V2.
func ScannerV2PVName() string {
	return "scanner-v2-db"
}

// ScannerV2PVSize returns the default PV size for scanner v2.
func ScannerV2PVSize() uint32 {
	return 50
}

// ScannerV2HostPath returns the default hostpath location for scanner v2.
func ScannerV2HostPath() string {
	return "/var/lib/stackrox/scanner"
}
