package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/generated/storage"
)

type filePermissionsCheck struct {
	Name            string
	Description     string
	PermissionLevel uint32
	IncludesLower   bool
	File            string
}

func (f *filePermissionsCheck) Definition() Definition {
	return Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        f.Name,
			Description: f.Description,
		},
	}
}

func compareFilePermissions(file string, permissionLevel uint32, includesLower bool) (result storage.BenchmarkCheckResult) {
	info, err := os.Stat(ContainerPath(file))
	if os.IsNotExist(err) {
		Note(&result)
		AddNotef(&result, "Test may not be applicable because '%v' does not exist", file)
		return
	} else if err != nil {
		Warn(&result)
		AddNotef(&result, "Error getting file info for '%v': %+v", file, err)
		return
	}

	if uint32(info.Mode().Perm()) == permissionLevel || (uint32(info.Mode().Perm()) < permissionLevel && includesLower) {
		Pass(&result)
		return
	}
	Warn(&result)
	AddNotef(&result, "Permission level '%#o' is higher than '%#o' on file '%v'", uint32(info.Mode().Perm()), permissionLevel, file)
	return
}

func (f *filePermissionsCheck) Run() (result storage.BenchmarkCheckResult) {
	if f.File == "" {
		Note(&result)
		AddNotes(&result, "Test is not applicable. File is not defined")
		return
	}
	result = compareFilePermissions(f.File, f.PermissionLevel, f.IncludesLower)
	return
}

// NewPermissionsCheck takes a file and verifies the permissions are as expected
func NewPermissionsCheck(name, description, file string, permissionLevel uint32, includesLower bool) Check {
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

// NewSystemdPermissionsCheck takes the systemd service and tries to find the service file to check permissions
func NewSystemdPermissionsCheck(name, description, service string, permissionLevel uint32, includesLower bool) Check {
	return &systemdPermissionsCheck{
		Name:            name,
		Description:     description,
		Service:         service,
		IncludesLower:   includesLower,
		PermissionLevel: permissionLevel,
	}
}

func (s *systemdPermissionsCheck) Definition() Definition {
	return Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{Name: s.Name,
			Description: s.Description,
		},
	}
}

func (s *systemdPermissionsCheck) Run() (result storage.BenchmarkCheckResult) {
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

func (r *recursivePermissionsCheck) Definition() Definition {
	return Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        r.Name,
			Description: r.Description,
		},
	}
}

func (r *recursivePermissionsCheck) Run() (result storage.BenchmarkCheckResult) {
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
		AddNotef(&result, "Could not check permissions due to %+v", err)
		return
	}
	for _, file := range files {
		tempResult := compareFilePermissions(filepath.Join(r.Directory, file.Name()), r.PermissionLevel, r.IncludesLower)
		if tempResult.Result != storage.BenchmarkCheckStatus_PASS {
			AddNotes(&result, tempResult.Notes...)
			result.Result = tempResult.Result
		}
	}
	return
}

// NewRecursivePermissionsCheck takes a directory and checks the permissions of the stored files within it
func NewRecursivePermissionsCheck(name, description, filepath string, permissionLevel uint32, includesLower bool) Check {
	return &recursivePermissionsCheck{
		Name:            name,
		Description:     description,
		Directory:       filepath,
		IncludesLower:   includesLower,
		PermissionLevel: permissionLevel,
	}
}
