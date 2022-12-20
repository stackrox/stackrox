package profiling

import (
	"io/fs"
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

	entries, err := os.ReadDir(fd.dirPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading directory: %s", fd.dirPath)
	}

	entryInfos, err := dirEntriesToFileInfo(entries)
	if err != nil {
		return nil, err
	}

	sort.Slice(entryInfos, func(i, j int) bool {
		return entryInfos[i].ModTime().Before(entryInfos[j].ModTime())
	})

	for len(entryInfos) >= fd.maxFileCount {
		rmPath := path.Join(fd.dirPath, entryInfos[0].Name())
		err := os.Remove(rmPath)
		if err != nil {
			return nil, errors.Wrapf(err, "removing file: %s", rmPath)
		}
		entryInfos = entryInfos[1:]
	}

	filePath := path.Join(fd.dirPath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "creating file: %s", filePath)
	}

	return file, nil
}

func dirEntriesToFileInfo(entries []os.DirEntry) ([]fs.FileInfo, error) {
	entryInfos := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, errors.Wrapf(err, "getting dir entry info: %s", entry.Name())
		}

		entryInfos = append(entryInfos, info)
	}

	return entryInfos, nil
}
