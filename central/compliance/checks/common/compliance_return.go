package common

import (
	"fmt"

	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
	"github.com/stackrox/stackrox/pkg/compliance/msgfmt"
)

// PerNodeCheck takes a function that handles a ComplianceReturn object and writes a check around that
func PerNodeCheck(f func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn)) framework.CheckFunc {
	return func(ctx framework.ComplianceContext) {
		framework.ForEachNode(ctx, func(ctx framework.ComplianceContext, node *storage.Node) {
			scrape := ctx.Data().HostScraped(node)
			if scrape == nil {
				ctx.Finalize(fmt.Errorf("no host scrape data available for node %q", node.GetName()))
				return
			}
			f(ctx, scrape)
		})
	}
}

// CommandLineFileOwnership returns a check that checks the ownership of a file that is specified by the command line
func CommandLineFileOwnership(name string, processName, flag, user, group string) framework.Check {
	md := framework.CheckMetadata{
		ID:               name,
		Scope:            pkgFramework.NodeKind,
		DataDependencies: []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := GetProcess(ret, processName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q was not running on the host", processName)
			}
			arg := GetArgForFlag(process.Args, flag)
			if arg == nil {
				framework.PassNowf(ctx, "Could not find flag %q in process %q and thus file ownership does not need to be checked", flag, processName)
			} else if arg.GetFile() == nil {
				framework.FailNowf(ctx, "File %q specified by flag %q could not be found. Please check manually.", msgfmt.FormatStrings(arg.GetValues()...), flag)
			}
			CheckRecursiveOwnership(ctx, arg.GetFile(), user, group)
		}))
}

// CommandLineFilePermissions returns a check that checks the permissions of a file that is specified by the command line
func CommandLineFilePermissions(name string, processName, flag string, perms uint32) framework.Check {
	md := framework.CheckMetadata{
		ID:               name,
		Scope:            pkgFramework.NodeKind,
		DataDependencies: []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := GetProcess(ret, processName)
			if !exists {
				framework.PassNowf(ctx, "Process %q was not running on the host", processName)
			}
			arg := GetArgForFlag(process.Args, flag)
			if arg == nil {
				framework.PassNowf(ctx, "Flag %q was not found in process %q and thus file permissions do not need to be checked", flag, processName)
			} else if arg.GetFile() == nil {
				framework.FailNowf(ctx, "File %q specified by flag %q could not be found. Please check manually.", msgfmt.FormatStrings(arg.GetValues()...), flag)
			}
			CheckRecursivePermissions(ctx, arg.GetFile(), perms)
		}))
}
