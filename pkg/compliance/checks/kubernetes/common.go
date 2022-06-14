package kubernetes

import (
	"fmt"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/compliance/msgfmt"
)

func genericKubernetesCommandlineCheck(processName string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc, failOverride ...common.FailOverride) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			process, exists := common.GetProcess(complianceData, processName)
			if !exists {
				return common.NoteListf("Process %q not found on host, therefore check is not applicable", processName)
			}
			values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, key)
			return evalFunc(values, key, target, defaultVal, failOverride...)
		},
	}
}

func multipleFlagsSetCheck(processName string, override common.FailOverride, keys ...string) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			process, exists := common.GetProcess(complianceData, processName)
			if !exists {
				return common.NoteListf("Process %q not found on host, therefore check is not applicable", processName)
			}
			var results []*storage.ComplianceResultValue_Evidence
			for _, k := range keys {
				values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, k)
				if len(values) == 0 {
					msg := fmt.Sprintf("%q is unset", k)
					if override == nil {
						results = append(results, common.Fail(msg))
					} else {
						results = append(results, override(msg)...)
					}
				} else {
					results = append(results, common.Passf("%q is set to %s", k, msgfmt.FormatStrings(values...)))
				}
			}
			return results
		},
	}
}
