package v2backuprestore

import (
	stdErrors "errors"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// DetermineFormat determines which of the given formats is applicable for the given manifest. The first matching format
// will be returned.
func DetermineFormat(manifest *v1.DBExportManifest, formats []*v1.DBExportFormat) (*v1.DBExportFormat, int, error) {
	if len(formats) == 0 {
		return nil, -1, errors.New("the list of supported formats is empty")
	}

	var formatErrs error
	for i, format := range formats {
		err := CheckManifest(manifest, format)
		if err == nil {
			return format, i, nil
		}
		formatErrs = stdErrors.Join(formatErrs, errors.Wrapf(err, "format %s is not applicable", format.GetFormatName()))
	}
	return nil, -1, errors.Wrap(formatErrs, "no format matched manifest")
}

// CheckManifest checks if the given manifest is valid with respect to the given format.
func CheckManifest(manifest *v1.DBExportManifest, format *v1.DBExportFormat) error {
	filesInManifest := make(map[string]*v1.DBExportManifest_File)
	for _, file := range manifest.GetFiles() {
		filesInManifest[file.GetName()] = file
	}

	for _, expectedFile := range format.GetFiles() {
		mfFile := filesInManifest[expectedFile.GetName()]
		if mfFile == nil {
			if !expectedFile.GetOptional() {
				return errors.Errorf("required file %s not found in manifest", expectedFile.GetName())
			}
			continue
		}
		delete(filesInManifest, expectedFile.GetName())
	}

	if len(filesInManifest) > 0 {
		fileNames := make([]string, 0, len(filesInManifest))
		for fileName := range filesInManifest {
			fileNames = append(fileNames, fileName)
		}
		return errors.Errorf("manifest contains files unknown to format %s: %s", format.GetFormatName(), strings.Join(fileNames, ", "))
	}

	return nil
}
