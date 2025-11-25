package utils

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	titleCase = cases.Title(language.English)
)

// GetCategoryNameToIDs gets a map of category name to category id
func GetCategoryNameToIDs(categories []*storage.PolicyCategory) map[string]string {
	lowerNameIDMap := make(map[string]*storage.PolicyCategory)
	nameIDMap := make(map[string]string, len(categories))
	for _, c := range categories {
		// ROX-31406
		// If the entry exists, but the one we're looking at doesn't match the titleCase spec, we want to use this, as
		// it's likely "more uppercase" than the current candidate for this name.
		if _, found := lowerNameIDMap[strings.ToLower(c.GetName())]; found && titleCase.String(c.GetName()) != c.GetName() {
			lowerNameIDMap[strings.ToLower(c.GetName())] = c
		} else if !found {
			lowerNameIDMap[strings.ToLower(c.GetName())] = c
		}
	}
	for _, c := range categories {
		nameIDMap[c.GetName()] = lowerNameIDMap[strings.ToLower(c.GetName())].GetId()
	}
	return nameIDMap
}

// GetCategoryIDToNames gets a map of category id to category name
func GetCategoryIDToNames(categories []*storage.PolicyCategory) map[string]string {
	idNameMap := make(map[string]string, len(categories))
	for _, c := range categories {
		idNameMap[c.GetId()] = c.GetName()
	}
	return idNameMap
}
