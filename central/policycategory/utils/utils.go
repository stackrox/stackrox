package utils

import "github.com/stackrox/rox/generated/storage"

// GetCategoryNameToIDs gets a map of category name to category id
func GetCategoryNameToIDs(categories []*storage.PolicyCategory) map[string]string {
	nameIDMap := make(map[string]string, len(categories))
	for _, c := range categories {
		nameIDMap[c.GetName()] = c.GetId()
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
