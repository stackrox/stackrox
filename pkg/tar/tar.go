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

// FromPathMap writes the contents of files to a tar. The pathMap contains the map from the relative path in tar to the path of the source file/directory.
func FromPathMap(pathMap map[string]string, to *tar.Writer) error {
	for toPath, fromPath := range pathMap {
		err := createOrAddPathWithBase(fromPath, toPath, to)
		if err != nil {
			return err
		}
	}
	return nil
}

// FromPath writes the contents of a file path to a tar using relative file paths.
func FromPath(srcPath string, to *tar.Writer) error {
	return createOrAddPathWithBase(srcPath, ".", to)
}

func createOrAddPathWithBase(fromPath string, toPath string, to *tar.Writer) error {
	// fromPath may contain symbolic links.
	resolvedFromPath, err := filepath.EvalSymlinks(fromPath)
	if err != nil {
		return errors.Wrapf(err, "cannot resolve path %s", fromPath)
	}

	return filepath.WalkDir(resolvedFromPath, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return errors.Wrapf(err, "unexpected error traversing backup file path %s", filePath)
		}

		info, err := entry.Info()
		if err != nil {
			return errors.Wrapf(err, "unexpected error getting file info when traversing backup file path %s", filePath)
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
		relPath := strings.TrimPrefix(filePath, resolvedFromPath)
		header.Name = filepath.Join(toPath, relPath)

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
	}

	return errors.Wrap(err, "unable to generate backup in scratch path")
}
