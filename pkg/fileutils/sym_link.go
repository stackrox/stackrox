package fileutils

import (
	"os"
	"path/filepath"
)

// AtomicSymlink create symbolic link to src and overwrite tgt atomically if tgt exists.
func AtomicSymlink(src string, tgt string) error {
	tgtTemp := tgt + ".tmp"
	defer func() { _ = os.Remove(tgtTemp) }()
	if err := os.Remove(tgtTemp); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(src, tgtTemp); err != nil {
		return err
	}
	if err := os.Rename(tgtTemp, tgt); err != nil {
		return err
	}
	return nil
}

// ResolveIfSymlink resolve path if it is a symlink, otherwise return path itself.
func ResolveIfSymlink(path string) (string, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		origPath, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if filepath.IsAbs(origPath) {
			return origPath, nil
		}
		origPath = filepath.Join(filepath.Dir(path), origPath)
		return filepath.Clean(origPath), nil
	}
	return path, nil
}
