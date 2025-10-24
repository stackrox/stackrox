package tar

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/fileutils"
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
	// Create a secured root directory to prevent path traversal attacks
	root, err := os.OpenRoot(untarTo)
	if err != nil {
		return errors.Wrapf(err, "unable to open root directory: %s", untarTo)
	}
	defer root.Close()

	tarReader := tar.NewReader(fileReader)
	var header *tar.Header
	for header, err = tarReader.Next(); err == nil; header, err = tarReader.Next() {
		if header == nil {
			continue
		}

		// Handle directory entries with preserved permissions
		if header.Typeflag == tar.TypeDir {
			if err := fileutils.MkdirAllInRoot(root, header.Name, header.FileInfo().Mode().Perm()); err != nil {
				return errors.Wrapf(err, "unable to create directory: %s", header.Name)
			}
			continue
		}

		// Skip non-regular files (symlinks, devices, etc.)
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Write regular file using helper - combines dir creation, file open, and copy
		rc := io.NopCloser(tarReader)
		if err := fileutils.WriteFileInRoot(root, header.Name, os.FileMode(header.Mode), rc); err != nil {
			return err
		}
	}
	if err == io.EOF {
		return nil
	}

	return errors.Wrap(err, "unable to extract tar archive")
}
