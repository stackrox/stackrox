package fsutils

import (
	"syscall"

	"github.com/pkg/errors"
)

// DiskStatsIn gets file system overall and used capacity in bytes from file system of path.
func DiskStatsIn(path string) (capacity uint64, used uint64, err error) {
	stat, err := getDiskStats(path)
	if err != nil {
		return 0, 0, err
	}
	capacity = stat.Blocks * uint64(stat.Bsize)
	used = (stat.Blocks - stat.Bavail) * uint64(stat.Bsize)
	return capacity, used, nil
}

func getDiskStats(path string) (*syscall.Statfs_t, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, errors.Wrapf(err, "failed to get disk stats: %s", path)
	}
	return &stat, nil
}
