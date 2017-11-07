package configurationfiles

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type filePermissionsCheck struct {
	Name            string
	Description     string
	PermissionLevel uint32
	IncludesLower   bool
	File            string
}

func (f *filePermissionsCheck) Definition() common.Definition {
	return common.Definition{
		Name:        f.Name,
		Description: f.Description,
	}
}

func compareFilePermissions(file string, permissionLevel uint32, includesLower bool) (result common.TestResult) {
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

	if uint32(info.Mode().Perm()) == permissionLevel || (uint32(info.Mode().Perm()) < permissionLevel && includesLower) {
		result.Pass()
		return
	}
	result.Warn()
	result.AddNotef("Permission level %d is higher than %v on file %v", uint32(info.Mode().Perm()), permissionLevel, file)
	return
}

func (f *filePermissionsCheck) Run() (result common.TestResult) {
	if f.File == "" {
		result.Note()
		result.AddNotes("Test is not applicable. File is not defined")
		return
	}
	result = compareFilePermissions(f.File, f.PermissionLevel, f.IncludesLower)
	return
}

func newPermissionsCheck(name, description, file string, permissionLevel uint32, includesLower bool) common.Benchmark {
	return &filePermissionsCheck{
		Name:            name,
		Description:     description,
		File:            file,
		PermissionLevel: permissionLevel,
		IncludesLower:   includesLower,
	}
}

type systemdPermissionsCheck struct {
	Name            string
	Description     string
	PermissionLevel uint32
	IncludesLower   bool
	Service         string
}

func newSystemdPermissionsCheck(name, description, service string, permissionLevel uint32, includesLower bool) common.Benchmark {
	return &systemdPermissionsCheck{
		Name:            name,
		Description:     description,
		Service:         service,
		IncludesLower:   includesLower,
		PermissionLevel: permissionLevel,
	}
}

func (s *systemdPermissionsCheck) Definition() common.Definition {
	return common.Definition{
		Name:        s.Name,
		Description: s.Description,
	}
}

func (s *systemdPermissionsCheck) Run() (result common.TestResult) {
	if s.Service == "" {
		result.Note()
		result.AddNotes("Test is not applicable. Service is not defined")
		return
	}
	systemdFile := common.GetSystemdFile(s.Service)
	result = compareFilePermissions(systemdFile, s.PermissionLevel, s.IncludesLower)
	return
}

type recursivePermissionsCheck struct {
	Name            string
	Description     string
	PermissionLevel uint32
	IncludesLower   bool
	Directory       string
}

func (r *recursivePermissionsCheck) Definition() common.Definition {
	return common.Definition{
		Name:        r.Name,
		Description: r.Description,
	}
}

func (r *recursivePermissionsCheck) Run() (result common.TestResult) {
	result.Pass()
	if r.Directory == "" {
		result.Note()
		result.AddNotes("Test is not applicable. Directory is not defined")
		return
	}
	files, err := ioutil.ReadDir(r.Directory)
	if os.IsNotExist(err) {
		result.Note()
		result.AddNotef("Directory %v does not exist. Test may not be applicable", r.Directory)
		return
	}
	if err != nil {
		result.Warn()
		result.AddNotef("Could not check permissions due to %+v", err)
		return
	}
	for _, file := range files {
		tempResult := compareFilePermissions(filepath.Join(r.Directory, file.Name()), r.PermissionLevel, r.IncludesLower)
		if tempResult.Result != common.Pass {
			result.AddNotes(tempResult.Notes...)
			result.Result = tempResult.Result
		}
	}
	return
}

func newRecursivePermissionsCheck(name, description, filepath string, permissionLevel uint32, includesLower bool) common.Benchmark {
	return &recursivePermissionsCheck{
		Name:            name,
		Description:     description,
		Directory:       filepath,
		IncludesLower:   includesLower,
		PermissionLevel: permissionLevel,
	}
}
