package bundle

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/stackrox/stackrox/pkg/set"
)

var (
	// filesBlacklist is a set of filenames that we want to ignore in all bundles.
	// Essentially, we try to make the upgrader function as if those files didn't exist.
	filesBlacklist = set.NewFrozenStringSet(
		"upgrader-serviceaccount.yaml",
	)
)

// OpenFunc encapsulates the functionality of opening a file.
type OpenFunc func() (io.ReadCloser, error)

// Contents is an abstraction for the contents of a bundle.
type Contents interface {
	ListFiles() []string
	File(fileName string) OpenFunc
}

type contentsMap map[string]OpenFunc

func (c contentsMap) ListFiles() []string {
	files := make([]string, 0, len(c))
	for f := range c {
		files = append(files, f)
	}
	return files
}

func (c contentsMap) File(fileName string) OpenFunc {
	return c[fileName]
}

func (c contentsMap) add(fileName string, openFunc OpenFunc) bool {
	if filesBlacklist.Contains(path.Base(fileName)) {
		return false
	}
	c[fileName] = openFunc
	return true
}

func buildDirContentsMapRecursive(dir, base string, m contentsMap) error {
	fileInfos, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, fi := range fileInfos {
		if fi.IsDir() {
			if err := buildDirContentsMapRecursive(filepath.Join(dir, fi.Name()), path.Join(base, fi.Name()), m); err != nil {
				return err
			}
			continue
		}

		pathToFile := filepath.Join(dir, fi.Name())
		m.add(path.Join(base, fi.Name()), func() (io.ReadCloser, error) {
			return os.Open(pathToFile)
		})
	}

	return nil
}

// ContentsFromDir retrieves a view of the contents of a directory.
func ContentsFromDir(dir string) (Contents, error) {
	cm := make(contentsMap)
	if err := buildDirContentsMapRecursive(dir, "", cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// ContentsFromZIPData parses the given reader as a ZIP file, and returns a view of its contents.
func ContentsFromZIPData(zipData io.ReaderAt, length int64) (Contents, error) {
	zipReader, err := zip.NewReader(zipData, length)
	if err != nil {
		return nil, err
	}

	contentsMap := make(contentsMap)
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}
		contentsMap.add(file.Name, file.Open)
	}

	return contentsMap, nil
}
