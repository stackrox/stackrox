package blevesearch

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve/mapping"
	"github.com/stackrox/rox/generated/api/v1"
)

// AddDefaultTypeField does something
func AddDefaultTypeField(docMap *mapping.DocumentMapping) {
	docMap.AddFieldMappingsAt("type", mapping.NewTextFieldMapping())
}

// FieldsToDocumentMapping does something
func FieldsToDocumentMapping(fieldsMap map[string]*v1.SearchField) *mapping.DocumentMapping {
	rootDocumentMapping := newDocumentMapping(false)
	for _, field := range fieldsMap {
		path := strings.Split(field.FieldPath, ".")
		addToDocumentMapping(path, field, rootDocumentMapping)
	}
	return rootDocumentMapping
}

func addToDocumentMapping(path []string, searchField *v1.SearchField, docMap *mapping.DocumentMapping) {
	// Base case is either no path or the leaf of the path, for which we add a field mapping.
	if len(path) < 1 {
		panic("path is empty, check that FieldPath is set in the search field")
	}
	if len(path) == 1 {
		switch searchField.GetType() {
		case v1.SearchDataType_SEARCH_MAP:
			childDocMapping := newDocumentMapping(true)
			docMap.AddSubDocumentMapping(path[0], childDocMapping)
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

func searchFieldToMapping(sf *v1.SearchField) *mapping.FieldMapping {
	switch sf.Type {
	case v1.SearchDataType_SEARCH_STRING:
		return setFieldMappingDefaults(mapping.NewTextFieldMapping(), sf.GetStore())
	case v1.SearchDataType_SEARCH_BOOL:
		return setFieldMappingDefaults(mapping.NewBooleanFieldMapping(), sf.GetStore())
	case v1.SearchDataType_SEARCH_NUMERIC, v1.SearchDataType_SEARCH_ENFORCEMENT, v1.SearchDataType_SEARCH_SEVERITY, v1.SearchDataType_SEARCH_DATETIME:
		return setFieldMappingDefaults(mapping.NewNumericFieldMapping(), sf.GetStore())
	default:
		panic(fmt.Errorf("Search Field '%s' is not handled in the mapping", sf.Type))
	}
}

func setFieldMappingDefaults(m *mapping.FieldMapping, store bool) *mapping.FieldMapping {
	// Allows for string query
	m.IncludeInAll = true
	m.IncludeTermVectors = true
	// This allows us to retrieve the value out of the index (e.g. filtering images by cluster using image shas retrieved from a deployments query)
	m.Store = store
	// DocValues are used for sorting the values, which we don't do
	m.DocValues = false
	return m
}
