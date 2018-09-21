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
	// CVE
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
	// requiredlabel
	// requiredannotation
	builders.PrivilegedQueryBuilder{},
	// drop_caps
	// add_caps
	// container_resource
	// total_resource
}
