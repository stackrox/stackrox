package mappings

import (
	imageMappings "bitbucket.org/stack-rox/apollo/central/image/index/mappings"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve/mapping"
)

// DocumentMap provides the document mapping for the indexer to use.
var DocumentMap = func() *mapping.DocumentMapping {
	documentMap := blevesearch.FieldsToDocumentMapping(OptionsMap)
	blevesearch.AddDefaultTypeField(documentMap)

	documentMap.Properties["deployment"].Properties["containers"].AddSubDocumentMapping("image", imageMappings.DocumentMap.Properties["image"])
	return documentMap
}()
