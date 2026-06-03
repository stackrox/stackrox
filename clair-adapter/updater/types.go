package updater

// BundleData represents unpacked vulnerability data ready to serve to Clair.
type BundleData struct {
	Name        string // e.g., "alpine", "nvd", "rhel-vex"
	Data        []byte // Raw content (zstd-compressed JSON)
	Fingerprint string // ETag/fingerprint for conditional serving
}
