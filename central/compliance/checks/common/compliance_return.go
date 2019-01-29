package common

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
)

// PerNodeCheck takes a function that handles a ComplianceReturn object and writes a check around that
func PerNodeCheck(f func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn)) framework.CheckFunc {
	return func(ctx framework.ComplianceContext) {
		framework.ForEachNode(ctx, func(ctx framework.ComplianceContext, node *storage.Node) {
			returnData, ok := ctx.Data().HostScraped()[node.GetName()]
			if !ok {
				framework.FailNow(ctx, "Could not find scraped data")
			}
			f(ctx, returnData)
		})
	}
}

// CommandLineFileOwnership returns a check that checks the ownership of a file that is specified by the command line
func CommandLineFileOwnership(name string, processName, flag, user, group string) framework.Check {
	return framework.NewCheckFromFunc(name, framework.NodeKind, nil, PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := GetProcess(ret, processName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q was not running on the host", processName)
			}
			arg := GetArgForFlag(process.Args, flag)
			if arg == nil {
				framework.FailNowf(ctx, "Could not find flag %q in process %q", flag, processName)
			} else if arg.GetFile() == nil {
				framework.FailNowf(ctx, "Could not find file %q for flag %q", arg.Value, flag)
			}
			CheckRecursiveOwnership(ctx, arg.GetFile(), user, group)
		}))
}

// CommandLineFilePermissions returns a check that checks the permissions of a file that is specified by the command line
func CommandLineFilePermissions(name string, processName, flag string, perms uint32) framework.Check {
	return framework.NewCheckFromFunc(name, framework.NodeKind, nil, PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := GetProcess(ret, processName)
			if !exists {
				framework.PassNowf(ctx, "Process %q was not running on the host", processName)
			}
			arg := GetArgForFlag(process.Args, flag)
			if arg == nil {
				framework.PassNowf(ctx, "Flag %q was not found in process %q", flag, processName)
			} else if arg.GetFile() == nil {
				framework.FailNowf(ctx, "File %q has not been specified for flag %q", arg.Value, flag)
			}
			CheckRecursivePermissions(ctx, arg.GetFile(), perms)
		}))
}
