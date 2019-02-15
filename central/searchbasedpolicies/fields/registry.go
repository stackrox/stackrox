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
	builders.CVSSQueryBuilder{},
	builders.CVEQueryBuilder{},
	componentQueryBuilder,
	disallowedAnnotationQueryBuilder,
	scanAgeQueryBuilder,
	builders.ScanExistsQueryBuilder{},
	builders.EnvQueryBuilder{},
	volumeQueryBuilder,
	portQueryBuilder,
	requiredLabelQueryBuilder,
	requiredAnnotationQueryBuilder,
	builders.PrivilegedQueryBuilder{},
	builders.NewAddCapQueryBuilder(),
	builders.NewDropCapQueryBuilder(),
	resourcePolicy,
	builders.ProcessQueryBuilder{},
	builders.ReadOnlyRootFSQueryBuilder{},
}
