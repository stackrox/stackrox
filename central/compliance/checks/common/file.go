package common

import (
	"fmt"
	"path/filepath"

	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

// HasPermissions checks the permissions on a file
func HasPermissions(f *compliance.File, permissionLevel uint32) bool {
	return f.GetPermissions() == permissionLevel || f.GetPermissions() < permissionLevel
}

// HasOwnershipUser checks the user owner on a file
func HasOwnershipUser(f *compliance.File, user string) bool {
	return f.GetUserName() == user
}

// HasOwnershipGroup checks the group owner on a file
func HasOwnershipGroup(f *compliance.File, group string) bool {
	return f.GetGroupName() == group
}

// OptionalSystemdOwnershipCheck checks the users and groups of the file if it exists. If it does not exist, then the check passes
func OptionalSystemdOwnershipCheck(name, file, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the systemd file %s on each node is owned by user %q and group %q", file, user, group),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, systemdOwnershipCheckFunc(file, user, group, true))
}

// SystemdOwnershipCheck checks the users and groups of the file
func SystemdOwnershipCheck(name, file, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the systemd file %s on each node is owned by user %q and group %q", file, user, group),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, systemdOwnershipCheckFunc(file, user, group, false))
}

// OwnershipCheck checks the users and groups of the file
func OwnershipCheck(name, file, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the file %s on each node is owned by user %q and group %q", file, user, group),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, ownershipCheckFunc(file, user, group, false))
}

// OptionalOwnershipCheck checks the users and groups of the file if it exists. If it does not exist, then the check passes
func OptionalOwnershipCheck(name, file, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the file %s on each node (if existing) is owned by user %q and group %q", file, user, group),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, ownershipCheckFunc(file, user, group, true))
}

// RecursiveOwnershipCheck is a framework Check for recursively checking the ownership
func RecursiveOwnershipCheck(name, dir, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that all files under the path %s are owned by user %q and group %q", dir, user, group),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, recursiveOwnershipCheckFunc(dir, user, group, false))
}

// RecursiveOwnershipCheckIfDirExists is a framework Check for recursively checking the ownership
func RecursiveOwnershipCheckIfDirExists(name, dir, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that all files under the path %s are owned by user %q and group %q", dir, user, group),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, recursiveOwnershipCheckFunc(dir, user, group, true))
}

// CheckRecursiveOwnership checks the files against the passed user and group
func CheckRecursiveOwnership(ctx framework.ComplianceContext, f *compliance.File, user, group string) {
	ownershipCheck(ctx, f, user, group)
	for _, f := range f.Children {
		CheckRecursiveOwnership(ctx, f, user, group)
	}
}

func recursiveOwnershipCheckFunc(path, user, group string, optional bool) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[path]
		if !ok && optional {
			framework.PassNowf(ctx, "File %q does not exist on host, therefore check is not applicable", path)
		} else if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
		}
		CheckRecursiveOwnership(ctx, f, user, group)
	})
}

func systemdOwnershipCheckFunc(path, user, group string, optional bool) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.SystemdFiles[path]
		if !ok {
			if optional {
				framework.PassNowf(ctx, "Service %q does not exist on host, therefore check is not applicable", path)
			} else {
				framework.FailNowf(ctx, "Service %q could not be found in scraped data", path)
			}
		}
		ownershipCheck(ctx, f, user, group)
	})
}

func ownershipCheckFunc(path, user, group string, optional bool) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[path]
		if !ok && optional {
			framework.PassNowf(ctx, "File %q does not exist on host, therefore check is not applicable", path)
		} else if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
		}
		ownershipCheck(ctx, f, user, group)
	})
}

func ownershipCheck(ctx framework.ComplianceContext, f *compliance.File, user, group string) {
	var fail bool
	if !HasOwnershipUser(f, user) {
		fail = true
		framework.Failf(ctx, "Expected user %q on file %q, but found %q", user, f.GetPath(), f.GetUserName())
	}
	if !HasOwnershipGroup(f, group) {
		fail = true
		framework.Failf(ctx, "Expected group %q on file %q, but found %q", group, f.GetPath(), f.GetGroupName())
	}
	if !fail {
		framework.Passf(ctx, "Found group %q and user %q on file %q", group, user, f.GetPath())
	}
}

// PermissionCheck checks the permissions of the file
func PermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions on file %s on each node are set to '%#o'", file, permissions),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, permissionCheckFunc(file, permissions, false))
}

// OptionalPermissionCheck checks the permissions of the optional file
func OptionalPermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions on file %s on each node (if existing) are set to '%#o'", file, permissions),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, permissionCheckFunc(file, permissions, true))
}

// OptionalSystemdPermissionCheck checks the permissions of the file
func OptionalSystemdPermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions on the systemd file %s on each node are set to '%#o'", file, permissions),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, systemdPermissionCheckFunc(file, permissions, true))
}

// SystemdPermissionCheck checks the permissions of the file
func SystemdPermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions on the systemd file %s on each node are set to '%#o'", file, permissions),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, systemdPermissionCheckFunc(file, permissions, false))
}

// RecursivePermissionCheck recursively checks the permissions of the file
func RecursivePermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions of all files under the path %s on each node are set to '%#o'", file, permissions),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, recursivePermissionCheckFunc(file, permissions))
}

func systemdPermissionCheckFunc(path string, permissions uint32, optional bool) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.SystemdFiles[path]
		if !ok {
			if optional {
				framework.PassNowf(ctx, "Service %q does not exist on host, therefore check is not applicable", path)
			} else {
				framework.FailNowf(ctx, "Service %q could not be found in scraped data", path)
			}
		}
		permissionCheck(ctx, f, permissions)
	})
}

func permissionCheck(ctx framework.ComplianceContext, f *compliance.File, permissions uint32) {
	if !HasPermissions(f, permissions) {
		framework.FailNowf(ctx, "Expected permissions '%#o' on file %q, but found '%#o'", permissions, f.GetPath(), f.GetPermissions())
	} else {
		framework.Passf(ctx, "Found permissions '%#o' on file %q", permissions, f.GetPath())
	}
}

// CheckRecursivePermissions does the actual checking of the files
func CheckRecursivePermissions(ctx framework.ComplianceContext, f *compliance.File, permissions uint32) {
	permissionCheck(ctx, f, permissions)
	for _, child := range f.Children {
		CheckRecursivePermissions(ctx, child, permissions)
	}
}

func permissionCheckFunc(path string, permissions uint32, optional bool) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[path]
		if !ok && optional {
			framework.PassNowf(ctx, "File %q does not exist on host, therefore check is not applicable", path)
		} else if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
		}
		permissionCheck(ctx, f, permissions)
	})
}

func recursivePermissionCheckFunc(path string, permissions uint32) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[path]
		if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
		}
		CheckRecursivePermissions(ctx, f, permissions)
	})
}

// RecursivePermissionCheckWithFileExtIfDirExists recursively checks the permissions of the file with given extension
func RecursivePermissionCheckWithFileExtIfDirExists(name, dir, ext string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions of files with extension %s under the path %s on each node are set to '%#o'", ext, dir, permissions),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, recursivePermissionCheckWithFileExtFunc(dir, ext, permissions, true))
}

func recursivePermissionCheckWithFileExtFunc(path, fileExtension string, permissions uint32, optional bool) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[path]
		if !ok && optional {
			framework.PassNowf(ctx, "File %q does not exist on host, therefore check is not applicable", path)
		} else if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
		}
		CheckRecursivePermissionWithFileExt(ctx, f, fileExtension, permissions)
	})
}

// CheckRecursivePermissionWithFileExt does the actual checking of the files
func CheckRecursivePermissionWithFileExt(ctx framework.ComplianceContext, f *compliance.File, fileExtension string, permissions uint32) {
	if filepath.Ext(f.GetPath()) == fileExtension {
		permissionCheck(ctx, f, permissions)
		return
	}
	for _, child := range f.Children {
		CheckRecursivePermissionWithFileExt(ctx, child, fileExtension, permissions)
	}
}
