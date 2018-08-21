package mapping

import (
	"github.com/blevesearch/bleve/mapping"
	"github.com/stackrox/rox/central/secret/search/options"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// DocumentMap provides the document mapping for the indexer to use when indexing secrets and relationships.
var DocumentMap = func() *mapping.DocumentMapping {
	ret := blevesearch.FieldsToDocumentMapping(options.Map)
	blevesearch.AddDefaultTypeField(ret)
	return ret
}()
