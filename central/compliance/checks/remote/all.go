package remote

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/features"

	_ "github.com/stackrox/rox/pkg/compliance/checks" // Make sure all checks are available
)

func init() {
	if !features.ComplianceInNodes.Enabled() {
		return
	}
	framework.MustRegisterChecks(makeChecksFromRemoteFuncs()...)
}

func makeChecksFromRemoteFuncs() []framework.Check {
	var checks []framework.Check
	for _, standardChecks := range standards.NodeChecks {
		for checkName, funcAndInterpretation := range standardChecks {
			checks = append(checks, framework.NewCheckFromFunc(
				framework.CheckMetadata{
					ID:                 checkName,
					Scope:              pkgFramework.NodeKind,
					DataDependencies:   []string{"HostScraped"},
					InterpretationText: funcAndInterpretation.InterpretationText,
					RemoteCheck:        true,
				},
				nil,
			))
		}
	}
	return checks
}
