package configurationfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type fileOwnershipCheck struct {
	Name        string
	Description string
	User        string
	Group       string
	File        string
}

func (f *fileOwnershipCheck) Definition() common.Definition {
	return common.Definition{
		Name:        f.Name,
		Description: f.Description,
	}
}

func compareFileOwnership(file string, expectedUser string, expectedGroup string) (result common.TestResult) {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		result.Note()
		result.AddNotef("Test may not be applicable because %v does not exist", file)
		return
	} else if err != nil {
		result.Warn()
		result.AddNotef("Error getting file info for %v: %+v", file, err.Error())
		return
	}

	gid := info.Sys().(*syscall.Stat_t).Gid
	if expectedGroup != "" {
		fileGroup, err := user.LookupGroup(expectedGroup)
		if err != nil {
			result.Warn()
			result.AddNotef("Failed to lookup fileGroup %v", expectedGroup)
			return
		}

		if fileGroup.Gid != strconv.Itoa(int(gid)) {
			result.Warn()
			result.AddNotef("Group did not match expected. expected: %v. actual: %v", fileGroup.Gid, strconv.Itoa(int(gid)))
			return
		}
	}
	uid := info.Sys().(*syscall.Stat_t).Uid
	if expectedUser != "" {
		fileUser, err := user.Lookup(expectedUser)
		if err != nil {
			result.Warn()
			result.AddNotef("Failed to lookup user %v", expectedUser)
			return
		}
		if fileUser.Uid != strconv.Itoa(int(uid)) {
			result.Warn()
			result.AddNotef("User did not match expected. expected: %v. actual: %v", fileUser.Uid, strconv.Itoa(int(gid)))
			return
		}
	}
	result.Pass()
	return
}

func (f *fileOwnershipCheck) Run() (result common.TestResult) {
	if f.File == "" {
		result.Note()
		result.AddNotes("Test is not applicable. File is not defined")
		return
	}
	result = compareFileOwnership(f.File, f.User, f.Group)
	return
}

func newOwnershipCheck(name, description, file, user, group string) common.Benchmark {
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

func newSystemdOwnershipCheck(name, description, service, user, group string) common.Benchmark {
	return &systemdOwnershipCheck{
		Name:        name,
		Description: description,
		Service:     service,
		User:        user,
		Group:       group,
	}
}

func (s *systemdOwnershipCheck) Definition() common.Definition {
	return common.Definition{
		Name:        s.Name,
		Description: s.Description,
	}
}

func (s *systemdOwnershipCheck) Run() (result common.TestResult) {
	if s.Service == "" {
		result.Note()
		result.AddNotes("Test is not applicable. Service is not defined")
		return
	}
	systemdFile := common.GetSystemdFile(s.Service)
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

func (r *recursiveOwnershipCheck) Definition() common.Definition {
	return common.Definition{
		Name:        r.Name,
		Description: r.Description,
	}
}

func (r *recursiveOwnershipCheck) Run() (result common.TestResult) {
	result.Pass()
	if r.Directory == "" {
		result.Note()
		result.AddNotes("Test is not applicable. Directory is not defined")
		return
	}
	files, err := ioutil.ReadDir(r.Directory)
	if os.IsNotExist(err) {
		result.Note()
		result.AddNotef("Directory %v does not exist. Test may not be applicable", files)
		return
	}
	if err != nil {
		result.Warn()
		result.AddNotes(fmt.Sprintf("Could not check permissions due to %+v", err))
		return
	}
	for _, file := range files {
		tempResult := compareFileOwnership(filepath.Join(r.Directory, file.Name()), r.User, r.Group)
		if tempResult.Result != common.Pass {
			result.AddNotes(tempResult.Notes...)
			result.Result = tempResult.Result
		}
	}
	return
}

func newRecursiveOwnershipCheck(name, description, directory, user, group string) common.Benchmark {
	return &recursiveOwnershipCheck{
		Name:        name,
		Description: description,
		Directory:   directory,
		User:        user,
		Group:       group,
	}
}
