package renderer

import (
	"path"

	"github.com/stackrox/stackrox/pkg/zip"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// newZipFile creates a new zip file entry, automatically setting flags to executable if the file is a shell script
// (determined by the .sh extension).
func newZipFile(fileName string, contents []byte, flags zip.FileFlags) *zip.File {
	if path.Ext(fileName) == ".sh" {
		flags |= zip.Executable
	}
	return zip.NewFile(fileName, contents, flags)
}

func withPrefix(prefix string, files []*zip.File) []*zip.File {
	result := make([]*zip.File, 0, len(files))
	for _, f := range files {
		outF := *f
		outF.Name = path.Join(prefix, f.Name)
		result = append(result, &outF)
	}
	return result
}

func convertBufferedFiles(files []*loader.BufferedFile) []*zip.File {
	zipFiles := make([]*zip.File, 0, len(files))
	for _, file := range files {
		zipFiles = append(zipFiles, newZipFile(file.Name, file.Data, 0))
	}
	return zipFiles
}
