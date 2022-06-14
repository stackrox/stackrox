package renderer

import (
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/zip"
)

// LoadAssets loads the given asset files (i.e., non-templates) and returns them as ZipFiles.
func LoadAssets(fileNames FileNameMap) ([]*zip.File, error) {
	files := make([]*zip.File, 0, len(fileNames))

	for srcName, tgtName := range fileNames {
		contents, err := image.AssetFS.ReadFile(srcName)
		if err != nil {
			return nil, errors.Wrapf(err, "reading asset file %s", srcName)
		}
		if tgtName == "" {
			tgtName = path.Base(srcName)
		}
		files = append(files, newZipFile(tgtName, contents, 0))
	}

	return files, nil
}
