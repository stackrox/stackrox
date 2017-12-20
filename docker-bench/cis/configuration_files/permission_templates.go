package configurationfiles

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type filePermissionsCheck struct {
	Name            string
	Description     string
	PermissionLevel uint32
	IncludesLower   bool
	File            string
}

func (f *filePermissionsCheck) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        f.Name,
			Description: f.Description,
		},
	}
}

func compareFilePermissions(file string, permissionLevel uint32, includesLower bool) (result v1.CheckResult) {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		utils.Note(&result)
		utils.AddNotef(&result, "Test may not be applicable because '%v' does not exist", file)
		return
	} else if err != nil {
		utils.Warn(&result)
		utils.AddNotef(&result, "Error getting file info for '%v': %+v", file, err)
		return
	}

	if uint32(info.Mode().Perm()) == permissionLevel || (uint32(info.Mode().Perm()) < permissionLevel && includesLower) {
		utils.Pass(&result)
		return
	}
	utils.Warn(&result)
	utils.AddNotef(&result, "Permission level '%d' is higher than '%v' on file '%v'", uint32(info.Mode().Perm()), permissionLevel, file)
	return
}

func (f *filePermissionsCheck) Run() (result v1.CheckResult) {
	if f.File == "" {
		utils.Note(&result)
		utils.AddNotes(&result, "Test is not applicable. File is not defined")
		return
	}
	result = compareFilePermissions(f.File, f.PermissionLevel, f.IncludesLower)
	return
}

func newPermissionsCheck(name, description, file string, permissionLevel uint32, includesLower bool) utils.Check {
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

func newSystemdPermissionsCheck(name, description, service string, permissionLevel uint32, includesLower bool) utils.Check {
	return &systemdPermissionsCheck{
		Name:            name,
		Description:     description,
		Service:         service,
		IncludesLower:   includesLower,
		PermissionLevel: permissionLevel,
	}
}

func (s *systemdPermissionsCheck) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{Name: s.Name,
			Description: s.Description,
		},
	}
}

func (s *systemdPermissionsCheck) Run() (result v1.CheckResult) {
	if s.Service == "" {
		utils.Note(&result)
		utils.AddNotes(&result, "Test is not applicable. Service is not defined")
		return
	}
	systemdFile := utils.GetSystemdFile(s.Service)
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

func (r *recursivePermissionsCheck) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        r.Name,
			Description: r.Description,
		},
	}
}

func (r *recursivePermissionsCheck) Run() (result v1.CheckResult) {
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
		utils.AddNotef(&result, "Could not check permissions due to %+v", err)
		return
	}
	for _, file := range files {
		tempResult := compareFilePermissions(filepath.Join(r.Directory, file.Name()), r.PermissionLevel, r.IncludesLower)
		if tempResult.Result != v1.CheckStatus_PASS {
			utils.AddNotes(&result, tempResult.Notes...)
			result.Result = tempResult.Result
		}
	}
	return
}

func newRecursivePermissionsCheck(name, description, filepath string, permissionLevel uint32, includesLower bool) utils.Check {
	return &recursivePermissionsCheck{
		Name:            name,
		Description:     description,
		Directory:       filepath,
		IncludesLower:   includesLower,
		PermissionLevel: permissionLevel,
	}
}
