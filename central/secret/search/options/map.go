package options

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Map is the map of indexed fields in secret and relationship objects.
var Map = map[search.FieldLabel]*v1.SearchField{
	search.SecretID:         search.NewField(v1.SearchCategory_SECRETS, "secret.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.SecretName:       search.NewStringField(v1.SearchCategory_SECRETS, "secret.name"),
	search.Cluster:          search.NewStringField(v1.SearchCategory_SECRETS, "secret.cluster_name"),
	search.ClusterID:        search.NewField(v1.SearchCategory_SECRETS, "secret.cluster_id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.Namespace:        search.NewStringField(v1.SearchCategory_SECRETS, "secret.namespace"),
	search.SecretType:       search.NewSecretTypeField(v1.SearchCategory_SECRETS, "secret.files.type"),
	search.SecretExpiration: search.NewTimeField(v1.SearchCategory_SECRETS, "secret.files.Metadata.Cert.end_date.seconds"),
}
