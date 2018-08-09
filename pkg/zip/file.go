package zip

import (
	"archive/zip"
	"fmt"
	"os"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
)

// NewFile returns a File object with the given parameters
func NewFile(name, content string, exec bool) *v1.File {
	return &v1.File{
		Name:       name,
		Content:    content,
		Executable: exec,
	}
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
	_, err = f.Write([]byte(file.GetContent()))
	if err != nil {
		return fmt.Errorf("file writing: %s", err)
	}
	return nil
}
