package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/api/v1"
)

type sarWrapper struct {
	// Json name of this field must match what is used in secret/search/options/map
	*v1.Secret `json:"secret"`
	Type       string `json:"type"`
}

type indexerImpl struct {
	index bleve.Index
}

// SecretAndRelationship indexes a secret abd its relationships.
func (i *indexerImpl) UpsertSecret(secret *v1.Secret) error {
	// We need to wrap here because the input to .Index needs to implement .Type()
	wrapped := &sarWrapper{
		Secret: secret,
		// Type used here must match type used in globalindex/bleveindex.
		Type: v1.SearchCategory_SECRETS.String(),
	}
	return i.index.Index(secret.GetId(), wrapped)
}

func (i *indexerImpl) RemoveSecret(id string) error {
	return i.index.Delete(id)
}
