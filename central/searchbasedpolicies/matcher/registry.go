package matcher

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/central/searchbasedpolicies/fields"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// Registry is the registry of top-level query builders.
// Policy evaluation is effectively a conjunction of these.
type Registry []searchbasedpolicies.PolicyQueryBuilder

// NewRegistry returns a new registry of the builders with the given underlying datastore for fetching process indicators.
func NewRegistry(processIndicators processDataStore.DataStore,
	k8sRoles roleDataStore.DataStore,
	k8sBindings bindingDataStore.DataStore,
	serviceAccounts serviceAccountDataStore.DataStore,
	clusters clusterDataStore.DataStore) Registry {
	reg := []searchbasedpolicies.PolicyQueryBuilder{
		fields.ImageNameQueryBuilder,
		fields.ImageAgeQueryBuilder,
		builders.NewDockerFileLineQueryBuilder(),
		builders.CVSSQueryBuilder{},
		builders.CVEQueryBuilder{},
		fields.ComponentQueryBuilder,
		fields.DisallowedAnnotationQueryBuilder,
		fields.ScanAgeQueryBuilder,
		builders.ScanExistsQueryBuilder{},
		builders.EnvQueryBuilder{},
		fields.VolumeQueryBuilder,
		fields.PortQueryBuilder,
		fields.RequiredLabelQueryBuilder,
		fields.RequiredAnnotationQueryBuilder,
		builders.PrivilegedQueryBuilder{},
		builders.NewAddCapQueryBuilder(),
		builders.NewDropCapQueryBuilder(),
		fields.ResourcePolicy,
		builders.ProcessQueryBuilder{
			ProcessGetter: processIndicators,
		},
		builders.ReadOnlyRootFSQueryBuilder{},
		builders.PortExposureQueryBuilder{},
		builders.ProcessWhitelistingBuilder{},
		builders.HostMountQueryBuilder{},
	}
	if features.K8sRBAC.Enabled() {
		reg = append(reg, builders.K8sRBACQueryBuilder{
			Clusters:        clusters,
			K8sRoles:        k8sRoles,
			K8sBindings:     k8sBindings,
			ServiceAccounts: serviceAccounts,
		})
	}
	return reg
}
