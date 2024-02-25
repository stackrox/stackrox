package probeupload

import (
	"errors"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// AnalyzeManifest analyzes the given manifest, checking that every contained file is valid, and returning the total
// size of all files.
func AnalyzeManifest(mf *v1.ProbeUploadManifest) (int64, error) {
	var totalSize int64
	var validateErrs error
	for _, f := range mf.GetFiles() {
		if !IsValidFilePath(f.GetName()) {
			validateErrs = errors.Join(validateErrs, fmt.Errorf("invalid file path %q", f.GetName()))
		} else {
			totalSize += f.GetSize_()
		}
	}

	if validateErrs != nil {
		return 0, fmt.Errorf("invalid entries in probe upload manifest: %w", validateErrs)
	}
	return totalSize, nil
}
