package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
)

// Registry is the registry of top-level query builders.
var Registry = []searchbasedpolicies.PolicyQueryBuilder{
	imageNameQueryBuilder,
	scanAgeQueryBuilder,
	builders.NewDockerFileLineQueryBuilder(),
}
