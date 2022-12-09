package profiling

import (
	"io/ioutil"
	"os"
	"path"
	"sort"

	"github.com/pkg/errors"
)

const (
	fifoDefaultMaxFileCount = 10
)

type fifoDir struct {
	maxFileCount int
	dirPath      string
}

func (fd fifoDir) Create(fileName string) (*os.File, error) {
	err := os.MkdirAll(fd.dirPath, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) {
			return nil, errors.Wrapf(err, "creating directory: %s", fd.dirPath)
		}
	}

	entries, err := ioutil.ReadDir(fd.dirPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading directory: %s", fd.dirPath)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ModTime().Before(entries[j].ModTime())
	})

	for len(entries) >= fd.maxFileCount {
		rmPath := path.Join(fd.dirPath, entries[0].Name())
		os.Remove(rmPath)
		entries = entries[1:]
	}

	filePath := path.Join(fd.dirPath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "creating file: %s", filePath)
	}

	return file, nil
}
