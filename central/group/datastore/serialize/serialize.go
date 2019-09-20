package serialize

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// KeySeparator is the separator between the group key
const KeySeparator = "\x00"

// PropsKey is the key function for GroupProperties objects
func PropsKey(props *storage.GroupProperties) string {
	return StringKey(props.GetAuthProviderId(), props.GetKey(), props.GetValue())
}

// StringKey is the key function for GroupProperties objects with direct input values.
func StringKey(authProviderID, attrKey, attrValue string) string {
	return strings.Join([]string{authProviderID, attrKey, attrValue}, KeySeparator)
}
