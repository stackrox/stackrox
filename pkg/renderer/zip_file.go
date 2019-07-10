package renderer

import (
	"path"

	"github.com/stackrox/rox/pkg/zip"
)

// newZipFile creates a new zip file entry, automatically setting flags to executable if the file is a shell script
// (determined by the .sh extension).
func newZipFile(fileName string, contents []byte, flags zip.FileFlags) *zip.File {
	if path.Ext(fileName) == ".sh" {
		flags |= zip.Executable
	}
	return zip.NewFile(fileName, contents, flags)
}
