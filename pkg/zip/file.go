package zip

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
)

// NewFile returns a File object with the given parameters
func NewFile(name string, content []byte, exec bool) *v1.File {
	return &v1.File{
		Name:       name,
		Content:    content,
		Executable: exec,
	}
}

// NewFromFile creates a v1.File from an existing file.
func NewFromFile(srcFilename, tgtFilename string, exec bool) (*v1.File, error) {
	contents, err := ioutil.ReadFile(srcFilename)
	if err != nil {
		return nil, err
	}
	return NewFile(tgtFilename, contents, exec), nil
}

// AddFile adds a file to the zip writer
func AddFile(zipW *zip.Writer, file *v1.File) error {
	hdr := &zip.FileHeader{
		Name: file.GetName(),
	}
	hdr.Modified = time.Now()
	if file.GetExecutable() {
		hdr.SetMode(os.ModePerm & 0755)
	}
	f, err := zipW.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("file creation: %s", err)
	}
	_, err = f.Write(file.GetContent())
	if err != nil {
		return fmt.Errorf("file writing: %s", err)
	}
	return nil
}
