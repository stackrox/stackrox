package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"time"
)

// Wrapper is a wrapper around the current zip implementation so we can
// also output to files if we have direct filesystem access
type Wrapper struct {
	content map[string]*File
}

// NewWrapper creates a new zip file wrapper
func NewWrapper() *Wrapper {
	return &Wrapper{
		content: make(map[string]*File),
	}
}

// AddFiles adds files to the wrapper
func (w *Wrapper) AddFiles(files ...*File) {
	for _, f := range files {
		w.content[f.Name] = f
	}
}

// Zip returns the bytes of the zip archive or an error
func (w *Wrapper) Zip() ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	zipW := zip.NewWriter(buf)
	for _, file := range w.content {
		hdr := &zip.FileHeader{
			Name: file.Name,
		}
		hdr.Modified = time.Now()
		mode := os.FileMode(0644)
		if file.Flags&Executable != 0 {
			mode |= os.FileMode(0111)
		}
		if file.Flags&Sensitive != 0 {
			mode &= ^os.FileMode(0077)
		}
		hdr.SetMode(mode & os.ModePerm)

		f, err := zipW.CreateHeader(hdr)
		if err != nil {
			return nil, fmt.Errorf("file creation: %s", err)
		}
		_, err = f.Write(file.Content)
		if err != nil {
			return nil, fmt.Errorf("file writing: %s", err)
		}
	}
	if err := zipW.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
