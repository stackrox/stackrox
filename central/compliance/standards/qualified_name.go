package standards

import "strings"

// BuildQualifiedID creates an ID that uniquely identifies a control or category by prepending
// the standard ID
func BuildQualifiedID(standardID, controlOrCategoryID string) string {
	return standardID + ":" + controlOrCategoryID
}

// ChildOfStandard returns whether or not an ID is a child of the passed standard
func ChildOfStandard(id, standardID string) bool {
	return strings.HasPrefix(id, standardID+":")
}
