package remote

import (
	"github.com/stackrox/rox/central/compliance/framework"
	_ "github.com/stackrox/rox/pkg/compliance/checks" // Make sure all checks are available
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	framework.MustRegisterChecks(makeChecksFromRemoteFuncs()...)
}

func makeChecksFromRemoteFuncs() []framework.Check {
	registry := framework.RegistrySingleton()
	var checks []framework.Check
	for _, standardChecks := range standards.NodeChecks {
		for checkName, checkAndMetadata := range standardChecks {
			if registry.Lookup(checkName) != nil {
				// Some checks are partially implemented in the nodes and partially implemented in Central.  These will already be registered.
				continue
			}
			checks = append(checks, framework.NewCheckFromFunc(
				framework.CheckMetadata{
					ID:                 checkName,
					Scope:              checkAndMetadata.Metadata.TargetKind,
					DataDependencies:   []string{"HostScraped"},
					InterpretationText: checkAndMetadata.Metadata.InterpretationText,
					RemoteCheck:        true,
				},
				nil,
			))
		}
	}
	return checks
}
