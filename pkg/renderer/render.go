package renderer

import (
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/renderer/utils"
	"github.com/stackrox/rox/pkg/zip"
)

// RenderFiles takes the template files from the given FileNameMap, and instantiates them with the given values. The
// results are returned as ZipFiles.
func RenderFiles(filenames map[string]string, values interface{}) ([]*zip.File, error) {
	var files []*zip.File
	for f, tgtName := range filenames {
		t, err := image.ReadFileAndTemplate(f, utils.BuiltinFuncs)
		if err != nil {
			return nil, errors.Wrapf(err, "reading template file %s", f)
		}
		d, err := ExecuteTemplate(t, values)
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
