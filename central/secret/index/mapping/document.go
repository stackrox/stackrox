package mapping

import (
	"bitbucket.org/stack-rox/apollo/central/secret/search/options"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve/mapping"
)

// IndexedType is the object type we wrap indexed secrets and relationships with.
const IndexedType = "SecretAndRelationship"

// DocumentMap provides the document mapping for the indexer to use when indexing secrets and relationships.
var DocumentMap = func() *mapping.DocumentMapping {
	ret := blevesearch.FieldsToDocumentMapping(options.Map)
	blevesearch.AddDefaultTypeField(ret)
	return ret
}()
