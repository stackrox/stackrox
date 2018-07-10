package index

import (
	"bitbucket.org/stack-rox/apollo/central/secret/search/options"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve/mapping"
)

// Mapping provides the document mapping for the indexer to use when indexing secrets and relationships.
var Mapping = func() *mapping.DocumentMapping {
	ret := blevesearch.FieldsToDocumentMapping(options.Map)
	blevesearch.AddDefaultTypeField(ret)
	return ret
}()
