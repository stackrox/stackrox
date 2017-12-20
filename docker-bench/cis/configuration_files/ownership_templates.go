package configurationfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type fileOwnershipCheck struct {
	Name        string
	Description string
	User        string
	Group       string
	File        string
}

func (f *fileOwnershipCheck) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        f.Name,
			Description: f.Description,
		},
	}
}

func compareFileOwnership(file string, expectedUser string, expectedGroup string) (result v1.CheckResult) {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		utils.Note(&result)
		utils.AddNotef(&result, "Test may not be applicable because %v does not exist", file)
		return
	} else if err != nil {
		utils.Warn(&result)
		utils.AddNotef(&result, "Error getting file info for '%v': %+v", file, err.Error())
		return
	}

	gid := info.Sys().(*syscall.Stat_t).Gid
	if expectedGroup != "" {
		fileGroup, err := user.LookupGroup(expectedGroup)
		if err != nil {
			utils.Warn(&result)
			utils.AddNotef(&result, "Failed to lookup fileGroup '%v'", expectedGroup)
			return
		}

		if fileGroup.Gid != strconv.Itoa(int(gid)) {
			utils.Warn(&result)
			utils.AddNotef(&result, "Group did not match expected. expected: '%v'. actual: '%v'", fileGroup.Gid, strconv.Itoa(int(gid)))
			return
		}
	}
	uid := info.Sys().(*syscall.Stat_t).Uid
	if expectedUser != "" {
		fileUser, err := user.Lookup(expectedUser)
		if err != nil {
			utils.Warn(&result)
			utils.AddNotef(&result, "Failed to lookup user '%v'", expectedUser)
			return
		}
		if fileUser.Uid != strconv.Itoa(int(uid)) {
			utils.Warn(&result)
			utils.AddNotef(&result, "User did not match expected. expected: '%v'. actual: '%v'", fileUser.Uid, strconv.Itoa(int(gid)))
			return
		}
	}
	utils.Pass(&result)
	return
}

func (f *fileOwnershipCheck) Run() (result v1.CheckResult) {
	if f.File == "" {
		utils.Note(&result)
		utils.AddNotes(&result, "Test is not applicable. File is not defined")
		return
	}
	result = compareFileOwnership(f.File, f.User, f.Group)
	return
}

func newOwnershipCheck(name, description, file, user, group string) utils.Check {
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

func newSystemdOwnershipCheck(name, description, service, user, group string) utils.Check {
	return &systemdOwnershipCheck{
		Name:        name,
		Description: description,
		Service:     service,
		User:        user,
		Group:       group,
	}
}

func (s *systemdOwnershipCheck) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{Name: s.Name,
			Description: s.Description,
		},
	}
}

func (s *systemdOwnershipCheck) Run() (result v1.CheckResult) {
	if s.Service == "" {
		utils.Note(&result)
		utils.AddNotes(&result, "Test is not applicable. Service is not defined")
		return
	}
	systemdFile := utils.GetSystemdFile(s.Service)
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

func (r *recursiveOwnershipCheck) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{Name: r.Name,
			Description: r.Description,
		},
	}
}

func (r *recursiveOwnershipCheck) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	if r.Directory == "" {
		utils.Note(&result)
		utils.AddNotes(&result, "Test is not applicable. Directory is not defined")
		return
	}
	files, err := ioutil.ReadDir(r.Directory)
	if os.IsNotExist(err) {
		utils.Note(&result)
		utils.AddNotef(&result, "Directory '%v' does not exist. Test may not be applicable", r.Directory)
		return
	}
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, fmt.Sprintf("Could not check permissions due to %+v", err))
		return
	}
	for _, file := range files {
		tempResult := compareFileOwnership(filepath.Join(r.Directory, file.Name()), r.User, r.Group)
		if tempResult.Result != v1.CheckStatus_PASS {
			utils.AddNotes(&result, tempResult.Notes...)
			result.Result = tempResult.Result
		}
	}
	return
}

func newRecursiveOwnershipCheck(name, description, directory, user, group string) utils.Check {
	return &recursiveOwnershipCheck{
		Name:        name,
		Description: description,
		Directory:   directory,
		User:        user,
		Group:       group,
	}
}
