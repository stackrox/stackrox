package index

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
)

// IndexedType is the object type we wrap indexed secrets and relationships with.
const IndexedType = "SecretAndRelationship"

type sarWrapper struct {
	// Json name of this field must match what is used in secret/search/options/map
	*v1.SecretAndRelationship `json:"secret_and_relationship"`
	Type                      string `json:"type"`
}

// SecretAndRelationship indexes a secret abd its relationships.
func SecretAndRelationship(index bleve.Index, sar *v1.SecretAndRelationship) error {
	// We need to wrap here because the input to .Index needs to implement .Type()
	wrapped := &sarWrapper{
		SecretAndRelationship: sar,
		// Type used here must match type used in globalindex/bleveindex.
		Type: IndexedType,
	}
	return index.Index(sar.GetSecret().GetId(), wrapped)
}
