package check451

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_5_1"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"CISBenchmarks"},
			InterpretationText: interpretationText,
		},
		clusterisCompliant)
}

func clusterisCompliant(ctx framework.ComplianceContext) {
	common.CISBenchmarksSatisfied(ctx)
	common.CheckKHostOperatingSystem(ctx)
}
