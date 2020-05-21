package kubernetes

import (
	"fmt"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/msgfmt"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

func genericKubernetesCommandlineCheck(name string, processName string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc, failOverride ...common.FailOverride) framework.Check {
	md := framework.CheckMetadata{
		ID:               name,
		Scope:            framework.NodeKind,
		DataDependencies: []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := common.GetProcess(ret, processName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q not found on host, therefore check is not applicable", processName)
			}
			values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, key)
			evalFunc(ctx, values, key, target, defaultVal, failOverride...)
		}))
}

func multipleFlagsSetCheck(name string, processName string, override common.FailOverride, keys ...string) framework.Check {
	md := framework.CheckMetadata{
		ID:               name,
		Scope:            framework.NodeKind,
		DataDependencies: []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := common.GetProcess(ret, processName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q not found on host, therefore check is not applicable", processName)
			}
			for _, k := range keys {
				values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, k)
				if len(values) == 0 {
					msg := fmt.Sprintf("%q is unset", k)
					if override == nil {
						framework.Fail(ctx, msg)
					} else {
						override(ctx, msg)
					}
				} else {
					framework.Passf(ctx, "%q is set to %s", k, msgfmt.FormatStrings(values...))
				}
			}
		}))
}
