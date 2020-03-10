package fileutils

import "os"

// Exists checks if the file under the given path exists by means of `stat()`.
// If there was an error stat'ing the file which is not due to its non-existence,
// this error is returned.
func Exists(filePath string) (bool, error) {
	if _, err := os.Stat(filePath); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return false, nil
}

// AllExist check if all of the given files exist. If there is an error stat'ing any
// of the files (which is not due to its non-existence), this error is guaranteed to
// be returned, even if some other files do not exist.
func AllExist(filePaths ...string) (bool, error) {
	allExist := true
	for _, filePath := range filePaths {
		if exists, err := Exists(filePath); err != nil {
			return false, err
		} else if !exists {
			allExist = false
		}
	}
	return allExist, nil
}

// NoneExists checks if none of the given files exist. If there is an error stat'ing any
// of the files (which is not due to its non-existence), this error is guaranteed to be
// returned.
func NoneExists(filePaths ...string) (bool, error) {
	for _, filePath := range filePaths {
		if exists, err := Exists(filePath); err != nil {
			return false, err
		} else if exists {
			return false, nil
		}
	}
	return true, nil
}
