package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	systemdBus "github.com/coreos/go-systemd/dbus"
	"github.com/godbus/dbus"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

const maxFileSize = 5 * 1024

var filesWithContents = []string{
	"/etc/audit",
	"/etc/docker",
	"/etc/fstab",
}

var filesWithoutContents = []string{
	"/etc/kubernetes",
	"/etc/cni",
	"/opt/cni",

	// individual files
	"/var/run/docker.sock",
	"/run/docker.sock",
	"/etc/default/docker",
}

var systemdUnits = []string{
	"docker",
	"kube",
}

var fileExtensions = set.NewStringSet(
	".yaml",
	".rules",
)

var (
	log = logging.LoggerForModule()
)

func dbusConn() (*dbus.Conn, error) {
	conn, err := dbus.Dial("unix:path=/host/run/systemd/private")
	if err != nil {
		return nil, err
	}
	methods := []dbus.Auth{dbus.AuthExternal(strconv.Itoa(os.Getuid()))}

	err = conn.Auth(methods)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

// CollectSystemdFiles returns the result of data collection of the systemd files
func CollectSystemdFiles() (map[string]*compliance.File, error) {
	conn, err := systemdBus.NewConnection(dbusConn)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	systemdUnitFiles, err := conn.ListUnitFiles()
	if err != nil {
		return nil, err
	}
	systemFiles := make(map[string]*compliance.File)
	for _, u := range systemdUnitFiles {
		for _, unitSubstring := range systemdUnits {
			if strings.Contains(u.Path, unitSubstring) {
				file, exists, err := EvaluatePath(u.Path, false)
				if err != nil || !exists {
					continue
				}
				systemFiles[filepath.Base(u.Path)] = file
			}
		}
	}
	return systemFiles, nil
}

// CollectFiles returns the result of data collection of the files
func CollectFiles() (map[string]*compliance.File, error) {
	allFiles := make(map[string]*compliance.File)
	for _, f := range filesWithoutContents {
		file, exists, err := EvaluatePath(f, false)
		if err != nil || !exists {
			continue
		}
		allFiles[file.GetPath()] = file
	}
	for _, f := range filesWithContents {
		file, exists, err := EvaluatePath(f, true)
		if err != nil || !exists {
			continue
		}
		allFiles[file.GetPath()] = file
	}
	return allFiles, nil
}

func containerPath(s string) string {
	return filepath.Join("/host", s)
}

// EvaluatePath takes in a path and returns the corresponding File, if it exists or an error
func EvaluatePath(path string, withContents bool) (*compliance.File, bool, error) {
	pathInContainer := containerPath(path)
	fi, err := os.Stat(pathInContainer)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	file := getFile(path, fi)
	if fi.IsDir() {
		files, err := ioutil.ReadDir(pathInContainer)
		if err != nil {
			return nil, false, err
		}
		for _, f := range files {

			child, exists, err := EvaluatePath(filepath.Join(path, f.Name()), withContents)
			if err != nil {
				return nil, false, err
			}
			if !exists {
				continue
			}
			file.Children = append(file.Children, child)
		}
	} else if withContents {
		if fi.Size() == 0 || fi.Size() > maxFileSize || !fileExtensions.Contains(filepath.Ext(pathInContainer)) {
			return file, true, nil
		}
		var err error
		file.Content, err = ioutil.ReadFile(pathInContainer)
		if err != nil {
			return nil, false, err
		}
	}
	return file, true, nil
}

func getFile(path string, fi os.FileInfo) *compliance.File {
	gid := fi.Sys().(*syscall.Stat_t).Gid
	uid := fi.Sys().(*syscall.Stat_t).Uid
	return &compliance.File{
		Path:        path,
		User:        uid,
		UserName:    userMap[uid],
		Group:       gid,
		GroupName:   groupMap[gid],
		Permissions: uint32(fi.Mode().Perm()),
		IsDir:       fi.IsDir(),
	}
}
