package fields

import (
	"github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
)

// Registry is the registry of top-level query builders.
// Policy evaluation is effectively a conjunction of these.
type Registry []searchbasedpolicies.PolicyQueryBuilder

// NewRegistry returns a new registry of the builders with the given underlying datastore for fetching process indicators.
func NewRegistry(processIndicators datastore.DataStore) Registry {
	return []searchbasedpolicies.PolicyQueryBuilder{
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
		builders.ProcessQueryBuilder{
			ProcessGetter: processIndicators,
		},
		builders.ReadOnlyRootFSQueryBuilder{},
		builders.PortExposureQueryBuilder{},
		builders.HostMountQueryBuilder{},
	}
}
