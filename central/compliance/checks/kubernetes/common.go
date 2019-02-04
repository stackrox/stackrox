package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/msgfmt"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

func genericKubernetesCommandlineCheck(name string, processName string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	md := framework.CheckMetadata{
		ID:    name,
		Scope: framework.NodeKind,
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			process, exists := common.GetProcess(ret, processName)
			if !exists {
				framework.NoteNowf(ctx, "Process %q not found on host, therefore check is not applicable", processName)
			}
			values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, key)
			evalFunc(ctx, values, key, target, defaultVal)
		}))
}

func multipleFlagsSetCheck(name string, processName string, keys ...string) framework.Check {
	md := framework.CheckMetadata{
		ID:    name,
		Scope: framework.NodeKind,
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
					framework.Failf(ctx, "%q is unset", k)
				} else {
					framework.Passf(ctx, "%q is set to %s", k, msgfmt.FormatStrings(values...))
				}
			}
		}))
}
