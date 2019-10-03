package k8scfgwatch

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func dirContentsMTime(dir string) (time.Time, error) {
	entries, err := ioutil.ReadDir(dir)

	if err != nil {
		return time.Time{}, err
	}

	var contentsMTime time.Time
	for _, e := range entries {
		if e.Name() == "." || e.Name() == ".." || e.IsDir() {
			continue
		}

		if e.Mode()&os.ModeSymlink == os.ModeSymlink {
			resolved, err := os.Stat(filepath.Join(dir, e.Name()))
			if resolved != nil && err == nil {
				e = resolved
			}
		}

		if e.ModTime().After(contentsMTime) {
			contentsMTime = e.ModTime()
		}
	}

	return contentsMTime, nil
}
