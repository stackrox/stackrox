package mappings

import (
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve/mapping"
)

// DocumentMap provides the document mapping for the indexer to use.
var DocumentMap = func() *mapping.DocumentMapping {
	policyMap := blevesearch.FieldsToDocumentMapping(OptionsMap)
	blevesearch.AddDefaultTypeField(policyMap)
	return policyMap
}()
