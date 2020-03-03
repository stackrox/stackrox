package matcher

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/builders"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/fields"
)

var log = logging.LoggerForModule()

// Registry is the registry of top-level query builders.
// Policy evaluation is effectively a conjunction of these.
type Registry []searchbasedpolicies.PolicyQueryBuilder

// NewRegistry returns a new registry of the builders with the given underlying datastore for fetching process indicators.
func NewRegistry(processIndicators searchbasedpolicies.ProcessIndicatorGetter) Registry {
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
		builders.K8sRBACQueryBuilder{},
		fields.RequiredImageLabelQueryBuilder,
		fields.DisallowedImageLabelQueryBuilder,
	}
	return reg
}
