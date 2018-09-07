package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/stackrox/rox/generated/api/v1"
)

type fileOwnershipCheck struct {
	Name        string
	Description string
	User        string
	Group       string
	File        string
}

func (f *fileOwnershipCheck) Definition() Definition {
	return Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        f.Name,
			Description: f.Description,
		},
	}
}

func compareFileOwnership(file string, expectedUser string, expectedGroup string) (result v1.CheckResult) {
	info, err := os.Stat(ContainerPath(file))
	if os.IsNotExist(err) {
		Note(&result)
		AddNotef(&result, "Test may not be applicable because %v does not exist", file)
		return
	} else if err != nil {
		Warn(&result)
		AddNotef(&result, "Error getting file info for '%v': %+v", file, err.Error())
		return
	}

	gid := info.Sys().(*syscall.Stat_t).Gid
	if expectedGroup != "" {
		fileGroup, err := user.LookupGroup(expectedGroup)
		if err != nil {
			Warn(&result)
			AddNotef(&result, "Failed to lookup file Group '%v'", expectedGroup)
			return
		}

		if fileGroup.Gid != strconv.Itoa(int(gid)) {
			Warn(&result)
			AddNotef(&result, "Group did not match expected. expected: '%v'. actual: '%v'", fileGroup.Gid, strconv.Itoa(int(gid)))
			return
		}
	}
	uid := info.Sys().(*syscall.Stat_t).Uid
	if expectedUser != "" {
		fileUser, err := user.Lookup(expectedUser)
		if err != nil {
			Warn(&result)
			AddNotef(&result, "Failed to lookup user '%v'", expectedUser)
			return
		}
		if fileUser.Uid != strconv.Itoa(int(uid)) {
			Warn(&result)
			AddNotef(&result, "User did not match expected. expected: '%v'. actual: '%v'", fileUser.Uid, strconv.Itoa(int(gid)))
			return
		}
	}
	Pass(&result)
	return
}

func (f *fileOwnershipCheck) Run() (result v1.CheckResult) {
	if f.File == "" {
		Note(&result)
		AddNotes(&result, "Test is not applicable. File is not defined")
		return
	}
	result = compareFileOwnership(f.File, f.User, f.Group)
	return
}

// NewOwnershipCheck takes a file and verifies the user and group are as expected
func NewOwnershipCheck(name, description, file, user, group string) Check {
	return &fileOwnershipCheck{
		Name:        name,
		Description: description,
		File:        file,
		User:        user,
		Group:       group,
	}
}

type systemdOwnershipCheck struct {
	Name        string
	Description string
	Service     string
	User        string
	Group       string
}

// NewSystemdOwnershipCheck takes the systemd service and tries to find the service file to check its ownership
func NewSystemdOwnershipCheck(name, description, service, user, group string) Check {
	return &systemdOwnershipCheck{
		Name:        name,
		Description: description,
		Service:     service,
		User:        user,
		Group:       group,
	}
}

func (s *systemdOwnershipCheck) Definition() Definition {
	return Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *systemdOwnershipCheck) Run() (result v1.CheckResult) {
	if s.Service == "" {
		Note(&result)
		AddNotes(&result, "Test is not applicable. Service is not defined")
		return
	}
	systemdFile, err := GetSystemdFile(s.Service)
	if err != nil {
		Note(&result)
		AddNotef(&result, "Test may not be applicable. Systemd file could not be found for service %v", s.Service)
		return
	}
	result = compareFileOwnership(systemdFile, s.User, s.Group)
	return
}

type recursiveOwnershipCheck struct {
	Name        string
	Description string
	Directory   string
	User        string
	Group       string
}

func (r *recursiveOwnershipCheck) Definition() Definition {
	return Definition{
		CheckDefinition: v1.CheckDefinition{Name: r.Name,
			Description: r.Description,
		},
	}
}

func (r *recursiveOwnershipCheck) Run() (result v1.CheckResult) {
	Pass(&result)
	if r.Directory == "" {
		Note(&result)
		AddNotes(&result, "Test is not applicable. Directory is not defined")
		return
	}
	files, err := ioutil.ReadDir(ContainerPath(r.Directory))
	if os.IsNotExist(err) {
		Note(&result)
		AddNotef(&result, "Directory '%v' does not exist. Test may not be applicable", r.Directory)
		return
	}
	if err != nil {
		Warn(&result)
		AddNotes(&result, fmt.Sprintf("Could not check permissions due to %+v", err))
		return
	}
	for _, file := range files {
		tempResult := compareFileOwnership(filepath.Join(r.Directory, file.Name()), r.User, r.Group)
		if tempResult.Result != v1.CheckStatus_PASS {
			AddNotes(&result, tempResult.Notes...)
			result.Result = tempResult.Result
		}
	}
	return
}

// NewRecursiveOwnershipCheck takes a directory and checks the ownership of the stored files within it
func NewRecursiveOwnershipCheck(name, description, directory, user, group string) Check {
	return &recursiveOwnershipCheck{
		Name:        name,
		Description: description,
		Directory:   directory,
		User:        user,
		Group:       group,
	}
}
