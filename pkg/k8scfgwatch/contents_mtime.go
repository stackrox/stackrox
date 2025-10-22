package k8scfgwatch

import (
	"os"
	"path/filepath"
	"time"
)

func dirContentsMTime(dir string) (time.Time, error) {
	// Get the directory's own modification time first.
	// This is important for detecting when a directory becomes empty,
	// as the directory mtime changes when files are added/removed.
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return time.Time{}, err
	}
	contentsMTime := dirInfo.ModTime()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, err
	}
	for _, e := range entries {
		if e.Name() == "." || e.Name() == ".." || e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return time.Time{}, err
		}

		if e.Type()&os.ModeSymlink == os.ModeSymlink {
			resolved, err := os.Stat(filepath.Join(dir, e.Name()))
			if resolved != nil && err == nil {
				info = resolved
			}
		}

		if info.ModTime().After(contentsMTime) {
			contentsMTime = info.ModTime()
		}
	}

	return contentsMTime, nil
}
