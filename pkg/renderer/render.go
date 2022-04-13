package renderer

import (
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/image"
	"github.com/stackrox/stackrox/pkg/templates"
	"github.com/stackrox/stackrox/pkg/zip"
)

// RenderFiles takes the template files from the given FileNameMap, and instantiates them with the given values. The
// results are returned as ZipFiles.
func RenderFiles(filenames map[string]string, values interface{}) ([]*zip.File, error) {
	helmImage := image.GetDefaultImage()
	var files []*zip.File
	for f, tgtName := range filenames {
		t, err := helmImage.ReadFileAndTemplate(f)
		if err != nil {
			return nil, errors.Wrapf(err, "reading template file %s", f)
		}
		d, err := templates.ExecuteToBytes(t, values)
		if err != nil {
			return nil, err
		}

		if tgtName == "" {
			tgtName = filepath.Base(f)
		}

		var flags zip.FileFlags
		if path.Ext(tgtName) == ".sh" {
			flags |= zip.Executable
		}
		files = append(files, zip.NewFile(tgtName, d, flags))
	}
	return files, nil
}
