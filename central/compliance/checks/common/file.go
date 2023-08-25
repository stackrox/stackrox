package common

import (
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

// CheckRecursiveOwnership checks the files against the passed user and group
func CheckRecursiveOwnership(ctx framework.ComplianceContext, f *compliance.File, user, group string) {
	ownershipCheck(ctx, f, user, group)
	for _, f := range f.Children {
		CheckRecursiveOwnership(ctx, f, user, group)
	}
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
