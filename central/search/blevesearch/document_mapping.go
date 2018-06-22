package blevesearch

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve/mapping"
)

var alertObjectMap = map[string]string{
	"image":      "alert.deployment.containers.image",
	"deployment": "alert.deployment",
	"policy":     "alert.policy",
}

var deploymentObjectMap = map[string]string{
	"image": "deployment.containers.image",
}

var imageObjectMap = map[string]string{}
var policyObjectMap = map[string]string{}

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
	m.IncludeInAll = true
	m.IncludeTermVectors = false
	m.Store = store
	return m
}

func newDocumentMapping() *mapping.DocumentMapping {
	docMap := mapping.NewDocumentMapping()
	docMap.Dynamic = false
	return docMap
}

func addDefaultTypeField(docMap *mapping.DocumentMapping) {
	docMap.AddFieldMappingsAt("type", mapping.NewTextFieldMapping())
}

func fieldsToDocumentMapping(fieldsMap map[string]*v1.SearchField) *mapping.DocumentMapping {
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

var policyDocumentMap = func() *mapping.DocumentMapping {
	policyMap := fieldsToDocumentMapping(search.PolicyOptionsMap)
	addDefaultTypeField(policyMap)
	return policyMap
}()

var imageDocumentMap = func() *mapping.DocumentMapping {
	imageMap := fieldsToDocumentMapping(search.ImageOptionsMap)
	addDefaultTypeField(imageMap)
	return imageMap
}()

var deploymentDocumentMap = func() *mapping.DocumentMapping {
	documentMap := fieldsToDocumentMapping(search.DeploymentOptionsMap)
	addDefaultTypeField(documentMap)
	documentMap.Properties["deployment"].Properties["containers"].AddSubDocumentMapping("image", imageDocumentMap.Properties["image"])
	return documentMap
}()

var alertDocumentMap = func() *mapping.DocumentMapping {
	alertMap := fieldsToDocumentMapping(search.AlertOptionsMap)
	addDefaultTypeField(alertMap)

	alertMap.Properties["alert"].AddSubDocumentMapping("deployment", deploymentDocumentMap.Properties["deployment"])
	alertMap.Properties["alert"].AddSubDocumentMapping("policy", policyDocumentMap.Properties["policy"])
	return alertMap
}()
