package mappings

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test.
var OptionsMap = map[string]*v1.SearchField{
	search.Violation: search.NewStringField(v1.SearchCategory_ALERTS, "alert.violations.message"),
	search.Stale:     search.NewBoolField(v1.SearchCategory_ALERTS, "alert.stale"),

	search.Enforcement: search.NewEnforcementField(v1.SearchCategory_ALERTS, "alert.policy.enforcement"),
	search.PolicyID:    search.NewStringField(v1.SearchCategory_ALERTS, "alert.policy.id"),
	search.PolicyName:  search.NewStringField(v1.SearchCategory_ALERTS, "alert.policy.name"),
	search.Category:    search.NewStringField(v1.SearchCategory_ALERTS, "alert.policy.categories"),
	search.Severity:    search.NewSeverityField(v1.SearchCategory_ALERTS, "alert.policy.severity"),

	search.DeploymentID:   search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.id"),
	search.Cluster:        search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.cluster_name"),
	search.Namespace:      search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.namespace"),
	search.LabelKey:       search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.labels.key"),
	search.LabelValue:     search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.labels.value"),
	search.DeploymentName: search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.name"),
	search.Privileged:     search.NewBoolField(v1.SearchCategory_ALERTS, "alert.deployment.containers.security_context.privileged"),

	search.ImageName:     search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.containers.image.name.full_name"),
	search.ImageRegistry: search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.containers.image.name.registry"),
	search.ImageRemote:   search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.containers.image.name.remote"),
	search.ImageTag:      search.NewStringField(v1.SearchCategory_ALERTS, "alert.deployment.containers.image.name.tag"),
}
