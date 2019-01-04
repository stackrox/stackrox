package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/logging"
)

var filesToCollect = []string{
	// generalized files
	"/etc/audit",
	"/etc/docker",
	"/etc/kubernetes",

	// systemd
	"/etc/systemd/system",
	"/lib/systemd/system",
	"/usr/lib/systemd/system",

	// individual files
	"/var/run/docker.sock",
	"/run/docker.sock",
	"/etc/default/docker",
}

var log = logging.LoggerForModule()

// CollectFiles returns the result of data collection of the files
func CollectFiles() ([]*compliance.File, error) {
	var files []*compliance.File
	for _, c := range filesToCollect {
		file, exists, err := EvaluatePath(c)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		files = append(files, file)
	}
	return files, nil
}

func containerPath(s string) string {
	return filepath.Join("/host", s)
}

// EvaluatePath takes in a path and returns the corresponding File, if it exists or an error
func EvaluatePath(path string) (*compliance.File, bool, error) {
	fi, err := os.Stat(containerPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	File := getFile(path, fi)
	if fi.IsDir() {
		files, err := ioutil.ReadDir(containerPath(path))
		if err != nil {
			return nil, false, err
		}
		for _, f := range files {
			file, exists, err := EvaluatePath(filepath.Join(path, f.Name()))
			if err != nil {
				return nil, false, err
			}
			if !exists {
				continue
			}
			File.Children = append(File.Children, file)
		}
	} else {
		if fi.Size() == 0 {
			return File, true, nil
		}
		var err error
		File.Content, err = ioutil.ReadFile(containerPath(path))
		if err != nil {
			return nil, false, err
		}
	}
	return File, true, nil
}

func getFile(path string, fi os.FileInfo) *compliance.File {
	gid := fi.Sys().(*syscall.Stat_t).Gid
	uid := fi.Sys().(*syscall.Stat_t).Uid
	return &compliance.File{
		Path:        path,
		User:        uid,
		Group:       gid,
		Permissions: uint32(fi.Mode().Perm()),
		IsDir:       fi.IsDir(),
	}
}
