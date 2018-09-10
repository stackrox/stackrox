package blevesearch

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
)

// highlightContext maintains context on highlights as we run a Bleve Query.
// It keeps track of the fields that we want highlighted in the search results,
// along with optional "translators" for each field, which convert values in
// one field to values in another. The translators come into play when we
// make multi-level queries.
// (A zero-valued highlightTranslator as a value means this field doesn't need any
// translation.)
type highlightContext map[string]highlightTranslator
type highlightTranslator struct {
	targetField  string
	valueMapping map[string][]string
}

// AddFieldToHighlight adds a field path to the highlight context. In the next search request,
// we will ask Bleve to retrieve highlights for this field.
func (h highlightContext) AddFieldToHighlight(fieldPath string) {
	h[fieldPath] = highlightTranslator{}
}

// AddTranslatedFieldIfNotExists adds a field path to the highlight context which needs to be translated
// to results in the target field using the valueMapping in the translator.
func (h highlightContext) AddTranslatedFieldIfNotExists(fieldPath, targetFieldPath string) {
	if _, ok := h[fieldPath]; !ok {
		h[fieldPath] = highlightTranslator{
			targetField:  targetFieldPath,
			valueMapping: make(map[string][]string),
		}
	}
}

// AddMappingToFieldTranslation adds a value mapping to the translator for the given field.
// It is the caller's responsibility to make sure that AddTranslatedField was called for this fieldPath
// first; this function WILL panic if that was not done.
func (h highlightContext) AddMappingToFieldTranslator(fieldPath, sourceValue, targetValue string) {
	h[fieldPath].valueMapping[sourceValue] = append(h[fieldPath].valueMapping[sourceValue], targetValue)
}

// Merge merges the other highlightContext into this one.
func (h highlightContext) Merge(other highlightContext) {
	for fieldPath, translator := range other {
		h[fieldPath] = translator
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
	for fieldName, translator := range h {
		matchedIndices, matched := hit.FieldIndices[fieldName]
		if !matched {
			continue
		}

		fieldValues := getMatchingValuesFromFields(fieldName, hit.Fields, matchedIndices)
		if len(fieldValues) == 0 {
			continue
		}

		// A zero-valued translator means that we don't have to translate this field.
		if translator.targetField == "" {
			matchingFields[fieldName] = fieldValues
			continue
		}
		// We complete the translation by mapping all values in the source field to their corresponding
		// value in the target field.
		// Example: mapping image ids to image tags.
		for _, value := range fieldValues {
			if targetValues, ok := translator.valueMapping[value]; ok {
				matchingFields[translator.targetField] = append(matchingFields[translator.targetField], targetValues...)
			}
		}
	}
	return
}
