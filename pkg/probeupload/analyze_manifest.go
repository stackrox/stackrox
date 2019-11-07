package probeupload

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// AnalyzeManifest analyzes the given manifest, checking that every contained file is valid, and returning the total
// size of all files.
func AnalyzeManifest(mf *v1.ProbeUploadManifest) (int64, error) {
	var totalSize int64
	errs := errorhelpers.NewErrorList("invalid entries in probe upload manifest")
	for _, f := range mf.GetFiles() {
		if !IsValidFilePath(f.GetName()) {
			errs.AddString(f.GetName())
		} else {
			totalSize += f.GetSize_()
		}
	}

	if err := errs.ToError(); err != nil {
		return 0, err
	}
	return totalSize, nil
}
