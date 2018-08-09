package mappings

import (
	"github.com/blevesearch/bleve/mapping"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// DocumentMap is the document mapping for alerts.
var DocumentMap = func() *mapping.DocumentMapping {
	alertMap := blevesearch.FieldsToDocumentMapping(OptionsMap)
	blevesearch.AddDefaultTypeField(alertMap)
	return alertMap
}()
