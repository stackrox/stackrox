package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/secret/index/mapping"
	"github.com/stackrox/rox/generated/api/v1"
)

type sarWrapper struct {
	// Json name of this field must match what is used in secret/search/options/map
	*v1.SecretAndRelationship `json:"secret_and_relationship"`
	Type                      string `json:"type"`
}

type indexerImpl struct {
	index bleve.Index
}

// SecretAndRelationship indexes a secret abd its relationships.
func (i *indexerImpl) SecretAndRelationship(sar *v1.SecretAndRelationship) error {
	// We need to wrap here because the input to .Index needs to implement .Type()
	wrapped := &sarWrapper{
		SecretAndRelationship: sar,
		// Type used here must match type used in globalindex/bleveindex.
		Type: mapping.IndexedType,
	}
	return i.index.Index(sar.GetSecret().GetId(), wrapped)
}
