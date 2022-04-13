package v2backuprestore

import v1 "github.com/stackrox/stackrox/generated/api/v1"

// RestoreBodySize returns the total size of all data specified in the manifest, i.e., the sum over all encoded file
// sizes.
func RestoreBodySize(manifest *v1.DBExportManifest) int64 {
	var totalSize int64
	for _, file := range manifest.GetFiles() {
		totalSize += file.GetEncodedSize()
	}
	return totalSize
}
