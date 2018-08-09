package mappings

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test.
var OptionsMap = map[string]*v1.SearchField{
	search.Violation: search.NewStringField("alert.violations.message"),
	search.Stale:     search.NewBoolField("alert.stale"),

	search.Enforcement: search.NewEnforcementField("alert.policy.enforcement"),
	search.PolicyID:    search.NewStringField("alert.policy.id"),
	search.PolicyName:  search.NewStringField("alert.policy.name"),
	search.Category:    search.NewStringField("alert.policy.categories"),
	search.Severity:    search.NewSeverityField("alert.policy.severity"),

	search.DeploymentID:   search.NewStringField("alert.deployment.id"),
	search.Cluster:        search.NewStringField("alert.deployment.cluster_name"),
	search.Namespace:      search.NewStringField("alert.deployment.namespace"),
	search.LabelKey:       search.NewStringField("alert.deployment.labels.key"),
	search.LabelValue:     search.NewStringField("alert.deployment.labels.value"),
	search.DeploymentName: search.NewStringField("alert.deployment.name"),
	search.Privileged:     search.NewBoolField("alert.deployment.containers.security_context.privileged"),

	search.ImageName:     search.NewStringField("alert.deployment.containers.image.name.full_name"),
	search.ImageRegistry: search.NewStringField("alert.deployment.containers.image.name.registry"),
	search.ImageRemote:   search.NewStringField("alert.deployment.containers.image.name.remote"),
	search.ImageTag:      search.NewStringField("alert.deployment.containers.image.name.tag"),
}
