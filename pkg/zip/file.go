package zip

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// FileFlags store metadata about a file in a zip archive.
type FileFlags uint32

const (
	// Executable indicates that the file should be marked executable.
	Executable FileFlags = 1 << iota
	// Sensitive indicates that the file contains sensitive information and thus should not be world-readable.
	Sensitive
)

// File represents a file entry in a Zip archive.
type File struct {
	Name    string
	Content []byte
	Flags   FileFlags
}

// NewFile returns a File object with the given parameters
func NewFile(name string, content []byte, flags FileFlags) *File {
	return &File{
		Name:    name,
		Content: content,
		Flags:   flags,
	}
}

// NewFromFile creates a zip.File from an existing file.
func NewFromFile(srcFilename, tgtFilename string, flags FileFlags) (*File, error) {
	contents, err := ioutil.ReadFile(srcFilename)
	if err != nil {
		return nil, err
	}
	return NewFile(tgtFilename, contents, flags), nil
}

// AddFile adds a file to the zip writer
func AddFile(zipW *zip.Writer, file *File) error {
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
		return fmt.Errorf("file creation: %s", err)
	}
	_, err = f.Write(file.Content)
	if err != nil {
		return fmt.Errorf("file writing: %s", err)
	}
	return nil
}
