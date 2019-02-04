package common

import (
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
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

// SystemdOwnershipCheck checks the users and groups of the file
func SystemdOwnershipCheck(name, file, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the systemd file %s on each node is owned by user %q and group %q", file, user, group),
	}
	return framework.NewCheckFromFunc(md, systemdOwnershipCheckFunc(file, user, group))
}

// OwnershipCheck checks the users and groups of the file
func OwnershipCheck(name, file, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the file %s on each node is owned by user %q and group %q", file, user, group),
	}
	return framework.NewCheckFromFunc(md, ownershipCheckFunc(file, user, group, false))
}

// OptionalOwnershipCheck checks the users and groups of the file
func OptionalOwnershipCheck(name, file, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the file %s on each node (if existing) is owned by user %q and group %q", file, user, group),
	}
	return framework.NewCheckFromFunc(md, ownershipCheckFunc(file, user, group, true))
}

// RecursiveOwnershipCheck is a framework Check for recursively checking the ownership
func RecursiveOwnershipCheck(name, dir, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that all files under the path %s are owned by user %q and group %q", dir, user, group),
	}
	return framework.NewCheckFromFunc(md, recursiveOwnershipCheckFunc(dir, user, group))
}

// CheckRecursiveOwnership checks the files against the passed user and group
func CheckRecursiveOwnership(ctx framework.ComplianceContext, f *compliance.File, user, group string) {
	ownershipCheck(ctx, f, user, group)
	for _, f := range f.Children {
		CheckRecursiveOwnership(ctx, f, user, group)
	}
}

func recursiveOwnershipCheckFunc(path, user, group string) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[path]
		if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
		}
		CheckRecursiveOwnership(ctx, f, user, group)
	})
}

func systemdOwnershipCheckFunc(path, user, group string) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.SystemdFiles[path]
		if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
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
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions on file %s on each node are set to '%#o'", file, permissions),
	}
	return framework.NewCheckFromFunc(md, permissionCheckFunc(file, permissions, false))
}

// OptionalPermissionCheck checks the permissions of the optional file
func OptionalPermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions on file %s on each node (if existing) are set to '%#o'", file, permissions),
	}
	return framework.NewCheckFromFunc(md, permissionCheckFunc(file, permissions, true))
}

// SystemdPermissionCheck checks the permissions of the file
func SystemdPermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions on the systemd file %s on each node are set to '%#o'", file, permissions),
	}
	return framework.NewCheckFromFunc(md, systemdPermissionCheckFunc(file, permissions))
}

// RecursivePermissionCheck recursively checks the permissions of the file
func RecursivePermissionCheck(name, file string, permissions uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that the permissions of all files under the path %s on each node are set to '%#o'", file, permissions),
	}
	return framework.NewCheckFromFunc(md, recursivePermissionCheckFunc(file, permissions))
}

func systemdPermissionCheckFunc(path string, permissions uint32) framework.CheckFunc {
	return PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.SystemdFiles[path]
		if !ok {
			framework.FailNowf(ctx, "File %q could not be found in scraped data", path)
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
