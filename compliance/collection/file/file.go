package file

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	systemdBus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

const maxFileSize = 5 * 1024

type fileEntry struct {
	path     string
	contents bool
	recurse  bool
}

func newFileEntry(path string, contents, recurse bool) fileEntry {
	return fileEntry{
		path:     path,
		contents: contents,
		recurse:  recurse,
	}
}

var files = []fileEntry{
	// Directories without contents
	newFileEntry("/var/lib/kubelet/kubeconfig", false, false),
	newFileEntry("/srv/kubernetes/ca.crt", false, false),
	newFileEntry("/etc/kubernetes", false, true),
	newFileEntry("/etc/cni", false, true),
	newFileEntry("/opt/cni", false, true),

	newFileEntry("/usr/sbin/runc", false, true),
	newFileEntry("/usr/bin/runc", false, true),
}

var systemdUnits = []string{
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
				file, exists, err := EvaluatePath(u.Path, false, true)
				if err != nil || !exists {
					continue
				}
				systemFiles[filepath.Base(u.Path)] = file
			}
		}
	}
	return systemFiles, nil
}

func collectAuditLog() *compliance.File {
	path := "/var/log/audit/audit.log"
	file, err := os.Open(filepath.Join("/host", path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Error(err)
		return nil
	}
	defer utils.IgnoreError(file.Close)

	stat, err := file.Stat()
	if err != nil {
		log.Error(err)
		return nil
	}
	complianceFile := getFile(path, stat)

	scanner := bufio.NewScanner(file)

	dockerByteSlice := []byte("docker")
	execByteSlice := []byte("exec")
	privilegedByteSlice := []byte("privileged")
	userByte := []byte("user=root")

	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), dockerByteSlice) && bytes.Contains(scanner.Bytes(), execByteSlice) {
			if bytes.Contains(scanner.Bytes(), privilegedByteSlice) || bytes.Contains(scanner.Bytes(), userByte) {
				complianceFile.Content = append(complianceFile.Content, scanner.Bytes()...)
				complianceFile.Content = append(complianceFile.Content, byte('\n'))
			}
		}
	}
	return complianceFile
}

// CollectFiles returns the result of data collection of the files
func CollectFiles() (map[string]*compliance.File, error) {
	allFiles := make(map[string]*compliance.File)

	for _, f := range files {
		file, exists, err := EvaluatePath(f.path, f.contents, f.recurse)
		if err != nil || !exists {
			continue
		}
		allFiles[file.GetPath()] = file
	}
	// Manual special case. We need to get "/var/log/audit/audit.log", but we should filter the data because it's large
	if auditFile := collectAuditLog(); auditFile != nil {
		allFiles[auditFile.GetPath()] = auditFile
	}
	return allFiles, nil
}

func containerPath(s string) string {
	return filepath.Join("/host", s)
}

// EvaluatePath takes in a path and returns the corresponding File, if it exists or an error
func EvaluatePath(path string, withContents, recurse bool) (*compliance.File, bool, error) {
	pathInContainer := containerPath(path)
	fi, err := os.Stat(pathInContainer)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	file := getFile(path, fi)
	if fi.IsDir() && recurse {
		files, err := os.ReadDir(pathInContainer)
		if err != nil {
			return nil, false, err
		}
		for _, f := range files {
			child, exists, err := EvaluatePath(filepath.Join(path, f.Name()), withContents, recurse)
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
		file.Content, err = os.ReadFile(pathInContainer)
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
