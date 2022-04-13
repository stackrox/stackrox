package blevesearch

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
)

// DocumentMappingFromOptionsMap creates a ready-to-use document mapping from the given optionsMap
func DocumentMappingFromOptionsMap(optionsMap map[search.FieldLabel]*search.Field) *mapping.DocumentMapping {
	rootDocumentMapping := newDocumentMapping(false)
	for _, field := range optionsMap {
		path := strings.Split(field.FieldPath, ".")
		addToDocumentMapping(path, field, rootDocumentMapping)
	}
	disabledSection := bleve.NewDocumentDisabledMapping()
	rootDocumentMapping.AddSubDocumentMapping("_all", disabledSection)
	// This allows us to index the type field, which is present on the wrap struct we create for every
	// searchable type. It is necessary to index this field since we store all documents in the same
	// index, so we can add a query matching the "type" field to the document type if we want to restrict
	// results to documents of that type.

	typeTextField := mapping.NewTextFieldMapping()
	typeTextField.Store = false
	typeTextField.DocValues = false
	typeTextField.IncludeInAll = false
	rootDocumentMapping.AddFieldMappingsAt("type", typeTextField)
	return rootDocumentMapping
}

func addToDocumentMapping(path []string, searchField *search.Field, docMap *mapping.DocumentMapping) {
	// Base case is either no path or the leaf of the path, for which we add a field mapping.
	if len(path) < 1 {
		panic("path is empty, check that FieldPath is set in the search field")
	}
	if len(path) == 1 {
		switch searchField.GetType() {
		case v1.SearchDataType_SEARCH_MAP:
			keypairDocMapping := newDocumentMapping(false)
			keypairDocMapping.AddFieldMappingsAt("key", setFieldMappingDefaults(mapping.NewTextFieldMapping(), searchField))
			keypairDocMapping.AddFieldMappingsAt("value", setFieldMappingDefaults(mapping.NewTextFieldMapping(), searchField))

			labelDocMap := newDocumentMapping(false)
			labelDocMap.AddSubDocumentMapping("keypair", keypairDocMapping)

			docMap.AddSubDocumentMapping(path[0], labelDocMap)
		default:
			docMap.AddFieldMappingsAt(path[0], searchFieldToMapping(searchField))
		}
		return
	}

	// Otherwise, we need to add to a sub-document mapping, creating one if necessary.
	childDocMapping, ok := docMap.Properties[path[0]]
	if !ok {
		childDocMapping = newDocumentMapping(false)
		docMap.AddSubDocumentMapping(path[0], childDocMapping)
	}
	addToDocumentMapping(path[1:], searchField, childDocMapping)
}

func newDocumentMapping(dynamic bool) *mapping.DocumentMapping {
	docMap := mapping.NewDocumentMapping()
	docMap.Dynamic = dynamic
	return docMap
}

func searchFieldToMapping(sf *search.Field) *mapping.FieldMapping {
	switch sf.GetType() {
	case v1.SearchDataType_SEARCH_STRING:
		return setFieldMappingDefaults(mapping.NewTextFieldMapping(), sf)
	case v1.SearchDataType_SEARCH_BOOL:
		return setFieldMappingDefaults(mapping.NewBooleanFieldMapping(), sf)
	case v1.SearchDataType_SEARCH_NUMERIC, v1.SearchDataType_SEARCH_ENUM, v1.SearchDataType_SEARCH_DATETIME:
		return setFieldMappingDefaults(mapping.NewNumericFieldMapping(), sf)
	default:
		panic(fmt.Errorf("Search Field '%s' is not handled in the mapping", sf.GetType()))
	}
}

func setFieldMappingDefaults(m *mapping.FieldMapping, searchField *search.Field) *mapping.FieldMapping {
	// Allows for string query
	m.IncludeInAll = false
	m.IncludeTermVectors = true
	// This allows us to retrieve the value out of the index (e.g. filtering images by cluster using image shas retrieved from a deployments query)
	m.Store = searchField.GetStore()
	// DocValues are used for sorting the values, which we don't do
	m.DocValues = false
	m.Analyzer = searchField.GetAnalyzer()
	return m
}
