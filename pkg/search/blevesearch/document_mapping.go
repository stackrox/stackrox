package blevesearch

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve/mapping"
)

// AddDefaultTypeField does something
func AddDefaultTypeField(docMap *mapping.DocumentMapping) {
	docMap.AddFieldMappingsAt("type", mapping.NewTextFieldMapping())
}

// FieldsToDocumentMapping does something
func FieldsToDocumentMapping(fieldsMap map[string]*v1.SearchField) *mapping.DocumentMapping {
	rootDocumentMapping := newDocumentMapping()
	rootDocumentMapping.Dynamic = false
	for _, field := range fieldsMap {
		separators := strings.Split(field.FieldPath, ".")
		// go to len(separators) - 2 because the last field is simply field mapping and not a sub document
		priorDocumentMap := rootDocumentMapping
		for i := 0; i < len(separators)-1; i++ {
			separator := separators[i]
			if priorDocumentMap == nil {
				priorDocumentMap = rootDocumentMapping
			}
			currentDocMap, ok := priorDocumentMap.Properties[separator]
			if !ok {
				currentDocMap = newDocumentMapping()
			}
			priorDocumentMap.AddSubDocumentMapping(separator, currentDocMap)
			priorDocumentMap = currentDocMap
		}
		priorDocumentMap.AddFieldMappingsAt(separators[len(separators)-1], searchFieldToMapping(field))
	}
	return rootDocumentMapping
}

func newDocumentMapping() *mapping.DocumentMapping {
	docMap := mapping.NewDocumentMapping()
	docMap.Dynamic = false
	return docMap
}

func searchFieldToMapping(sf *v1.SearchField) *mapping.FieldMapping {
	switch sf.Type {
	case v1.SearchDataType_SEARCH_STRING:
		return setFieldMappingDefaults(mapping.NewTextFieldMapping(), sf.GetStore())
	case v1.SearchDataType_SEARCH_BOOL:
		return setFieldMappingDefaults(mapping.NewBooleanFieldMapping(), sf.GetStore())
	case v1.SearchDataType_SEARCH_NUMERIC, v1.SearchDataType_SEARCH_ENFORCEMENT, v1.SearchDataType_SEARCH_SEVERITY:
		return setFieldMappingDefaults(mapping.NewNumericFieldMapping(), sf.GetStore())
	default:
		return setFieldMappingDefaults(mapping.NewTextFieldMapping(), sf.GetStore())
	}
}

func setFieldMappingDefaults(m *mapping.FieldMapping, store bool) *mapping.FieldMapping {
	// Allows for string query
	m.IncludeInAll = true
	// We don't use phrase queries so this term ordering is not important to us
	m.IncludeTermVectors = false
	// This allows us to retrieve the value out of the index (e.g. filtering images by cluster using image shas retrieved from a deployments query)
	m.Store = store
	// DocValues are used for sorting the values, which we don't do
	m.DocValues = false
	return m
}
