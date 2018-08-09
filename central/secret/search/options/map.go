package options

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Map is the map of indexed fields in secret and relationship objects.
var Map = map[string]*v1.SearchField{
	SecretID:      search.NewField("secret_and_relationship.secret.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	Secret:        search.NewStringField("secret_and_relationship.secret.name"),
	Cluster:       search.NewStringField("secret_and_relationship.relationship.cluster_relationship.name"),
	ClusterID:     search.NewField("secret_and_relationship.relationship.cluster_relationship.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	ContainerID:   search.NewField("secret_and_relationship.relationship.container_relationships.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	ContainerPath: search.NewStringField("secret_and_relationship.relationship.container_relationships.path"),
	DeploymentID:  search.NewField("secret_and_relationship.relationship.deployment_relationships.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	Deployment:    search.NewStringField("secret_and_relationship.relationship.deployment_relationships.name"),
	Namespace:     search.NewStringField("secret_and_relationship.relationship.namespace_relationship.namespace"),
}
