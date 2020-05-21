package tar

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// FromPath writes the contents of a file path to a tar using relative file paths.
func FromPath(tarTo string, to *tar.Writer) error {
	return filepath.Walk(tarTo, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "unexpected error traversing backup file path %s", filePath)
		}
		if !info.Mode().IsRegular() || info.IsDir() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return errors.Wrap(err, "unable to create file info header")
		}

		// update the name to correctly reflect the desired destination when untaring
		relPath := strings.TrimPrefix(filePath, tarTo)
		header.Name = relPath

		// write the header
		if err := to.WriteHeader(header); err != nil {
			return errors.Wrap(err, "unable to write file info header")
		}

		// open files for taring
		f, err := os.Open(filePath)
		defer utils.IgnoreError(f.Close)
		if err != nil {
			return errors.Wrap(err, "unable to open file to output to tar")
		}
		// copy file data into tar writer
		if _, err := io.Copy(to, f); err != nil {
			return errors.Wrap(err, "unable to copy file contents to tar")
		}
		if err := f.Close(); err != nil {
			return errors.Wrapf(err, "unable to close file: %s", filePath)
		}
		return nil
	})
}

// ToPath writes the contents of a tar file to a specified path.
func ToPath(untarTo string, fileReader io.Reader) error {
	tarReader := tar.NewReader(fileReader)
	var header *tar.Header
	var err error
	for header, err = tarReader.Next(); err == nil; header, err = tarReader.Next() {
		if header == nil || header.FileInfo().IsDir() || header.Typeflag != tar.TypeReg {
			continue
		}
		path := filepath.Join(untarTo, header.Name)
		dirPath := filepath.Dir(path)

		// Create the directory if it does not already exist.
		if _, err := os.Stat(dirPath); err != nil {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return errors.Wrapf(err, "unable to make directory: %s", dirPath)
			}
		}

		// Write the file to the matching target in the scratch path.
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
		if err != nil {
			return errors.Wrapf(err, "unable to open file for write: %s", path)
		}
		if _, err := io.Copy(f, tarReader); err != nil {
			utils.IgnoreError(f.Close)
			return errors.Wrapf(err, "unable to copy to opened file: %s", path)
		}
		if err := f.Close(); err != nil {
			return errors.Wrapf(err, "unable to close file: %s", path)
		}
	}
	if err == io.EOF {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "unable to generate backup in scratch path")
	}
	// This should not happen.
	return errors.New("tar was read, but data remained")
}
