package mappings

import (
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve/mapping"
)

// DocumentMap provides the document mapping for the indexer to use.
var DocumentMap = func() *mapping.DocumentMapping {
	imageMap := blevesearch.FieldsToDocumentMapping(OptionsMap)
	blevesearch.AddDefaultTypeField(imageMap)
	return imageMap
}()
