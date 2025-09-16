package crs

import "github.com/stackrox/rox/generated/storage"

func getPrettyUser(user *storage.User) string {
	if user == nil {
		return "(unknown)"
	}

	attributePrecedence := []string{"email", "name", "userid"}

	// Extract all attributes into map.
	attributes := make(map[string]string)
	for _, attr := range user.GetAttributes() {
		attributes[attr.GetKey()] = attr.GetValue()
	}

	// Return attribute value with highest precedence.
	for _, attrName := range attributePrecedence {
		if attrValue := attributes[attrName]; attrValue != "" {
			return attrValue
		}
	}

	return user.GetId()
}
