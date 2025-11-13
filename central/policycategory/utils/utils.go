package utils

import (
	"regexp"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

var (
	uppercaseRegex = regexp.MustCompile("[A-Z]")
)

// GetCategoryNameToIDs gets a map of category name to category id
func GetCategoryNameToIDs(categories []*storage.PolicyCategory) map[string]string {
	lowerNameIDMap := make(map[string]*storage.PolicyCategory)
	nameIDMap := make(map[string]string, len(categories))
	for _, c := range categories {
		if entry, found := lowerNameIDMap[strings.ToLower(c.GetName())]; found && isMoreUppercase(entry, c) {
			lowerNameIDMap[strings.ToLower(entry.GetName())] = c
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

func isMoreUppercase(current *storage.PolicyCategory, contender *storage.PolicyCategory) bool {
	return len(uppercaseRegex.FindAllStringIndex(contender.GetName(), -1)) >= len(uppercaseRegex.FindAllStringIndex(current.GetName(), -1))
}
