package blevesearch

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
)

const (
	highlightCtxIDField = "HIGHLIGHTCTXIDFIELD"
)

// highlightContext maintains context on highlights as we run a Bleve Query.
// It keeps track of the fields that we want highlighted in the search results,
// along with optional "translators" for each field, which convert values in
// one field to values in another. The translators come into play when we
// make multi-level queries.
type highlightContext map[string]map[string]highlightTranslator
type highlightTranslator struct {
	valueMapping map[string][]string
}

// AddFieldToHighlight adds a field path to the highlight context. In the next search request,
// we will ask Bleve to retrieve highlights for this field.
func (h highlightContext) AddFieldToHighlight(fieldPath string) {
	h[fieldPath] = nil
}

// addTranslatedFieldIfNotExists adds a field path to the highlight context which needs to be translated
// to results in the target field using the valueMapping in the translator.
func (h highlightContext) addTranslatedFieldIfNotExists(fieldPath, targetFieldPath string) {
	if _, ok := h[fieldPath]; !ok {
		h[fieldPath] = make(map[string]highlightTranslator)
	}
	if _, ok := h[fieldPath][targetFieldPath]; !ok {
		h[fieldPath][targetFieldPath] = highlightTranslator{
			valueMapping: make(map[string][]string),
		}
	}
}

// AddMappingToFieldTranslation adds a value mapping to the translator for the given field.
// It is the caller's responsibility to make sure that AddTranslatedField was called for this fieldPath
// first; this function WILL panic if that was not done.
func (h highlightContext) addMappingToFieldTranslator(fieldPath, targetFieldPath, sourceValue string, targetValues ...string) {
	h[fieldPath][targetFieldPath].valueMapping[sourceValue] = append(h[fieldPath][targetFieldPath].valueMapping[sourceValue], targetValues...)
}

// Merge merges the other highlightContext into this one.
func (h highlightContext) Merge(other highlightContext) {
	for fieldPath, translatorMap := range other {
		if _, ok := h[fieldPath]; !ok {
			h[fieldPath] = make(map[string]highlightTranslator)
		}
		for targetFieldPath, translator := range translatorMap {
			h[fieldPath][targetFieldPath] = translator
		}
	}
}

// ApplyToBleveReq applies info from the context to the passed in bleve request.
func (h highlightContext) ApplyToBleveReq(request *bleve.SearchRequest) {
	if len(h) > 0 {
		request.IncludeLocations = true
	}
	for field := range h {
		request.Fields = append(request.Fields, field)
	}
}

// ResolveMatches resolves the fragmentMap in the context of this highlightContext, translating
// any field values that need to be translated.
func (h highlightContext) ResolveMatches(hit *search.DocumentMatch) (matchingFields map[string][]string) {
	matchingFields = make(map[string][]string)
	for fieldName, translatorMap := range h {

		var fieldValues []string
		// If it's the ID field (special case, then we know the field value -- it's just the hit's ID!)
		if fieldName == highlightCtxIDField {
			fieldValues = []string{hit.ID}
		} else {
			validPositions := treeForField(hit.Locations, fieldName)
			fieldValues, _ = getMatchingValuesFromFields(fieldName, hit, validPositions, false)
		}
		if len(fieldValues) == 0 {
			continue
		}

		// No translators means that we don't have to translate this field.
		if len(translatorMap) == 0 {
			matchingFields[fieldName] = fieldValues
			continue
		}
		// We complete the translation by mapping all values in the source field to their corresponding
		// value in the target field.
		// Example: mapping image ids to image tags.
		for targetField, translator := range translatorMap {
			for _, value := range fieldValues {
				if targetValues, ok := translator.valueMapping[value]; ok {
					matchingFields[targetField] = append(matchingFields[targetField], targetValues...)
				}
			}
		}

	}
	return
}

func (h highlightContext) AddMappings(sourceFieldPath string, sourceFieldValues []string, matches map[string][]string) {
	if h == nil {
		return
	}
	if len(matches) == 0 {
		return
	}
	for targetFieldPath := range matches {
		h.addTranslatedFieldIfNotExists(sourceFieldPath, targetFieldPath)
	}
	for _, sourceFieldValue := range sourceFieldValues {
		for targetFieldPath, targetFieldValues := range matches {
			h.addMappingToFieldTranslator(sourceFieldPath, targetFieldPath, sourceFieldValue, targetFieldValues...)
		}
	}
}
