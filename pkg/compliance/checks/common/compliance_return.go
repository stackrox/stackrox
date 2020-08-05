package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/msgfmt"
)

// CommandLineFileOwnership returns a check that checks the ownership of a file that is specified by the command line
func CommandLineFileOwnership(processName, flag, user, group string) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			process, exists := GetProcess(complianceData, processName)
			if !exists {
				return NoteListf("Process %q was not running on the host", processName)
			}
			arg := GetArgForFlag(process.Args, flag)
			if arg == nil {
				return PassListf("Could not find flag %q in process %q and thus file ownership does not need to be checked", flag, processName)
			} else if arg.GetFile() == nil {
				return FailListf("File %q specified by flag %q could not be found. Please check manually.", msgfmt.FormatStrings(arg.GetValues()...), flag)
			}
			return CheckRecursiveOwnership(arg.GetFile(), user, group)
		},
	}
}

// CommandLineFilePermissions returns a check that checks the permissions of a file that is specified by the command line
func CommandLineFilePermissions(processName, flag string, perms uint32) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			process, exists := GetProcess(complianceData, processName)
			if !exists {
				return PassListf("Process %q was not running on the host", processName)
			}
			arg := GetArgForFlag(process.Args, flag)
			if arg == nil {
				return PassListf("Flag %q was not found in process %q and thus file permissions do not need to be checked", flag, processName)
			} else if arg.GetFile() == nil {
				return FailListf("File %q specified by flag %q could not be found. Please check manually.", msgfmt.FormatStrings(arg.GetValues()...), flag)
			}
			results, _ := CheckRecursivePermissions(arg.GetFile(), perms)
			return results
		},
	}
}
