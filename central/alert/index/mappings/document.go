package mappings

import (
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve/mapping"
)

// DocumentMap is the document mapping for alerts.
var DocumentMap = func() *mapping.DocumentMapping {
	alertMap := blevesearch.FieldsToDocumentMapping(OptionsMap)
	blevesearch.AddDefaultTypeField(alertMap)
	return alertMap
}()
