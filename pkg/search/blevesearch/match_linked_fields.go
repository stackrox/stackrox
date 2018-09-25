package blevesearch

import (
	"fmt"
	"sort"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search/blevesearch/validpositions"
)

func treeForField(locationsMap search.FieldTermLocationMap, fieldPath string) *validpositions.Tree {
	t := validpositions.NewTree()
	for _, locations := range locationsMap[fieldPath] {
		for _, location := range locations {
			t.Add(location.ArrayPositions)
		}
	}
	return t
}

func constructValidPositionsTree(locationsMap search.FieldTermLocationMap, fieldsAndValues []searchFieldAndValue) (validPositions *validpositions.Tree) {
	for i, fieldAndValue := range fieldsAndValues {
		t := treeForField(locationsMap, fieldAndValue.sf.GetFieldPath())
		if i == 0 {
			validPositions = t
		} else {
			validPositions.Merge(t)
		}
	}
	return
}

func fieldMatchesValidPositions(locationsMap search.FieldTermLocationMap, fieldPath string, validPositions *validpositions.Tree) bool {
	for _, locations := range locationsMap[fieldPath] {
		for _, location := range locations {
			if validPositions.Contains(location.ArrayPositions) {
				return true
			}
		}
	}
	return false
}

func allFieldsMatchValidPositions(locationsMap search.FieldTermLocationMap, fieldsAndValues []searchFieldAndValue, validPositions *validpositions.Tree) bool {
	for _, fieldAndValue := range fieldsAndValues {
		if !fieldMatchesValidPositions(locationsMap, fieldAndValue.sf.GetFieldPath(), validPositions) {
			return false
		}
	}
	return true
}

func matchAndHighlight(hit *search.DocumentMatch, fieldsAndValues []searchFieldAndValue, highlightCtx highlightContext) (matched bool) {
	// First, construct a true of all the valid array positions, by using the tree structure to merge all the
	// array positions together.
	validPositions := constructValidPositionsTree(hit.Locations, fieldsAndValues)

	// Now, query the tree and make sure all the fields have at least one array position that exists in the tree.
	allFieldsMatch := allFieldsMatchValidPositions(hit.Locations, fieldsAndValues, validPositions)
	if !allFieldsMatch {
		return false
	}
	// If there's no highlight context, we can just return now.
	if highlightCtx == nil {
		return true
	}

	matches := make(map[string][]string)
	for _, fieldAndValue := range fieldsAndValues {
		if !fieldAndValue.highlight {
			continue
		}
		matchingValues, arrayPositions := getMatchingValuesFromFields(fieldAndValue.sf.GetFieldPath(), hit, validPositions, true)
		if len(matchingValues) == 0 || len(matchingValues) != len(arrayPositions) {
			continue
		}
		// Sort so that the matching values are ordered by array position.
		sort.Slice(matchingValues, func(i, j int) bool {
			for index := 0; index < len(arrayPositions[i]); index++ {
				if arrayPositions[i][index] < arrayPositions[j][index] {
					return true
				}
			}
			return false
		})
		matches[fieldAndValue.sf.GetFieldPath()] = matchingValues
	}
	highlightCtx.AddMappings(highlightCtxIDField, []string{hit.ID}, matches)
	return true
}

func matchAllFieldsQuery(index bleve.Index, category v1.SearchCategory, fieldsAndValues []searchFieldAndValue, highlightCtx highlightContext) (query.Query, error) {
	if len(fieldsAndValues) == 0 {
		return bleve.NewMatchNoneQuery(), nil
	}
	// If there's only one field, just return a "regular" search query.
	if len(fieldsAndValues) == 1 {
		if highlightCtx != nil {
			highlightCtx.AddFieldToHighlight(fieldsAndValues[0].sf.GetFieldPath())
		}
		return matchFieldQuery(category, fieldsAndValues[0].sf.GetFieldPath(), fieldsAndValues[0].sf.GetType(), fieldsAndValues[0].value)
	}

	// If we have to match multiple fields, and check that the matches are in the corresponding positions,
	// we perform the query, and filter the results by those which have matches in corresponding positions of different
	// fields, and return a docID query for those fields.
	// See the comments on tree.Tree for details on how the array positions checks work.
	mfQs := make([]query.Query, 0, len(fieldsAndValues))
	for _, fieldAndValue := range fieldsAndValues {
		mfQ, err := matchFieldQuery(category, fieldAndValue.sf.GetFieldPath(), fieldAndValue.sf.GetType(), fieldAndValue.value)
		if err != nil {
			return nil, fmt.Errorf("computing match field query for %+v: %s", fieldAndValue, err)
		}
		mfQs = append(mfQs, mfQ)
		if fieldAndValue.highlight {
			highlightCtx.AddFieldToHighlight(fieldAndValue.sf.GetFieldPath())
		}
	}
	conjunction := bleve.NewConjunctionQuery(mfQs...)
	searchResult, err := runBleveQuery(conjunction, index, highlightCtx, true)
	if err != nil {
		return nil, fmt.Errorf("running sub query for category %s, fieldsAndValues: %+v: %s", category, fieldsAndValues, err)
	}

	var resultIDs []string
	for _, hit := range searchResult.Hits {
		if matched := matchAndHighlight(hit, fieldsAndValues, highlightCtx); matched {
			resultIDs = append(resultIDs, hit.ID)
		}
	}
	if len(resultIDs) == 0 {
		return bleve.NewMatchNoneQuery(), nil
	}
	return bleve.NewDocIDQuery(resultIDs), nil
}
