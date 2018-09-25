package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
)

// Registry is the registry of top-level query builders.
// Policy evaluation is effectively a conjunction of these.
var Registry = []searchbasedpolicies.PolicyQueryBuilder{
	imageNameQueryBuilder,
	imageAgeQueryBuilder,
	builders.NewDockerFileLineQueryBuilder(),
	// CVSS
	builders.CVEQueryBuilder{},
	componentQueryBuilder,
	scanAgeQueryBuilder,
	builders.ScanExistsQueryBuilder{},
	builders.EnvQueryBuilder{},
	commandQueryBuilder,
	commandArgsQueryBuilder,
	directoryQueryBuilder,
	userQueryBuilder,
	volumeQueryBuilder,
	portQueryBuilder,
	requiredLabelQueryBuilder,
	requiredAnnotationQueryBuilder,
	builders.PrivilegedQueryBuilder{},
	builders.NewAddCapQueryBuilder(),
	builders.NewDropCapQueryBuilder(),
	// container_resource
	// total_resource
}
