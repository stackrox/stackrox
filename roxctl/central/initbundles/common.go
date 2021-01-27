package initbundles

import "github.com/stackrox/rox/generated/storage"

func getPrettyUser(user *storage.User) string {
	attributePrecedence := []string{"email", "name", "userid"}
	attributes := make(map[string]string)

	if user == nil {
		return "(unknown)"
	}

	// Extract all attributes into map.
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
